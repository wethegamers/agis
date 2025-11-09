package services

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"
)

// GuildProvisioningService handles automatic server provisioning from guild treasury
type GuildProvisioningService struct {
	db             *sql.DB
	agonesService  *AgonesService // For creating game servers
	databaseSvc    *DatabaseService
	notificationSvc *NotificationService
}

// ServerTemplate defines a server configuration template
type ServerTemplate struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	GameType    string `json:"game_type"`    // e.g., "minecraft", "valheim", "palworld"
	Size        string `json:"size"`         // e.g., "small", "medium", "large"
	Cost        int    `json:"cost"`         // Cost in Game Credits per hour
	SetupCost   int    `json:"setup_cost"`   // One-time setup cost in GC
	MaxPlayers  int    `json:"max_players"`
	CPURequest  string `json:"cpu_request"`  // e.g., "1000m"
	MemoryRequest string `json:"memory_request"` // e.g., "2Gi"
	Description string `json:"description"`
}

// ProvisionRequest represents a request to provision a server
type ProvisionRequest struct {
	GuildID      string    `json:"guild_id"`
	RequestedBy  string    `json:"requested_by"`  // Discord user ID
	TemplateID   string    `json:"template_id"`
	ServerName   string    `json:"server_name"`
	DurationHours int      `json:"duration_hours"` // How long to run
	AutoRenew    bool      `json:"auto_renew"`     // Renew from treasury automatically
	RequestedAt  time.Time `json:"requested_at"`
	Status       string    `json:"status"` // "pending", "approved", "provisioning", "active", "terminated"
	ServerID     string    `json:"server_id,omitempty"`
}

// NewGuildProvisioningService creates a new guild provisioning service
func NewGuildProvisioningService(db *sql.DB, agonesService *AgonesService, dbSvc *DatabaseService, notifySvc *NotificationService) *GuildProvisioningService {
	return &GuildProvisioningService{
		db:             db,
		agonesService:  agonesService,
		databaseSvc:    dbSvc,
		notificationSvc: notifySvc,
	}
}

// GetAvailableTemplates returns all available server templates
func (s *GuildProvisioningService) GetAvailableTemplates() ([]*ServerTemplate, error) {
	// TODO: Load from database or config
	// For now, return hardcoded templates
	return []*ServerTemplate{
		{
			ID:          "minecraft-small",
			Name:        "Minecraft (Small)",
			GameType:    "minecraft",
			Size:        "small",
			Cost:        100, // 100 GC/hour
			SetupCost:   500,
			MaxPlayers:  10,
			CPURequest:  "1000m",
			MemoryRequest: "2Gi",
			Description: "Small Minecraft server for up to 10 players",
		},
		{
			ID:          "minecraft-medium",
			Name:        "Minecraft (Medium)",
			GameType:    "minecraft",
			Size:        "medium",
			Cost:        200,
			SetupCost:   1000,
			MaxPlayers:  25,
			CPURequest:  "2000m",
			MemoryRequest: "4Gi",
			Description: "Medium Minecraft server for up to 25 players",
		},
		{
			ID:          "valheim-small",
			Name:        "Valheim (Small)",
			GameType:    "valheim",
			Size:        "small",
			Cost:        150,
			SetupCost:   750,
			MaxPlayers:  10,
			CPURequest:  "1500m",
			MemoryRequest: "3Gi",
			Description: "Small Valheim server for up to 10 players",
		},
	}, nil
}

