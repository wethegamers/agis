package services

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/bwmarrin/discordgo"
)

// LogLevel represents the severity of a log entry
type LogLevel string

const (
	LogLevelDebug LogLevel = "debug"
	LogLevelInfo  LogLevel = "info"
	LogLevelWarn  LogLevel = "warn"
	LogLevelError LogLevel = "error"
	LogLevelFatal LogLevel = "fatal"
)

// LogCategory represents the type/source of a log entry
type LogCategory string

const (
	LogCategoryMod      LogCategory = "mod"      // Moderation actions
	LogCategoryUser     LogCategory = "user"     // User actions (server creation, deletion, etc.)
	LogCategoryCluster  LogCategory = "cluster"  // Kubernetes cluster events
	LogCategoryPod      LogCategory = "pod"      // Pod-specific events
	LogCategoryDatabase LogCategory = "database" // Database operations
	LogCategoryBot      LogCategory = "bot"      // Bot system events
	LogCategoryCleanup  LogCategory = "cleanup"  // Cleanup service events
	LogCategoryExport   LogCategory = "export"   // Save file export events
	LogCategoryError    LogCategory = "error"    // Error events
	LogCategoryAudit    LogCategory = "audit"    // Security/audit events
)

// LogEntry represents a single log entry
type LogEntry struct {
	ID        int                    `json:"id" db:"id"`
	Timestamp time.Time              `json:"timestamp" db:"timestamp"`
	Level     LogLevel               `json:"level" db:"level"`
	Category  LogCategory            `json:"category" db:"category"`
	UserID    string                 `json:"user_id" db:"user_id"`
	Action    string                 `json:"action" db:"action"`
	Message   string                 `json:"message" db:"message"`
	Details   map[string]interface{} `json:"details" db:"details"`
	ChannelID string                 `json:"channel_id" db:"channel_id"`
}

// LogChannelConfig represents Discord channel configuration for different log types
type LogChannelConfig struct {
	ModLogs     string `json:"mod_logs"`     // Channel for moderation logs
	UserLogs    string `json:"user_logs"`    // Channel for user action logs
	ClusterLogs string `json:"cluster_logs"` // Channel for cluster/infrastructure logs
	ErrorLogs   string `json:"error_logs"`   // Channel for error logs
	AuditLogs   string `json:"audit_logs"`   // Channel for security/audit logs
	GeneralLogs string `json:"general_logs"` // Channel for general system logs
	ExportLogs  string `json:"export_logs"`  // Channel for save export logs
	CleanupLogs string `json:"cleanup_logs"` // Channel for cleanup operation logs
}

// LoggingService handles logging to Discord channels and database
type LoggingService struct {
	db      *DatabaseService
	session *discordgo.Session
	config  *LogChannelConfig
	guildID string
	enabled bool
}

// NewLoggingService creates a new logging service
func NewLoggingService(db *DatabaseService, session *discordgo.Session, guildID string) *LoggingService {
	service := &LoggingService{
		db:      db,
		session: session,
		guildID: guildID,
		enabled: session != nil, // Enable Discord logging only if session is available
		config: &LogChannelConfig{
			// Default channel IDs - these should be configured via environment variables or commands
			ModLogs:     "",
			UserLogs:    "",
			ClusterLogs: "",
			ErrorLogs:   "",
			AuditLogs:   "",
			GeneralLogs: "",
			ExportLogs:  "",
			CleanupLogs: "",
		},
	}

	// Initialize database tables
	if err := service.initLogTables(); err != nil {
		log.Printf("Failed to initialize log tables: %v", err)
	}

	return service
}

// SetChannelConfig updates the Discord channel configuration
func (l *LoggingService) SetChannelConfig(config *LogChannelConfig) {
	l.config = config
}

// SetLogChannel sets a specific log channel
func (l *LoggingService) SetLogChannel(category LogCategory, channelID string) {
	switch category {
	case LogCategoryMod:
		l.config.ModLogs = channelID
	case LogCategoryUser:
		l.config.UserLogs = channelID
	case LogCategoryCluster, LogCategoryPod:
		l.config.ClusterLogs = channelID
	case LogCategoryError:
		l.config.ErrorLogs = channelID
	case LogCategoryAudit:
		l.config.AuditLogs = channelID
	case LogCategoryExport:
		l.config.ExportLogs = channelID
	case LogCategoryCleanup:
		l.config.CleanupLogs = channelID
	default:
		l.config.GeneralLogs = channelID
	}
}

