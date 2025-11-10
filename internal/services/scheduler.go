package services

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/robfig/cron/v3"
)

// SchedulerService manages server scheduling (start/stop/restart)
type SchedulerService struct {
	db       *DatabaseService
	enhanced *EnhancedServerService
	cron     *cron.Cron
	ctx      context.Context
	cancel   context.CancelFunc
	// Metrics
	activeGauge        prometheus.Gauge
	executionsCounter  *prometheus.CounterVec
}

// ServerSchedule represents a scheduled action for a server
type ServerSchedule struct {
	ID             int
	ServerID       int
	DiscordID      string
	Action         string // "start", "stop", "restart"
	CronExpression string
	Timezone       string
	Enabled        bool
	LastRun        *time.Time
	NextRun        *time.Time
	CreatedAt      time.Time
}

// NewSchedulerService creates a new scheduler service
func NewSchedulerService(db *DatabaseService, enhanced *EnhancedServerService, activeGauge prometheus.Gauge, executions *prometheus.CounterVec) *SchedulerService {
	ctx, cancel := context.WithCancel(context.Background())

	return &SchedulerService{
		db:       db,
		enhanced: enhanced,
		cron:     cron.New(cron.WithSeconds()),
		ctx:      ctx,
		cancel:   cancel,
		activeGauge:       activeGauge,
		executionsCounter: executions,
	}
}

// Start starts the scheduler service
func (s *SchedulerService) Start() error {
	s.cron.Start()

	// Background worker checks for due schedules every minute
	ticker := time.NewTicker(1 * time.Minute)
	go func() {
		for {
			select {
			case <-ticker.C:
				s.processSchedules()
			case <-s.ctx.Done():
				ticker.Stop()
				return
			}
		}
	}()

	log.Println("📅 Scheduler service started")
	return nil
}

// Stop stops the scheduler service
func (s *SchedulerService) Stop() {
	s.cancel()
	s.cron.Stop()
	log.Println("📅 Scheduler service stopped")
}

// CreateSchedule creates a new schedule for a server
func (s *SchedulerService) CreateSchedule(serverID int, discordID, action, cronExpr, timezone string) (*ServerSchedule, error) {
	// Validate cron expression
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	schedule, err := parser.Parse(cronExpr)
	if err != nil {
		return nil, fmt.Errorf("invalid cron expression: %v", err)
	}

	// Calculate next run time
	nextRun := schedule.Next(time.Now())

	// Insert into database
	var scheduleID int
	err = s.db.DB().QueryRow(`
		INSERT INTO server_schedules (server_id, discord_id, action, cron_expression, timezone, next_run)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id
	`, serverID, discordID, action, cronExpr, timezone, nextRun).Scan(&scheduleID)

	if err != nil {
		return nil, fmt.Errorf("failed to create schedule: %v", err)
	}

	log.Printf("✅ Created schedule %d: %s server %d at %s", scheduleID, action, serverID, cronExpr)

	sched := &ServerSchedule{
		ID:             scheduleID,
		ServerID:       serverID,
		DiscordID:      discordID,
		Action:         action,
		CronExpression: cronExpr,
		Timezone:       timezone,
		Enabled:        true,
		NextRun:        &nextRun,
		CreatedAt:      time.Now(),
	}
	s.updateActiveGauge()
	return sched, nil
}

// ListSchedules lists all schedules for a user
func (s *SchedulerService) ListSchedules(discordID string) ([]*ServerSchedule, error) {
	rows, err := s.db.DB().Query(`
		SELECT id, server_id, discord_id, action, cron_expression, timezone, enabled, last_run, next_run, created_at
		FROM server_schedules
		WHERE discord_id = $1
		ORDER BY next_run ASC
	`, discordID)
	if err != nil {
		return nil, fmt.Errorf("failed to list schedules: %v", err)
	}
	defer rows.Close()

	schedules := make([]*ServerSchedule, 0)
	for rows.Next() {
		var s ServerSchedule
		var lastRun, nextRun *time.Time

		err := rows.Scan(&s.ID, &s.ServerID, &s.DiscordID, &s.Action, &s.CronExpression,
			&s.Timezone, &s.Enabled, &lastRun, &nextRun, &s.CreatedAt)
		if err != nil {
			continue
		}

		s.LastRun = lastRun
		s.NextRun = nextRun
		schedules = append(schedules, &s)
	}

	return schedules, nil
}

