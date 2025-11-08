package services

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	"agis-bot/internal/config"

	_ "github.com/lib/pq"
)

type DatabaseService struct {
	db        *sql.DB
	localMode bool
	// In-memory storage for local development
	localUsers       map[string]*User
	localConversions map[string]bool
	localMutex       sync.RWMutex
}

// User represents a Discord user in our system
type User struct {
	DiscordID   string
	Credits     int
	Tier        string
	LastDaily   time.Time
	LastWork    time.Time
	ServersUsed int
	JoinDate    time.Time
}

// GameServer represents a user's game server
type GameServer struct {
	ID            int
	DiscordID     string
	Name          string
	GameType      string
	Status        string
	Address       string
	Port          int
	CreatedAt     time.Time
	StoppedAt     *time.Time
	LastHeartbeat *time.Time
	CostPerHour   int
	IsPublic      bool
	Description   string
	ErrorMessage  string
	CleanupAt     *time.Time
	// Kubernetes/Agones integration fields
	KubernetesUID  string     // UID of the GameServer in Kubernetes
	AgonesStatus   string     // Current Agones GameServer status
	LastStatusSync *time.Time // Last time we synced with Kubernetes
}

// PublicServer represents a server in the WTG Public Lobby
type PublicServer struct {
	ID          int
	ServerName  string
	GameType    string
	OwnerID     string
	OwnerName   string
	Address     string
	Port        int
	Description string
	Players     int
	MaxPlayers  int
	AddedAt     time.Time
}

// BotRole represents a Discord role assigned to bot permissions
type BotRole struct {
	ID       int
	RoleID   string
	RoleType string // "admin" or "moderator"
	GuildID  string
	AddedAt  time.Time
}

func NewDatabaseService(cfg *config.Config) (*DatabaseService, error) {
	// Skip database connection if host is empty (for local development)
	if cfg.Database.Host == "" {
		log.Println("📄 Database disabled (DB_HOST is empty) - running in local mode")
		return &DatabaseService{
			db:               nil,
			localMode:        true,
			localUsers:       make(map[string]*User),
			localConversions: make(map[string]bool),
			localMutex:       sync.RWMutex{},
		}, nil
	}

	var connStr string
	if cfg.Database.Password != "" {
		connStr = fmt.Sprintf("host=%s user=%s password=%s dbname=%s sslmode=disable",
			cfg.Database.Host, cfg.Database.User, cfg.Database.Password, cfg.Database.Name)
	} else {
		connStr = fmt.Sprintf("host=%s user=%s dbname=%s sslmode=disable",
			cfg.Database.Host, cfg.Database.User, cfg.Database.Name)
	}

	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to database: %v", err)
	}

	service := &DatabaseService{
		db:               db,
		localMode:        false,
		localUsers:       nil,
		localConversions: nil,
		localMutex:       sync.RWMutex{},
	}
	if err := service.initDatabase(); err != nil {
		return nil, fmt.Errorf("failed to initialize database: %v", err)
	}

	return service, nil
}

func (d *DatabaseService) Close() error {
	if d.db != nil {
		return d.db.Close()
	}
	return nil
}

// DB returns the underlying database connection
func (d *DatabaseService) DB() *sql.DB {
	return d.db
}