// RequestProvisioning creates a new provisioning request
func (s *GuildProvisioningService) RequestProvisioning(ctx context.Context, req *ProvisionRequest) error {
	// Validate guild exists and has treasury
	var treasuryBalance int
	err := s.db.QueryRowContext(ctx,
		"SELECT balance FROM guild_treasury WHERE guild_id = $1",
		req.GuildID,
	).Scan(&treasuryBalance)
	if err == sql.ErrNoRows {
		return errors.New("guild does not have a treasury")
	}
	if err != nil {
		return fmt.Errorf("failed to query treasury: %w", err)
	}

	// Get template
	templates, err := s.GetAvailableTemplates()
	if err != nil {
		return err
	}

	var template *ServerTemplate
	for _, t := range templates {
		if t.ID == req.TemplateID {
			template = t
			break
		}
	}
	if template == nil {
		return fmt.Errorf("template %s not found", req.TemplateID)
	}

	// Calculate total cost
	totalCost := template.SetupCost + (template.Cost * req.DurationHours)
	if treasuryBalance < totalCost {
		return fmt.Errorf("insufficient treasury balance: need %d GC, have %d GC", totalCost, treasuryBalance)
	}

	// Insert provisioning request
	req.RequestedAt = time.Now()
	req.Status = "pending"

	_, err = s.db.ExecContext(ctx,
		`INSERT INTO server_provision_requests 
		(guild_id, requested_by, template_id, server_name, duration_hours, auto_renew, requested_at, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		req.GuildID, req.RequestedBy, req.TemplateID, req.ServerName,
		req.DurationHours, req.AutoRenew, req.RequestedAt, req.Status,
	)
	if err != nil {
		return fmt.Errorf("failed to insert provision request: %w", err)
	}

	// TODO: Send notification to guild admins for approval
	// For now, auto-approve
	return s.ApproveProvisioning(ctx, req.GuildID, req.RequestedBy)
}

// ApproveProvisioning approves and executes a provisioning request
func (s *GuildProvisioningService) ApproveProvisioning(ctx context.Context, guildID, requestID string) error {
	// Get request
	var req ProvisionRequest
	err := s.db.QueryRowContext(ctx,
		`SELECT guild_id, requested_by, template_id, server_name, duration_hours, auto_renew, requested_at, status
		FROM server_provision_requests
		WHERE guild_id = $1 AND requested_by = $2 AND status = 'pending'
		LIMIT 1`,
		guildID, requestID,
	).Scan(&req.GuildID, &req.RequestedBy, &req.TemplateID, &req.ServerName,
		&req.DurationHours, &req.AutoRenew, &req.RequestedAt, &req.Status)
	if err == sql.ErrNoRows {
		return errors.New("no pending provision request found")
	}
	if err != nil {
		return fmt.Errorf("failed to query provision request: %w", err)
	}

	// Get template
	templates, _ := s.GetAvailableTemplates()
	var template *ServerTemplate
	for _, t := range templates {
		if t.ID == req.TemplateID {
			template = t
			break
		}
	}
	if template == nil {
		return fmt.Errorf("template %s not found", req.TemplateID)
	}

	// Deduct costs from treasury
	totalCost := template.SetupCost + (template.Cost * req.DurationHours)
	_, err = s.db.ExecContext(ctx,
		`UPDATE guild_treasury SET balance = balance - $1 WHERE guild_id = $2`,
		totalCost, req.GuildID,
	)
	if err != nil {
		return fmt.Errorf("failed to deduct from treasury: %w", err)
	}

	// Log transaction
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO treasury_transactions (guild_id, amount, transaction_type, description, created_at)
		VALUES ($1, $2, 'debit', $3, $4)`,
		req.GuildID, totalCost,
		fmt.Sprintf("Server provisioning: %s (%dh)", req.ServerName, req.DurationHours),
		time.Now(),
	)
	if err != nil {
		return fmt.Errorf("failed to log transaction: %w", err)
	}

	// Update request status
	_, err = s.db.ExecContext(ctx,
		`UPDATE server_provision_requests SET status = 'provisioning' WHERE guild_id = $1 AND requested_by = $2`,
		req.GuildID, req.RequestedBy,
	)
	if err != nil {
		return fmt.Errorf("failed to update request status: %w", err)
	}

	// Provision server via Agones
	serverID, err := s.provisionServer(ctx, &req, template)
	if err != nil {
		// Rollback: refund treasury
		s.db.ExecContext(ctx,
			`UPDATE guild_treasury SET balance = balance + $1 WHERE guild_id = $2`,
			totalCost, req.GuildID,
		)
		return fmt.Errorf("failed to provision server: %w", err)
	}

	// Update request with server ID
	_, err = s.db.ExecContext(ctx,
		`UPDATE server_provision_requests SET status = 'active', server_id = $1 WHERE guild_id = $2 AND requested_by = $3`,
		serverID, req.GuildID, req.RequestedBy,
	)
	if err != nil {
		return fmt.Errorf("failed to update request with server ID: %w", err)
	}

	// Schedule termination after duration
	go s.scheduleTermination(guildID, serverID, req.DurationHours, req.AutoRenew)

	return nil
}

