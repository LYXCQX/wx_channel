package middleware

import (
	"context"
	"crypto/rand"
	"fmt"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// contextKey 是用于 context value 的私有类型，防止跨包 key 冲突
type contextKey string

const (
	// ContextKeyUserID 用于在 context 中存储用户 ID
	ContextKeyUserID contextKey = "user_id"
	// ContextKeyEmail 用于在 context 中存储用户邮箱
	ContextKeyEmail contextKey = "user_email"
	// ContextKeyRole 用于在 context 中存储用户角色
	ContextKeyRole contextKey = "user_role"
)

// JWTSecret 在调用 InitJWTSecretFromEnv 之前为 nil，此时所有认证请求将被拒绝
var JWTSecret []byte

// InitJWTSecretFromEnv loads JWT secret from environment.
// If not set, generates a random secret (safe but tokens won't survive restart).
func InitJWTSecretFromEnv() error {
	secret := strings.TrimSpace(os.Getenv("HUB_JWT_SECRET"))
	if secret == "" {
		// 自动生成随机密钥（每次重启后旧 token 失效）
		randomBytes := make([]byte, 32)
		if _, err := rand.Read(randomBytes); err != nil {
			return fmt.Errorf("failed to generate random JWT secret: %w", err)
		}
		JWTSecret = randomBytes
		log.Printf("⚠️  HUB_JWT_SECRET 未设置，已自动生成随机密钥（重启后所有 token 将失效）")
		return nil
	}
	if len(secret) < 32 {
		return fmt.Errorf("HUB_JWT_SECRET must be at least 32 characters")
	}
	JWTSecret = []byte(secret)
	return nil
}

type Claims struct {
	UserID uint   `json:"user_id"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// GenerateToken generates a new JWT token for a user
func GenerateToken(userID uint, email, role string) (string, error) {
	if JWTSecret == nil {
		return "", fmt.Errorf("JWT secret not initialized")
	}
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		UserID: userID,
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(JWTSecret)
}

// AuthRequired is a middleware that validates the JWT token
func AuthRequired(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if JWTSecret == nil {
			http.Error(w, "Server not configured: JWT secret missing", http.StatusInternalServerError)
			return
		}

		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			http.Error(w, "Authorization header required", http.StatusUnauthorized)
			return
		}

		bearerToken := strings.Split(authHeader, " ")
		if len(bearerToken) != 2 || bearerToken[0] != "Bearer" {
			http.Error(w, "Invalid authorization format", http.StatusUnauthorized)
			return
		}

		tokenStr := bearerToken[1]
		claims := &Claims{}

		token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
			return JWTSecret, nil
		})

		if err != nil || !token.Valid {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		// Set user info in context using typed keys
		ctx := context.WithValue(r.Context(), ContextKeyUserID, claims.UserID)
		ctx = context.WithValue(ctx, ContextKeyEmail, claims.Email)
		ctx = context.WithValue(ctx, ContextKeyRole, claims.Role)

		next(w, r.WithContext(ctx))
	}
}

// OptionalAuth allows access but sets context if token is present
func OptionalAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if JWTSecret == nil {
			next(w, r)
			return
		}
		authHeader := r.Header.Get("Authorization")
		if authHeader != "" {
			bearerToken := strings.Split(authHeader, " ")
			if len(bearerToken) == 2 && bearerToken[0] == "Bearer" {
				tokenStr := bearerToken[1]
				claims := &Claims{}
				token, err := jwt.ParseWithClaims(tokenStr, claims, func(token *jwt.Token) (interface{}, error) {
					return JWTSecret, nil
				})
				if err == nil && token.Valid {
					ctx := context.WithValue(r.Context(), ContextKeyUserID, claims.UserID)
					ctx = context.WithValue(ctx, ContextKeyEmail, claims.Email)
					ctx = context.WithValue(ctx, ContextKeyRole, claims.Role)
					r = r.WithContext(ctx)
				}
			}
		}
		next(w, r)
	}
}

// AdminRequired requires both authentication and 'admin' role.
// Must be chained after AuthRequired: AuthRequired(AdminRequired(handler))
func AdminRequired(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		role := r.Context().Value(ContextKeyRole)
		if role == nil {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}

		if role.(string) != "admin" {
			http.Error(w, "Forbidden: Admin access required", http.StatusForbidden)
			return
		}

		next(w, r)
	}
}