func (d *DatabaseService) initDatabase() error {
	// Create users table
	createUsersTable := `
	CREATE TABLE IF NOT EXISTS users (
		discord_id VARCHAR(32) PRIMARY KEY,
		credits INTEGER DEFAULT 100,
		tier VARCHAR(20) DEFAULT 'free',
		last_daily TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		last_work TIMESTAMP DEFAULT '1970-01-01',
		servers_used INTEGER DEFAULT 0,
		join_date TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`

	// Create game_servers table with public lobby support
	createServersTable := `
	CREATE TABLE IF NOT EXISTS game_servers (
		id SERIAL PRIMARY KEY,
		discord_id VARCHAR(32) NOT NULL,
		name VARCHAR(100) NOT NULL,
		game_type VARCHAR(50) NOT NULL,
		status VARCHAR(20) DEFAULT 'creating',
		address VARCHAR(255),
		port INTEGER,
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		cost_per_hour INTEGER NOT NULL,
		is_public BOOLEAN DEFAULT FALSE,
		description TEXT DEFAULT '',
		FOREIGN KEY (discord_id) REFERENCES users(discord_id)
	)`

	// Create public_servers table for WTG Public Lobby
	createPublicServersTable := `
	CREATE TABLE IF NOT EXISTS public_servers (
		id SERIAL PRIMARY KEY,
		server_name VARCHAR(100) NOT NULL,
		game_type VARCHAR(50) NOT NULL,
		owner_id VARCHAR(32) NOT NULL,
		owner_name VARCHAR(100) NOT NULL,
		address VARCHAR(255) NOT NULL,
		port INTEGER NOT NULL,
		description TEXT DEFAULT '',
		players INTEGER DEFAULT 0,
		max_players INTEGER DEFAULT 20,
		added_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (owner_id) REFERENCES users(discord_id)
	)`

	// Create command_usage table for analytics
	createUsageTable := `
	CREATE TABLE IF NOT EXISTS command_usage (
		id SERIAL PRIMARY KEY,
		discord_id VARCHAR(32) NOT NULL,
		command VARCHAR(100) NOT NULL,
		used_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
	)`

	// Create bot_roles table for permission management
	createRolesTable := `
	CREATE TABLE IF NOT EXISTS bot_roles (
		id SERIAL PRIMARY KEY,
		role_id VARCHAR(32) NOT NULL,
		role_type VARCHAR(20) NOT NULL CHECK (role_type IN ('admin', 'moderator')),
		guild_id VARCHAR(32) NOT NULL,
		added_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		UNIQUE(role_id, guild_id)
	)`

	// Create user_stats table for profiles and analytics
	createUserStatsTable := `
	CREATE TABLE IF NOT EXISTS user_stats (
		discord_id VARCHAR(32) PRIMARY KEY,
		total_servers_created INTEGER DEFAULT 0,
		total_commands_used INTEGER DEFAULT 0,
		total_credits_earned INTEGER DEFAULT 0,
		total_credits_spent INTEGER DEFAULT 0,
		last_command_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (discord_id) REFERENCES users(discord_id)
	)`

	tables := []string{createUsersTable, createServersTable, createPublicServersTable, createUsageTable, createRolesTable, createUserStatsTable}

	for _, table := range tables {
		if _, err := d.db.Exec(table); err != nil {
			return fmt.Errorf("failed to create table: %v", err)
		}
	}

	// Add new columns for Kubernetes/Agones integration (if they don't exist)
	alterTable := `
	ALTER TABLE game_servers 
	ADD COLUMN IF NOT EXISTS kubernetes_uid VARCHAR(255),
	ADD COLUMN IF NOT EXISTS agones_status VARCHAR(50),
	ADD COLUMN IF NOT EXISTS last_status_sync TIMESTAMP,
	ADD COLUMN IF NOT EXISTS stopped_at TIMESTAMP,
	ADD COLUMN IF NOT EXISTS last_heartbeat TIMESTAMP,
	ADD COLUMN IF NOT EXISTS error_message TEXT,
	ADD COLUMN IF NOT EXISTS cleanup_at TIMESTAMP,
	ADD COLUMN IF NOT EXISTS updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP`

	if _, err := d.db.Exec(alterTable); err != nil {
		return fmt.Errorf("failed to alter game_servers table: %v", err)
	}

	// Migration: Add new columns if they don't exist
	migrations := []string{
		`ALTER TABLE users ADD COLUMN IF NOT EXISTS last_work TIMESTAMP DEFAULT '1970-01-01'`,
		`ALTER TABLE game_servers ADD COLUMN IF NOT EXISTS is_public BOOLEAN DEFAULT FALSE`,
		`ALTER TABLE game_servers ADD COLUMN IF NOT EXISTS description TEXT DEFAULT ''`,
		// Server cleanup implementation migrations
		`ALTER TABLE game_servers ADD COLUMN IF NOT EXISTS stopped_at TIMESTAMP`,
		`ALTER TABLE game_servers ADD COLUMN IF NOT EXISTS last_heartbeat TIMESTAMP DEFAULT NOW()`,
		`ALTER TABLE game_servers ADD COLUMN IF NOT EXISTS error_message TEXT DEFAULT ''`,
		`ALTER TABLE game_servers ADD COLUMN IF NOT EXISTS cleanup_at TIMESTAMP`,
	}

	// Ad conversions table (idempotency)
	createConversions := `
CREATE TABLE IF NOT EXISTS ad_conversions (
	conversion_id TEXT PRIMARY KEY,
	uid VARCHAR(64) NOT NULL,
	amount INTEGER NOT NULL,
	source VARCHAR(32) NOT NULL,
	created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
)`

	if _, err := d.db.Exec(createConversions); err != nil {
		return fmt.Errorf("failed to create ad_conversions: %v", err)
	}

	for _, migration := range migrations {
		if _, err := d.db.Exec(migration); err != nil {
			log.Printf("Migration warning: %v", err)
		}
	}

	log.Println("✅ Database initialization completed")
	return nil
}