// provisionServer creates the actual game server via Agones
func (s *GuildProvisioningService) provisionServer(ctx context.Context, req *ProvisionRequest, template *ServerTemplate) (string, error) {
	if s.agonesService == nil {
		// Local mode - simulate provisioning
		return fmt.Sprintf("sim-%s-%d", req.GuildID, time.Now().Unix()), nil
	}

	// Create Agones GameServer
	gameServer := &GameServerRequest{
		Name:      req.ServerName,
		Namespace: "game-servers", // TODO: Make configurable
		GameType:  template.GameType,
		Labels: map[string]string{
			"guild-id":    req.GuildID,
			"auto-renew":  fmt.Sprintf("%t", req.AutoRenew),
			"template-id": template.ID,
		},
		Resources: ResourceRequirements{
			CPURequest:    template.CPURequest,
			MemoryRequest: template.MemoryRequest,
		},
	}

	serverID, err := s.agonesService.CreateGameServer(ctx, gameServer)
	if err != nil {
		return "", err
	}

	return serverID, nil
}

// scheduleTermination schedules server termination after duration
func (s *GuildProvisioningService) scheduleTermination(guildID, serverID string, hours int, autoRenew bool) {
	time.Sleep(time.Duration(hours) * time.Hour)

	ctx := context.Background()

	if autoRenew {
		// Check treasury balance and renew if possible
		err := s.renewServer(ctx, guildID, serverID)
		if err != nil {
			// Not enough balance, terminate
			s.terminateServer(ctx, guildID, serverID)
		}
	} else {
		// No auto-renew, terminate
		s.terminateServer(ctx, guildID, serverID)
	}
}

// renewServer renews a server for another period
func (s *GuildProvisioningService) renewServer(ctx context.Context, guildID, serverID string) error {
	// Get server template
	var templateID string
	err := s.db.QueryRowContext(ctx,
		`SELECT template_id FROM server_provision_requests WHERE guild_id = $1 AND server_id = $2`,
		guildID, serverID,
	).Scan(&templateID)
	if err != nil {
		return fmt.Errorf("failed to find provision request: %w", err)
	}

	templates, _ := s.GetAvailableTemplates()
	var template *ServerTemplate
	for _, t := range templates {
		if t.ID == templateID {
			template = t
			break
		}
	}
	if template == nil {
		return fmt.Errorf("template %s not found", templateID)
	}

	// Deduct hourly cost from treasury
	renewalCost := template.Cost // 1 hour renewal
	_, err = s.db.ExecContext(ctx,
		`UPDATE guild_treasury SET balance = balance - $1 WHERE guild_id = $2 AND balance >= $1`,
		renewalCost, guildID,
	)
	if err != nil {
		return fmt.Errorf("insufficient balance for renewal: %w", err)
	}

	// Log transaction
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO treasury_transactions (guild_id, amount, transaction_type, description, created_at)
		VALUES ($1, $2, 'debit', $3, $4)`,
		guildID, renewalCost, fmt.Sprintf("Server renewal: %s", serverID), time.Now(),
	)
	if err != nil {
		return fmt.Errorf("failed to log renewal transaction: %w", err)
	}

	// Schedule next termination check in 1 hour
	go s.scheduleTermination(guildID, serverID, 1, true)

	return nil
}

// terminateServer terminates a running server
func (s *GuildProvisioningService) terminateServer(ctx context.Context, guildID, serverID string) error {
	if s.agonesService != nil {
		// Delete server via Agones
		err := s.agonesService.DeleteGameServer(ctx, serverID)
		if err != nil {
			return fmt.Errorf("failed to delete game server: %w", err)
		}
	}

	// Update request status
	_, err := s.db.ExecContext(ctx,
		`UPDATE server_provision_requests SET status = 'terminated' WHERE guild_id = $1 AND server_id = $2`,
		guildID, serverID,
	)
	if err != nil {
		return fmt.Errorf("failed to update request status: %w", err)
	}

	// TODO: Send notification to guild
	return nil
}

// GetGuildServers returns all active servers for a guild
func (s *GuildProvisioningService) GetGuildServers(ctx context.Context, guildID string) ([]*ProvisionRequest, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT guild_id, requested_by, template_id, server_name, duration_hours, auto_renew, requested_at, status, server_id
		FROM server_provision_requests
		WHERE guild_id = $1 AND status IN ('active', 'provisioning')
		ORDER BY requested_at DESC`,
		guildID,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to query guild servers: %w", err)
	}
	defer rows.Close()

	var servers []*ProvisionRequest
	for rows.Next() {
		var req ProvisionRequest
		var serverID sql.NullString
		err := rows.Scan(&req.GuildID, &req.RequestedBy, &req.TemplateID, &req.ServerName,
			&req.DurationHours, &req.AutoRenew, &req.RequestedAt, &req.Status, &serverID)
		if err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}
		if serverID.Valid {
			req.ServerID = serverID.String
		}
		servers = append(servers, &req)
	}

	return servers, nil
}