// LoadChannelConfigFromEnv loads channel configuration from environment variables
func (l *LoggingService) LoadChannelConfigFromEnv() {
	if channelID := os.Getenv("LOG_CHANNEL_MOD"); channelID != "" {
		l.config.ModLogs = channelID
	}
	if channelID := os.Getenv("LOG_CHANNEL_USER"); channelID != "" {
		l.config.UserLogs = channelID
	}
	if channelID := os.Getenv("LOG_CHANNEL_CLUSTER"); channelID != "" {
		l.config.ClusterLogs = channelID
	}
	if channelID := os.Getenv("LOG_CHANNEL_ERROR"); channelID != "" {
		l.config.ErrorLogs = channelID
	}
	if channelID := os.Getenv("LOG_CHANNEL_AUDIT"); channelID != "" {
		l.config.AuditLogs = channelID
	}
	if channelID := os.Getenv("LOG_CHANNEL_EXPORT"); channelID != "" {
		l.config.ExportLogs = channelID
	}
	if channelID := os.Getenv("LOG_CHANNEL_CLEANUP"); channelID != "" {
		l.config.CleanupLogs = channelID
	}
	if channelID := os.Getenv("LOG_CHANNEL_GENERAL"); channelID != "" {
		l.config.GeneralLogs = channelID
	}
}

// EnableDiscordLogging enables or disables Discord logging
func (l *LoggingService) EnableDiscordLogging(enabled bool) {
	l.enabled = enabled
}

// GetLogStats returns statistics about stored logs
func (l *LoggingService) GetLogStats() map[string]int {
	if l.db == nil || l.db.db == nil {
		return map[string]int{}
	}

	stats := make(map[string]int)

	// Count total logs
	row := l.db.db.QueryRow("SELECT COUNT(*) FROM system_logs")
	var total int
	if err := row.Scan(&total); err == nil {
		stats["total"] = total
	}

	// Count by category
	rows, err := l.db.db.Query("SELECT category, COUNT(*) FROM system_logs GROUP BY category")
	if err == nil {
		defer func() {
			if err := rows.Close(); err != nil {
				log.Printf("Failed to close rows: %v", err)
			}
		}()

		for rows.Next() {
			var category string
			var count int
			if err := rows.Scan(&category, &count); err == nil {
				stats[category] = count
			}
		}
	}

	return stats
}

// LogMod logs moderation actions
func (l *LoggingService) LogMod(userID, action, message string, details map[string]interface{}) {
	l.log(LogLevelInfo, LogCategoryMod, userID, action, message, details)
}

// LogUser logs user actions
func (l *LoggingService) LogUser(userID, action, message string, details map[string]interface{}) {
	l.log(LogLevelInfo, LogCategoryUser, userID, action, message, details)
}

// LogCluster logs cluster/infrastructure events
func (l *LoggingService) LogCluster(action, message string, details map[string]interface{}) {
	l.log(LogLevelInfo, LogCategoryCluster, "", action, message, details)
}

// LogPod logs pod-specific events
func (l *LoggingService) LogPod(action, message string, details map[string]interface{}) {
	l.log(LogLevelInfo, LogCategoryPod, "", action, message, details)
}

// LogDatabase logs database operations
func (l *LoggingService) LogDatabase(action, message string, details map[string]interface{}) {
	l.log(LogLevelInfo, LogCategoryDatabase, "", action, message, details)
}

// LogBot logs bot system events
func (l *LoggingService) LogBot(level LogLevel, action, message string, details map[string]interface{}) {
	l.log(level, LogCategoryBot, "", action, message, details)
}

// LogCleanup logs cleanup service events
func (l *LoggingService) LogCleanup(action, message string, details map[string]interface{}) {
	l.log(LogLevelInfo, LogCategoryCleanup, "", action, message, details)
}

// LogExport logs save file export events
func (l *LoggingService) LogExport(userID, action, message string, details map[string]interface{}) {
	l.log(LogLevelInfo, LogCategoryExport, userID, action, message, details)
}

// LogError logs error events
func (l *LoggingService) LogError(action, message string, details map[string]interface{}) {
	l.log(LogLevelError, LogCategoryError, "", action, message, details)
}