// User operations
func (d *DatabaseService) GetOrCreateUser(discordID string) (*User, error) {
	if d.localMode {
		d.localMutex.RLock()
		user, exists := d.localUsers[discordID]
		d.localMutex.RUnlock()

		if exists {
			return user, nil
		}

		// Create new user in local mode
		newUser := &User{
			DiscordID:   discordID,
			Credits:     100,
			Tier:        "free",
			LastDaily:   time.Now().AddDate(0, 0, -1),
			LastWork:    time.Now().AddDate(0, 0, -2),
			ServersUsed: 0,
			JoinDate:    time.Now(),
		}

		d.localMutex.Lock()
		d.localUsers[discordID] = newUser
		d.localMutex.Unlock()

		return newUser, nil
	}

	user := &User{}
	err := d.db.QueryRow(`
		SELECT discord_id, credits, tier, last_daily, COALESCE(last_work, '1970-01-01'), servers_used, join_date
		FROM users WHERE discord_id = $1
	`, discordID).Scan(&user.DiscordID, &user.Credits, &user.Tier, &user.LastDaily, &user.LastWork, &user.ServersUsed, &user.JoinDate)

	if err == sql.ErrNoRows {
		// Create new user
		_, err = d.db.Exec(`
			INSERT INTO users (discord_id, credits, tier, last_daily, last_work, servers_used, join_date)
			VALUES ($1, 100, 'free', $2, '1970-01-01', 0, CURRENT_TIMESTAMP)
		`, discordID, time.Now().AddDate(0, 0, -1))

		if err != nil {
			return nil, fmt.Errorf("failed to create user: %v", err)
		}

		return &User{
			DiscordID: discordID,
			Credits:   100,
			Tier:      "free",
			LastDaily: time.Now().AddDate(0, 0, -1),
			LastWork:  time.Now().AddDate(0, 0, -2),
			JoinDate:  time.Now(),
		}, nil
	} else if err != nil {
		return nil, fmt.Errorf("failed to get user: %v", err)
	}

	return user, nil
}

func (d *DatabaseService) UpdateUserCredits(discordID string, newCredits int) error {
	if d.db == nil {
		return nil
	}
	_, err := d.db.Exec(`UPDATE users SET credits = $1 WHERE discord_id = $2`, newCredits, discordID)
	return err
}

func (d *DatabaseService) UpdateUserWork(discordID string, credits int, lastWork time.Time) error {
	if d.db == nil {
		return nil
	}
	_, err := d.db.Exec(`UPDATE users SET credits = $1, last_work = $2 WHERE discord_id = $3`,
		credits, lastWork, discordID)
	return err
}

func (d *DatabaseService) UpdateLastDaily(discordID string) error {
	if d.localMode {
		d.localMutex.Lock()
		defer d.localMutex.Unlock()

		user, exists := d.localUsers[discordID]
		if exists {
			user.LastDaily = time.Now()
		}
		return nil
	}

	if d.db == nil {
		return nil
	}
	_, err := d.db.Exec(`UPDATE users SET last_daily = $1 WHERE discord_id = $2`,
		time.Now(), discordID)
	return err
}

func (d *DatabaseService) AddCredits(discordID string, amount int) error {
	if d.localMode {
		d.localMutex.Lock()
		defer d.localMutex.Unlock()

		user, exists := d.localUsers[discordID]
		if !exists {
			// Create user if doesn't exist
			user = &User{
				DiscordID:   discordID,
				Credits:     100,
				Tier:        "free",
				LastDaily:   time.Now().AddDate(0, 0, -1),
				LastWork:    time.Now().AddDate(0, 0, -2),
				ServersUsed: 0,
				JoinDate:    time.Now(),
			}
			d.localUsers[discordID] = user
		}

		user.Credits += amount
		if user.Credits < 0 {
			user.Credits = 0
		}
		return nil
	}

	_, err := d.db.Exec(`
		UPDATE users SET credits = credits + $1 WHERE discord_id = $2
	`, amount, discordID)

	return err
}

