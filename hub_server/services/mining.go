package services

import (
	"log"
	"time"
	"wx_channel/hub_server/database"
	"wx_channel/hub_server/models"
)

// StartMiningService starts the background worker to award credits for online devices
func StartMiningService() {
	// 1. Credit accumulation ticker (every 1 minute)
	ticker := time.NewTicker(1 * time.Minute)

	// 2. Cleanup ticker (every 1 hour)
	cleanupTicker := time.NewTicker(1 * time.Hour)

	// Run cleanup immediately on startup to fix current performance issues
	go func() {
		if err := database.CleanupOldTransactions(7); err != nil {
			log.Printf("Startup cleanup failed: %v", err)
		}
	}()

	go func() {
		for {
			select {
			case <-ticker.C:
				processMiningCredits()
			case <-cleanupTicker.C:
				// Keep only recent 7 days of mining logs to optimize DB size
				if err := database.CleanupOldTransactions(7); err != nil {
					log.Printf("Periodic cleanup failed: %v", err)
				}
			}
		}
	}()
}

func processMiningCredits() {
	// Find nodes active in the last 90 seconds
	// (Client sends heartbeat every 30s)
	nodes, err := database.GetActiveNodes(90 * time.Second)
	if err != nil {
		log.Printf("Error getting active nodes for mining: %v", err)
		return
	}

	if len(nodes) == 0 {
		return
	}

	// Group by UserID to minimize DB writes (batching)
	// But transaction records need to be individual or per user?
	// Let's do strictly per device for clarity in this version, or per user.
	// Map: UserID -> Count of active devices
	userDeviceCounts := make(map[uint]int)

	for _, node := range nodes {
		userDeviceCounts[node.UserID]++
	}

	for userID, count := range userDeviceCounts {
		if userID == 0 {
			continue
		}

		creditsEarned := int64(count) // 1 credit per device per minute

		// 1. Update User Credits
		if err := database.AddCredits(userID, creditsEarned); err != nil {
			log.Printf("Failed to add credits for user %d: %v", userID, err)
			continue
		}

		// 2. Record Transaction (Optional: Sampling or Aggregating?)
		// To avoid spamming the transaction table (1440 rows per device per day),
		// naturally we might want to aggregate.
		// For now, let's log it. User requested "system".
		// Maybe strict logging is required.
		tx := &models.Transaction{
			UserID:      userID,
			Amount:      creditsEarned,
			Type:        "mining",
			Description: "Online credits", // simplified
			CreatedAt:   time.Now(),
		}
		database.RecordTransaction(tx)
	}
}
