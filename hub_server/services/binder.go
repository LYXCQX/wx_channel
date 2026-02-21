package services

import (
	"crypto/rand"
	"encoding/hex"
	"errors"
	"sync"
	"time"
	"wx_channel/hub_server/database"
)

// tokenEntry 存储 token 与过期时间
type tokenEntry struct {
	userID    uint
	expiresAt time.Time
}

// BindingManager manages short-lived tokens for device binding
type BindingManager struct {
	tokens map[string]tokenEntry
	mu     sync.Mutex
}

var Binder = &BindingManager{
	tokens: make(map[string]tokenEntry),
}

// GenerateToken creates a short code (e.g. 6 chars) valid for 5 minutes
func (bm *BindingManager) GenerateToken(userID uint) (string, error) {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	// 惰性清理过期 token
	bm.cleanExpiredLocked()

	// Generate 3 random bytes = 6 hex chars
	bytes := make([]byte, 3)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	token := hex.EncodeToString(bytes)

	bm.tokens[token] = tokenEntry{
		userID:    userID,
		expiresAt: time.Now().Add(5 * time.Minute),
	}

	return token, nil
}

// ValidateAndConsume returns the UserID if valid, and consumes the token
func (bm *BindingManager) ValidateAndConsume(token string) (uint, error) {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	entry, ok := bm.tokens[token]
	if !ok || time.Now().After(entry.expiresAt) {
		// 清理这个过期 token（如果存在）
		delete(bm.tokens, token)
		return 0, errors.New("invalid or expired token")
	}

	delete(bm.tokens, token) // One-time use
	return entry.userID, nil
}

// cleanExpiredLocked 清理过期 token（必须在持有锁时调用）
func (bm *BindingManager) cleanExpiredLocked() {
	now := time.Now()
	for token, entry := range bm.tokens {
		if now.After(entry.expiresAt) {
			delete(bm.tokens, token)
		}
	}
}

// ProcessBindRequest validates the token and binds the node to the user
func ProcessBindRequest(nodeID string, token string) error {
	userID, err := Binder.ValidateAndConsume(token)
	if err != nil {
		return err
	}

	// Update Node in DB
	return database.UpdateNodeBinding(nodeID, userID)
}
