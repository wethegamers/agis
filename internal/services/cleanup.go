package services

import (
	"fmt"
	"log"
	"time"
)

// CleanupService handles automatic cleanup of stopped servers
type CleanupService struct {
	db          *DatabaseService
	logger      *LoggingService
	stopChan    chan bool
	cleanupTime time.Duration
}

// CleanupConfig holds configuration for server cleanup
type CleanupConfig struct {
	FreeUserCleanupTime time.Duration // Time after which free user servers are cleaned up
	PaidUserCleanupTime time.Duration // Time after which paid user servers are cleaned up
	CheckInterval       time.Duration // How often to check for servers to clean up
}

// NewCleanupService creates a new cleanup service
func NewCleanupService(db *DatabaseService, logger *LoggingService) *CleanupService {
	return &CleanupService{
		db:          db,
		logger:      logger,
		stopChan:    make(chan bool),
		cleanupTime: 2 * time.Hour, // Default 2 hours for free users
	}
}

// Start begins the cleanup background process
func (c *CleanupService) Start() {
	log.Println("🧹 Starting server cleanup service...")

	ticker := time.NewTicker(15 * time.Minute) // Check every 15 minutes
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.performCleanup()
		case <-c.stopChan:
			log.Println("🛑 Server cleanup service stopped")
			return
		}
	}
}

// Stop stops the cleanup service
func (c *CleanupService) Stop() {
	c.stopChan <- true
}

// performCleanup checks for servers that need to be cleaned up
func (c *CleanupService) performCleanup() {
	servers, err := c.db.GetAllServers()
	if err != nil {
		log.Printf("❌ Failed to get servers for cleanup: %v", err)
		if c.logger != nil {
			c.logger.LogError("cleanup_fetch_failed", "Failed to fetch servers for cleanup", map[string]interface{}{
				"error": err.Error(),
			})
		}
		return
	}

	now := time.Now()
	cleanedCount := 0
	skippedCount := 0

	// Log cleanup scan start
	if c.logger != nil {
		c.logger.LogCleanup("cleanup_scan_start", fmt.Sprintf("Starting cleanup scan of %d servers", len(servers)), map[string]interface{}{
			"total_servers": len(servers),
			"scan_time":     now,
		})
	}

	for _, server := range servers {
		if c.shouldCleanupServer(server, now) {
			err := c.cleanupServer(server)
			if err != nil {
				log.Printf("❌ Failed to cleanup server %s: %v", server.Name, err)
				if c.logger != nil {
					c.logger.LogError("cleanup_server_failed", fmt.Sprintf("Failed to cleanup server %s", server.Name), map[string]interface{}{
						"server_id":   server.ID,
						"server_name": server.Name,
						"user_id":     server.DiscordID,
						"game_type":   server.GameType,
						"error":       err.Error(),
					})
				}
			} else {
				log.Printf("🧹 Cleaned up server %s (user: %s)", server.Name, server.DiscordID)
				cleanedCount++
				if c.logger != nil {
					c.logger.LogCleanup("server_cleaned", fmt.Sprintf("Automatically cleaned up server %s", server.Name), map[string]interface{}{
						"server_id":   server.ID,
						"server_name": server.Name,
						"user_id":     server.DiscordID,
						"game_type":   server.GameType,
						"stopped_at":  server.StoppedAt,
						"cleanup_age": time.Since(*server.StoppedAt).String(),
					})
				}
			}
		} else {
			skippedCount++
		}
	}

	// Log cleanup completion
	if c.logger != nil {
		c.logger.LogCleanup("cleanup_scan_complete", fmt.Sprintf("Cleanup scan completed: %d cleaned, %d skipped", cleanedCount, skippedCount), map[string]interface{}{
			"cleaned_count": cleanedCount,
			"skipped_count": skippedCount,
			"total_servers": len(servers),
			"scan_duration": time.Since(now).String(),
		})
	}

	if cleanedCount > 0 {
		log.Printf("🧹 Cleanup completed: %d servers removed", cleanedCount)
	}
}

// shouldCleanupServer determines if a server should be cleaned up
func (c *CleanupService) shouldCleanupServer(server *GameServer, now time.Time) bool {
	// Only cleanup stopped servers
	if server.Status != "stopped" {
		return false
	}

	// Check if StoppedAt is set
	if server.StoppedAt == nil {
		return false
	}

	// For now, use the same cleanup time for all users (2 hours)
	// In the future, this could be based on user tier
	cleanupTime := c.cleanupTime

	// Check if enough time has passed since the server was stopped
	return now.Sub(*server.StoppedAt) > cleanupTime
}

// cleanupServer removes a server and its data
func (c *CleanupService) cleanupServer(server *GameServer) error {
	// TODO: Export save file before deletion (implement this next)

	// Remove from database
	err := c.db.DeleteGameServer(server.ID)
	if err != nil {
		return fmt.Errorf("failed to delete server from database: %v", err)
	}

	// TODO: Clean up Kubernetes resources if they exist
	// TODO: Clean up any persistent volumes/storage

	return nil
}

// ScheduleCleanup schedules a server for cleanup at a specific time
func (c *CleanupService) ScheduleCleanup(serverID int, cleanupTime time.Time) error {
	return c.db.ScheduleServerCleanup(serverID, cleanupTime)
}

// GetTimeUntilCleanup returns the time remaining until a server is cleaned up
func (c *CleanupService) GetTimeUntilCleanup(server *GameServer) *time.Duration {
	if server.Status != "stopped" || server.StoppedAt == nil {
		return nil
	}

	cleanupTime := server.StoppedAt.Add(c.cleanupTime)
	now := time.Now()

	if now.After(cleanupTime) {
		return nil // Already should be cleaned up
	}

	remaining := cleanupTime.Sub(now)
	return &remaining
}