// LogAudit logs security/audit events
func (l *LoggingService) LogAudit(userID, action, message string, details map[string]interface{}) {
	l.log(LogLevelWarn, LogCategoryAudit, userID, action, message, details)
}

// log is the internal logging method
func (l *LoggingService) log(level LogLevel, category LogCategory, userID, action, message string, details map[string]interface{}) {
	entry := &LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Category:  category,
		UserID:    userID,
		Action:    action,
		Message:   message,
		Details:   details,
	}

	// Store in database
	if err := l.storeLogEntry(entry); err != nil {
		log.Printf("Failed to store log entry: %v", err)
	}

	// Send to Discord channel if enabled and configured
	if l.enabled && l.session != nil {
		channelID := l.getChannelForCategory(category)
		if channelID != "" {
			entry.ChannelID = channelID
			if err := l.sendToDiscord(entry, channelID); err != nil {
				log.Printf("Failed to send log to Discord channel %s: %v", channelID, err)
			}
		}
	}

	// Always log to console as well
	l.logToConsole(entry)
}

// getChannelForCategory returns the appropriate Discord channel for a log category
func (l *LoggingService) getChannelForCategory(category LogCategory) string {
	switch category {
	case LogCategoryMod:
		return l.config.ModLogs
	case LogCategoryUser:
		return l.config.UserLogs
	case LogCategoryCluster, LogCategoryPod:
		return l.config.ClusterLogs
	case LogCategoryError:
		return l.config.ErrorLogs
	case LogCategoryAudit:
		return l.config.AuditLogs
	case LogCategoryExport:
		return l.config.ExportLogs
	case LogCategoryCleanup:
		return l.config.CleanupLogs
	default:
		return l.config.GeneralLogs
	}
}

// sendToDiscord sends a log entry to the appropriate Discord channel
func (l *LoggingService) sendToDiscord(entry *LogEntry, channelID string) error {
	embed := l.createLogEmbed(entry)
	_, err := l.session.ChannelMessageSendEmbed(channelID, embed)
	return err
}

// createLogEmbed creates a Discord embed for a log entry
func (l *LoggingService) createLogEmbed(entry *LogEntry) *discordgo.MessageEmbed {
	var color int
	var emoji string

	switch entry.Level {
	case LogLevelDebug:
		color = 0x808080 // Gray
		emoji = "🔍"
	case LogLevelInfo:
		color = 0x0099ff // Blue
		emoji = "ℹ️"
	case LogLevelWarn:
		color = 0xff9900 // Orange
		emoji = "⚠️"
	case LogLevelError:
		color = 0xff0000 // Red
		emoji = "❌"
	case LogLevelFatal:
		color = 0x800000 // Dark Red
		emoji = "💀"
	default:
		color = 0x808080
		emoji = "📝"
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("%s %s", emoji, entry.Action),
		Description: entry.Message,
		Color:       color,
		Timestamp:   entry.Timestamp.Format(time.RFC3339),
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Category",
				Value:  string(entry.Category),
				Inline: true,
			},
			{
				Name:   "Level",
				Value:  string(entry.Level),
				Inline: true,
			},
		},
	}

	if entry.UserID != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "User",
			Value:  fmt.Sprintf("<@%s>", entry.UserID),
			Inline: true,
		})
	}

	if len(entry.Details) > 0 {
		detailsStr := ""
		for key, value := range entry.Details {
			detailsStr += fmt.Sprintf("**%s**: %v\n", key, value)
		}
		if len(detailsStr) > 1024 {
			detailsStr = detailsStr[:1021] + "..."
		}
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Details",
			Value:  detailsStr,
			Inline: false,
		})
	}

	return embed
}

// logToConsole logs to the console
func (l *LoggingService) logToConsole(entry *LogEntry) {
	prefix := fmt.Sprintf("[%s][%s][%s]", entry.Level, entry.Category, entry.Timestamp.Format("15:04:05"))
	if entry.UserID != "" {
		prefix += fmt.Sprintf("[%s]", entry.UserID)
	}
	log.Printf("%s %s: %s", prefix, entry.Action, entry.Message)
}