// ProcessAdConversion credits the user if the conversion id is new (idempotent)
var ErrDuplicate = errors.New("duplicate conversion")

func (d *DatabaseService) ProcessAdConversion(uid string, amount int, conversionID, source string) error {
	if d.localMode {
		d.localMutex.Lock()
		defer d.localMutex.Unlock()
		if d.localConversions[conversionID] {
			return nil
		}
		if _, exists := d.localUsers[uid]; !exists {
			d.localUsers[uid] = &User{DiscordID: uid, Credits: 100, Tier: "free", LastDaily: time.Now().AddDate(0, 0, -1), LastWork: time.Now().AddDate(0, 0, -2), JoinDate: time.Now()}
		}
		d.localUsers[uid].Credits += amount
		if d.localUsers[uid].Credits < 0 {
			d.localUsers[uid].Credits = 0
		}
		d.localConversions[conversionID] = true
		return nil
	}
	if d.db == nil {
		return nil
	}
	// check existing
	var existing string
	err := d.db.QueryRow(`SELECT conversion_id FROM ad_conversions WHERE conversion_id=$1`, conversionID).Scan(&existing)
	if err == nil {
		return ErrDuplicate
	}
	// credit and insert within a transaction
	tx, err := d.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err := tx.Exec(`UPDATE users SET credits = credits + $1 WHERE discord_id=$2`, amount, uid); err != nil {
		return err
	}
	if _, err := tx.Exec(`INSERT INTO ad_conversions(conversion_id, uid, amount, source) VALUES ($1,$2,$3,$4)`, conversionID, uid, amount, source); err != nil {
		return err
	}
	return tx.Commit()
}

// Server operations
func (d *DatabaseService) GetUserServers(discordID string) ([]*GameServer, error) {
	if d.db == nil {
		return []*GameServer{}, nil
	}

	rows, err := d.db.Query(`
		SELECT id, discord_id, name, game_type, status, COALESCE(address, ''), port, created_at,
		       stopped_at, last_heartbeat, cost_per_hour, COALESCE(is_public, false),
		       COALESCE(description, ''), COALESCE(error_message, ''), cleanup_at
		FROM game_servers WHERE discord_id = $1 ORDER BY created_at DESC
	`, discordID)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Failed to close rows: %v", err)
		}
	}()

	var servers []*GameServer
	for rows.Next() {
		server := &GameServer{}
		err := rows.Scan(&server.ID, &server.DiscordID, &server.Name, &server.GameType,
			&server.Status, &server.Address, &server.Port, &server.CreatedAt,
			&server.StoppedAt, &server.LastHeartbeat, &server.CostPerHour,
			&server.IsPublic, &server.Description, &server.ErrorMessage, &server.CleanupAt)
		if err != nil {
			return nil, err
		}
		servers = append(servers, server)
	}

	return servers, nil
}