// DeleteSchedule deletes a schedule
func (s *SchedulerService) DeleteSchedule(scheduleID int, discordID string) error {
	result, err := s.db.DB().Exec(`
		DELETE FROM server_schedules
		WHERE id = $1 AND discord_id = $2
	`, scheduleID, discordID)

	if err != nil {
		return fmt.Errorf("failed to delete schedule: %v", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("schedule not found or unauthorized")
	}

	log.Printf("🗑️ Deleted schedule %d", scheduleID)
	s.updateActiveGauge()
	return nil
}

// EnableSchedule enables a schedule
func (s *SchedulerService) EnableSchedule(scheduleID int, discordID string) error {
	_, err := s.db.DB().Exec(`
		UPDATE server_schedules
		SET enabled = true
		WHERE id = $1 AND discord_id = $2
	`, scheduleID, discordID)

	if err != nil {
		return fmt.Errorf("failed to enable schedule: %v", err)
	}

	log.Printf("✅ Enabled schedule %d", scheduleID)
	s.updateActiveGauge()
	return nil
}

// DisableSchedule disables a schedule
func (s *SchedulerService) DisableSchedule(scheduleID int, discordID string) error {
	_, err := s.db.DB().Exec(`
		UPDATE server_schedules
		SET enabled = false
		WHERE id = $1 AND discord_id = $2
	`, scheduleID, discordID)

	if err != nil {
		return fmt.Errorf("failed to disable schedule: %v", err)
	}

	log.Printf("⏸️ Disabled schedule %d", scheduleID)
	s.updateActiveGauge()
	return nil
}

// processSchedules checks for schedules that need to be executed
func (s *SchedulerService) processSchedules() {
	// Query schedules due for execution
	rows, err := s.db.DB().Query(`
		SELECT id, server_id, action, cron_expression, timezone
		FROM server_schedules
		WHERE enabled = true AND (next_run IS NULL OR next_run <= NOW())
	`)
	if err != nil {
		log.Printf("Error fetching schedules: %v", err)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var id, serverID int
		var action, cronExpr, tz string

		if err := rows.Scan(&id, &serverID, &action, &cronExpr, &tz); err != nil {
			continue
		}

		// Execute action
		go s.executeSchedule(serverID, action, cronExpr, id)
	}
}

// executeSchedule executes a scheduled action
func (s *SchedulerService) executeSchedule(serverID int, action, cronExpr string, scheduleID int) {
	log.Printf("⏰ Executing scheduled %s for server %d", action, serverID)

	// Get server details
	var serverName, discordID string
	err := s.db.DB().QueryRow(`
		SELECT name, discord_id
		FROM game_servers
		WHERE id = $1
	`, serverID).Scan(&serverName, &discordID)

	if err != nil {
		log.Printf("❌ Failed to get server details: %v", err)
		return
	}

	// Execute the action based on type
	// TODO: Implement actual server lifecycle operations
	// For now, we'll log the action but not execute until start/stop/restart methods are implemented
	var execErr error
	
	switch action {
	case "start":
		log.Printf("🚀 Scheduled START for server %s (ID: %d)", serverName, serverID)
		// TODO: Implement start logic - may require pod manipulation or Agones state changes
		// execErr = s.startServer(serverID, serverName, discordID)
		log.Printf("⚠️  Start action not yet implemented - placeholder execution")
		
	case "stop":
		log.Printf("🛑 Scheduled STOP for server %s (ID: %d)", serverName, serverID)
		// TODO: Implement stop logic - Delete GameServer or scale to zero
		// execErr = s.enhanced.DeleteGameServer(ctx, serverName, discordID)
		log.Printf("⚠️  Stop action not yet implemented - placeholder execution")
		
	case "restart":
		log.Printf("🔄 Scheduled RESTART for server %s (ID: %d)", serverName, serverID)
		// TODO: Implement restart logic - Delete and recreate or send RCON restart command
		// execErr = s.restartServer(serverID, serverName, discordID)
		log.Printf("⚠️  Restart action not yet implemented - placeholder execution")
		
	default:
		log.Printf("⚠️ Unknown action: %s", action)
		if s.executionsCounter != nil {
			s.executionsCounter.WithLabelValues(action, "error").Inc()
		}
		return
	}

	// Update metrics based on execution result
	if s.executionsCounter != nil {
		status := "success"
		if execErr != nil {
			status = "error"
		}
		s.executionsCounter.WithLabelValues(action, status).Inc()
	}
	
	if execErr != nil {
		return // Don't update next_run on error
	}

	// Calculate next run time
	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	schedule, err := parser.Parse(cronExpr)
	if err != nil {
		log.Printf("Error parsing cron: %v", err)
		return
	}

	nextRun := schedule.Next(time.Now())

	// Update database
	_, err = s.db.DB().Exec(`
		UPDATE server_schedules
		SET last_run = NOW(), next_run = $1
		WHERE id = $2
	`, nextRun, scheduleID)

	if err != nil {
		log.Printf("Error updating schedule: %v", err)
	}

	log.Printf("✅ Schedule %d executed successfully, next run: %s", scheduleID, nextRun.Format("2006-01-02 15:04"))
	if s.executionsCounter != nil {
		s.executionsCounter.WithLabelValues(action, "success").Inc()
	}
}

// updateActiveGauge recalculates active schedules count
func (s *SchedulerService) updateActiveGauge() {
	if s.activeGauge == nil {
		return
	}
	var count int
	err := s.db.DB().QueryRow(`SELECT COUNT(*) FROM server_schedules WHERE enabled = true`).Scan(&count)
	if err != nil {
		log.Printf("⚠️ Failed to update active schedules gauge: %v", err)
		return
	}
	s.activeGauge.Set(float64(count))
}

// GetServerSchedules gets all schedules for a specific server
func (s *SchedulerService) GetServerSchedules(serverID int, discordID string) ([]*ServerSchedule, error) {
	rows, err := s.db.DB().Query(`
		SELECT id, server_id, discord_id, action, cron_expression, timezone, enabled, last_run, next_run, created_at
		FROM server_schedules
		WHERE server_id = $1 AND discord_id = $2
		ORDER BY next_run ASC
	`, serverID, discordID)
	if err != nil {
		return nil, fmt.Errorf("failed to get server schedules: %v", err)
	}
	defer rows.Close()

	schedules := make([]*ServerSchedule, 0)
	for rows.Next() {
		var s ServerSchedule
		var lastRun, nextRun *time.Time

		err := rows.Scan(&s.ID, &s.ServerID, &s.DiscordID, &s.Action, &s.CronExpression,
			&s.Timezone, &s.Enabled, &lastRun, &nextRun, &s.CreatedAt)
		if err != nil {
			continue
		}

		s.LastRun = lastRun
		s.NextRun = nextRun
		schedules = append(schedules, &s)
	}

	return schedules, nil
}