// initLogTables initializes the database tables for logging
func (l *LoggingService) initLogTables() error {
	if l.db == nil || l.db.db == nil {
		return nil // Skip if no database
	}

	createLogsTable := `
	CREATE TABLE IF NOT EXISTS system_logs (
		id SERIAL PRIMARY KEY,
		timestamp TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		level VARCHAR(10) NOT NULL,
		category VARCHAR(20) NOT NULL,
		user_id VARCHAR(32),
		action VARCHAR(100) NOT NULL,
		message TEXT NOT NULL,
		details JSONB DEFAULT '{}',
		channel_id VARCHAR(32)
	)`

	createIndexes := []string{
		`CREATE INDEX IF NOT EXISTS idx_system_logs_timestamp ON system_logs(timestamp)`,
		`CREATE INDEX IF NOT EXISTS idx_system_logs_category ON system_logs(category)`,
		`CREATE INDEX IF NOT EXISTS idx_system_logs_level ON system_logs(level)`,
		`CREATE INDEX IF NOT EXISTS idx_system_logs_user_id ON system_logs(user_id)`,
		`CREATE INDEX IF NOT EXISTS idx_system_logs_action ON system_logs(action)`,
	}

	// Create table
	if _, err := l.db.db.Exec(createLogsTable); err != nil {
		return fmt.Errorf("failed to create logs table: %v", err)
	}

	// Create indexes
	for _, indexSQL := range createIndexes {
		if _, err := l.db.db.Exec(indexSQL); err != nil {
			log.Printf("Warning: failed to create index: %v", err)
		}
	}

	log.Println("✅ Logging tables initialized")
	return nil
}

// storeLogEntry stores a log entry in the database
func (l *LoggingService) storeLogEntry(entry *LogEntry) error {
	if l.db == nil || l.db.db == nil {
		return nil // Skip if no database
	}

	// Convert details to JSON
	detailsJSON := "{}"
	if len(entry.Details) > 0 {
		// Simple JSON marshaling for basic types
		detailsJSON = "{"
		first := true
		for key, value := range entry.Details {
			if !first {
				detailsJSON += ", "
			}
			detailsJSON += fmt.Sprintf(`"%s": "%v"`, key, value)
			first = false
		}
		detailsJSON += "}"
	}

	_, err := l.db.db.Exec(`
		INSERT INTO system_logs (timestamp, level, category, user_id, action, message, details, channel_id)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, entry.Timestamp, entry.Level, entry.Category, entry.UserID, entry.Action, entry.Message, detailsJSON, entry.ChannelID)

	return err
}

// GetLogs retrieves logs from the database with filters
func (l *LoggingService) GetLogs(category LogCategory, limit int, offset int) ([]*LogEntry, error) {
	if l.db == nil || l.db.db == nil {
		return []*LogEntry{}, nil
	}

	query := `
		SELECT id, timestamp, level, category, COALESCE(user_id, ''), action, message, COALESCE(channel_id, '')
		FROM system_logs
		WHERE category = $1
		ORDER BY timestamp DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := l.db.db.Query(query, category, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var logs []*LogEntry
	for rows.Next() {
		entry := &LogEntry{
			Details: make(map[string]interface{}),
		}
		err := rows.Scan(&entry.ID, &entry.Timestamp, &entry.Level, &entry.Category,
			&entry.UserID, &entry.Action, &entry.Message, &entry.ChannelID)
		if err != nil {
			return nil, err
		}
		logs = append(logs, entry)
	}

	return logs, rows.Err()
}

// CleanupOldLogs removes logs older than the specified duration
func (l *LoggingService) CleanupOldLogs(maxAge time.Duration) error {
	if l.db == nil || l.db.db == nil {
		return nil
	}

	cutoffTime := time.Now().Add(-maxAge)
	result, err := l.db.db.Exec(`DELETE FROM system_logs WHERE timestamp < $1`, cutoffTime)
	if err != nil {
		return err
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		log.Printf("🧹 Cleaned up %d old log entries", rowsAffected)
	}

	return nil
}

// StartLogRotation starts a background process to rotate logs
func (l *LoggingService) StartLogRotation(rotationInterval time.Duration, maxAge time.Duration) {
	go func() {
		ticker := time.NewTicker(rotationInterval)
		defer ticker.Stop()

		for range ticker.C {
			if err := l.CleanupOldLogs(maxAge); err != nil {
				log.Printf("Failed to cleanup old logs: %v", err)
			}
		}
	}()
	log.Printf("🔄 Log rotation started (interval: %v, max age: %v)", rotationInterval, maxAge)
}

// GetChannelConfig returns the current channel configuration
func (l *LoggingService) GetChannelConfig() *LogChannelConfig {
	return l.config
}