func (d *DatabaseService) GetAllServers() ([]*GameServer, error) {
	if d.db == nil {
		return []*GameServer{}, nil
	}

	rows, err := d.db.Query(`
		SELECT id, discord_id, name, game_type, status, COALESCE(address, ''), port, created_at,
		       stopped_at, last_heartbeat, cost_per_hour, COALESCE(is_public, false),
		       COALESCE(description, ''), COALESCE(error_message, ''), cleanup_at
		FROM game_servers ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Failed to close rows: %v", err)
		}
	}()

	var servers []*GameServer
	for rows.Next() {
		server := &GameServer{}
		err := rows.Scan(&server.ID, &server.DiscordID, &server.Name, &server.GameType,
			&server.Status, &server.Address, &server.Port, &server.CreatedAt,
			&server.StoppedAt, &server.LastHeartbeat, &server.CostPerHour,
			&server.IsPublic, &server.Description, &server.ErrorMessage, &server.CleanupAt)
		if err != nil {
			return nil, err
		}
		servers = append(servers, server)
	}

	return servers, nil
}

func (d *DatabaseService) SaveGameServer(server *GameServer) error {
	if d.db == nil {
		return nil
	}

	_, err := d.db.Exec(`
		INSERT INTO game_servers (discord_id, name, game_type, status, address, port, cost_per_hour, is_public, description)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`, server.DiscordID, server.Name, server.GameType, server.Status, server.Address,
		server.Port, server.CostPerHour, server.IsPublic, server.Description)

	return err
}

func (d *DatabaseService) UpdateServerStatus(serverName, discordID, status string) error {
	if d.db == nil {
		return nil
	}

	_, err := d.db.Exec(`
		UPDATE game_servers SET status = $1 WHERE name = $2 AND discord_id = $3
	`, status, serverName, discordID)

	return err
}

func (d *DatabaseService) GetServerByName(serverName, discordID string) (*GameServer, error) {
	if d.db == nil {
		return nil, fmt.Errorf("database not available")
	}

	server := &GameServer{}
	err := d.db.QueryRow(`
		SELECT id, discord_id, name, game_type, status, COALESCE(address, ''), port, created_at, cost_per_hour,
		       COALESCE(is_public, false), COALESCE(description, '')
		FROM game_servers WHERE name = $1 AND discord_id = $2
	`, serverName, discordID).Scan(&server.ID, &server.DiscordID, &server.Name, &server.GameType,
		&server.Status, &server.Address, &server.Port, &server.CreatedAt,
		&server.CostPerHour, &server.IsPublic, &server.Description)

	if err != nil {
		return nil, err
	}
	return server, nil
}

// Public lobby operations
func (d *DatabaseService) AddToPublicLobby(server *GameServer, ownerName string) error {
	if d.db == nil {
		return nil
	}

	_, err := d.db.Exec(`
		INSERT INTO public_servers (server_name, game_type, owner_id, owner_name, address, port, description)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		ON CONFLICT (server_name) DO UPDATE SET
		address = EXCLUDED.address,
		port = EXCLUDED.port,
		description = EXCLUDED.description
	`, server.Name, server.GameType, server.DiscordID, ownerName, server.Address, server.Port, server.Description)

	// Also update the game_servers table
	_, err2 := d.db.Exec(`UPDATE game_servers SET is_public = true WHERE name = $1 AND discord_id = $2`,
		server.Name, server.DiscordID)

	if err != nil {
		return err
	}
	return err2
}

func (d *DatabaseService) RemoveFromPublicLobby(serverName, discordID string) error {
	if d.db == nil {
		return nil
	}

	_, err := d.db.Exec(`DELETE FROM public_servers WHERE server_name = $1 AND owner_id = $2`,
		serverName, discordID)

	// Also update the game_servers table
	_, err2 := d.db.Exec(`UPDATE game_servers SET is_public = false WHERE name = $1 AND discord_id = $2`,
		serverName, discordID)

	if err != nil {
		return err
	}
	return err2
}

func (d *DatabaseService) GetPublicServers() ([]*PublicServer, error) {
	if d.db == nil {
		return []*PublicServer{}, nil
	}

	rows, err := d.db.Query(`
		SELECT id, server_name, game_type, owner_id, owner_name, address, port,
		       COALESCE(description, ''), players, max_players, added_at
		FROM public_servers ORDER BY added_at DESC
	`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var servers []*PublicServer
	for rows.Next() {
		server := &PublicServer{}
		err := rows.Scan(&server.ID, &server.ServerName, &server.GameType, &server.OwnerID,
			&server.OwnerName, &server.Address, &server.Port, &server.Description,
			&server.Players, &server.MaxPlayers, &server.AddedAt)
		if err != nil {
			return nil, err
		}
		servers = append(servers, server)
	}

	return servers, nil
}

func (d *DatabaseService) RecordCommandUsage(discordID, command string) {
	if d.db == nil {
		return
	}

	_, err := d.db.Exec(`
		INSERT INTO command_usage (discord_id, command) VALUES ($1, $2)
	`, discordID, command)

	if err != nil {
		log.Printf("Failed to record command usage: %v", err)
	}
}

func (d *DatabaseService) UpdateServerPublicStatus(serverName, discordID string, isPublic bool) error {
	if d.db == nil {
		return nil
	}

	_, err := d.db.Exec(`
		UPDATE game_servers SET is_public = $1 WHERE name = $2 AND discord_id = $3
	`, isPublic, serverName, discordID)

	return err
}

// DeleteGameServer removes a game server from the database
func (d *DatabaseService) DeleteGameServer(serverID int) error {
	if d.db == nil {
		// In local mode, we would remove from memory, but we don't have local server storage implemented
		return nil
	}

	_, err := d.db.Exec("DELETE FROM game_servers WHERE id = $1", serverID)
	if err != nil {
		return fmt.Errorf("failed to delete server: %v", err)
	}

	log.Printf("🗑️ Deleted server with ID %d from database", serverID)
	return nil
}

// UpdateServerStatusWithDetails updates the status and related fields of a game server
func (d *DatabaseService) UpdateServerStatusWithDetails(serverID int, status string, errorMessage string) error {
	if d.db == nil {
		return nil // Skip in local mode
	}

	now := time.Now()

	// If status is "stopped", set StoppedAt timestamp
	if status == "stopped" {
		_, err := d.db.Exec(`
			UPDATE game_servers
			SET status = $1, error_message = $2, stopped_at = $3, last_heartbeat = $4
			WHERE id = $5
		`, status, errorMessage, now, now, serverID)
		return err
	}

	// For other statuses, update last_heartbeat
	_, err := d.db.Exec(`
		UPDATE game_servers
		SET status = $1, error_message = $2, last_heartbeat = $3
		WHERE id = $4
	`, status, errorMessage, now, serverID)
	return err
}

// ScheduleServerCleanup schedules a server for cleanup at a specific time
func (d *DatabaseService) ScheduleServerCleanup(serverID int, cleanupTime time.Time) error {
	if d.db == nil {
		return nil // Skip in local mode
	}

	_, err := d.db.Exec(`
		UPDATE game_servers
		SET cleanup_at = $1
		WHERE id = $2
	`, cleanupTime, serverID)

	if err != nil {
		return fmt.Errorf("failed to schedule cleanup: %v", err)
	}

	return nil
}

// GetStoppedServersForCleanup returns servers that have been stopped and are ready for cleanup
func (d *DatabaseService) GetStoppedServersForCleanup() ([]*GameServer, error) {
	if d.db == nil {
		return []*GameServer{}, nil
	}

	rows, err := d.db.Query(`
		SELECT id, discord_id, name, game_type, status, address, port,
			   created_at, stopped_at, last_heartbeat, cost_per_hour,
			   is_public, description, error_message, cleanup_at
		FROM game_servers
		WHERE status = 'stopped'
		  AND stopped_at IS NOT NULL
		  AND (cleanup_at IS NULL OR cleanup_at <= NOW())
		ORDER BY stopped_at ASC
	`)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Failed to close rows: %v", err)
		}
	}()

	var servers []*GameServer
	for rows.Next() {
		server := &GameServer{}
		err := rows.Scan(
			&server.ID, &server.DiscordID, &server.Name, &server.GameType,
			&server.Status, &server.Address, &server.Port, &server.CreatedAt,
			&server.StoppedAt, &server.LastHeartbeat, &server.CostPerHour,
			&server.IsPublic, &server.Description, &server.ErrorMessage,
			&server.CleanupAt,
		)
		if err != nil {
			return nil, err
		}
		servers = append(servers, server)
	}

	return servers, rows.Err()
}

// Role management methods

// AddBotRole adds a role to the bot's permission system
func (d *DatabaseService) AddBotRole(roleID, roleType, guildID string) error {
	if d.db == nil {
		return nil // Skip in local mode
	}

	_, err := d.db.Exec(`
		INSERT INTO bot_roles (role_id, role_type, guild_id)
		VALUES ($1, $2, $3)
		ON CONFLICT (role_id, guild_id) DO UPDATE SET
		role_type = EXCLUDED.role_type,
		added_at = CURRENT_TIMESTAMP
	`, roleID, roleType, guildID)

	return err
}

// RemoveBotRole removes a role from the bot's permission system
func (d *DatabaseService) RemoveBotRole(roleID, guildID string) error {
	if d.db == nil {
		return nil // Skip in local mode
	}

	_, err := d.db.Exec(`
		DELETE FROM bot_roles WHERE role_id = $1 AND guild_id = $2
	`, roleID, guildID)

	return err
}

// GetBotRoles returns all roles of a specific type for a guild
func (d *DatabaseService) GetBotRoles(roleType, guildID string) ([]string, error) {
	if d.db == nil {
		return []string{}, nil // Return empty slice in local mode
	}

	rows, err := d.db.Query(`
		SELECT role_id FROM bot_roles
		WHERE role_type = $1 AND guild_id = $2
		ORDER BY added_at ASC
	`, roleType, guildID)
	if err != nil {
		return nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Failed to close rows: %v", err)
		}
	}()

	var roles []string
	for rows.Next() {
		var roleID string
		if err := rows.Scan(&roleID); err != nil {
			return nil, err
		}
		roles = append(roles, roleID)
	}

	return roles, rows.Err()
}

// GetAllBotRoles returns all roles for a guild grouped by type
func (d *DatabaseService) GetAllBotRoles(guildID string) (adminRoles, modRoles []string, err error) {
	if d.db == nil {
		return []string{}, []string{}, nil // Return empty slices in local mode
	}

	rows, err := d.db.Query(`
		SELECT role_id, role_type FROM bot_roles
		WHERE guild_id = $1
		ORDER BY role_type, added_at ASC
	`, guildID)
	if err != nil {
		return nil, nil, err
	}
	defer func() {
		if err := rows.Close(); err != nil {
			log.Printf("Failed to close rows: %v", err)
		}
	}()

	for rows.Next() {
		var roleID, roleType string
		if err := rows.Scan(&roleID, &roleType); err != nil {
			return nil, nil, err
		}

		switch roleType {
		case "admin":
			adminRoles = append(adminRoles, roleID)
		case "moderator":
			modRoles = append(modRoles, roleID)
		}
	}

	return adminRoles, modRoles, rows.Err()
}

// UpdateServerError updates the error message for a server
func (d *DatabaseService) UpdateServerError(serverName, discordID, errorMessage string) error {
	if d.localMode {
		return nil // Local mode doesn't persist errors
	}

	_, err := d.db.Exec(`
		UPDATE game_servers 
		SET error_message = $1, updated_at = NOW() 
		WHERE name = $2 AND discord_id = $3`,
		errorMessage, serverName, discordID)
	return err
}

// UpdateServerKubernetesInfo updates the Kubernetes UID and Agones status
func (d *DatabaseService) UpdateServerKubernetesInfo(serverName, discordID, kubernetesUID, agonesStatus string) error {
	if d.localMode {
		return nil // Local mode doesn't persist Kubernetes info
	}

	_, err := d.db.Exec(`
		UPDATE game_servers 
		SET kubernetes_uid = $1, agones_status = $2, last_status_sync = NOW(), updated_at = NOW()
		WHERE name = $3 AND discord_id = $4`,
		kubernetesUID, agonesStatus, serverName, discordID)
	return err
}

// UpdateServerAgonesStatus updates the Agones status and last sync time
func (d *DatabaseService) UpdateServerAgonesStatus(serverName, discordID, agonesStatus string, syncTime *time.Time) error {
	if d.localMode {
		return nil // Local mode doesn't persist Agones status
	}

	_, err := d.db.Exec(`
		UPDATE game_servers 
		SET agones_status = $1, last_status_sync = $2, updated_at = NOW()
		WHERE name = $3 AND discord_id = $4`,
		agonesStatus, syncTime, serverName, discordID)
	return err
}

// GetServerStatus gets the current status of a server
func (d *DatabaseService) GetServerStatus(serverName, discordID string) string {
	if d.localMode {
		return "unknown" // Local mode doesn't track status
	}

	var status string
	err := d.db.QueryRow(`
		SELECT status FROM game_servers 
		WHERE name = $1 AND discord_id = $2`,
		serverName, discordID).Scan(&status)

	if err != nil {
		return "unknown"
	}
	return status
}

// UpdateServerAddress updates the server address and port
func (d *DatabaseService) UpdateServerAddress(serverName, discordID, address string, port int) error {
	if d.localMode {
		return nil // Local mode doesn't persist address info
	}

	_, err := d.db.Exec(`
		UPDATE game_servers 
		SET address = $1, port = $2, updated_at = NOW()
		WHERE name = $3 AND discord_id = $4`,
		address, port, serverName, discordID)
	return err
}
