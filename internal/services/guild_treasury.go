package services

import (
	"database/sql"
	"fmt"
	"log"
	"time"
)

// GuildTreasuryService manages guild shared wallets for server funding
type GuildTreasuryService struct {
	db *DatabaseService
}

// NewGuildTreasuryService creates a new guild treasury service
func NewGuildTreasuryService(db *DatabaseService) *GuildTreasuryService {
	return &GuildTreasuryService{db: db}
}

// GuildTreasury represents a guild's shared wallet
type GuildTreasury struct {
	ID            int
	GuildID       string
	GuildName     string
	OwnerID       string
	Balance       int       // GameCredits balance
	TotalDeposits int       // All-time deposits
	TotalSpent    int       // All-time spending
	MemberCount   int       // Current member count
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

// GuildMember represents a member's contribution to a guild
type GuildMember struct {
	GuildID       string
	DiscordID     string
	Username      string
	TotalDeposits int       // Lifetime contributions
	LastDeposit   time.Time
	JoinedAt      time.Time
	Role          string    // 'owner', 'admin', 'member'
}

// CreateGuild creates a new guild treasury
func (g *GuildTreasuryService) CreateGuild(guildID, guildName, ownerID string) error {
	if g.db.LocalMode() {
		return fmt.Errorf("guild treasury not available in local mode")
	}

	tx, err := g.db.DB().Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	// Create guild treasury
	_, err = tx.Exec(`
		INSERT INTO guild_treasury (guild_id, guild_name, owner_id, balance, total_deposits, total_spent, member_count, created_at, updated_at)
		VALUES ($1, $2, $3, 0, 0, 0, 1, CURRENT_TIMESTAMP, CURRENT_TIMESTAMP)
		ON CONFLICT (guild_id) DO NOTHING
	`, guildID, guildName, ownerID)
	if err != nil {
		return fmt.Errorf("failed to create guild: %v", err)
	}

	// Add owner as first member
	_, err = tx.Exec(`
		INSERT INTO guild_members (guild_id, discord_id, total_deposits, joined_at, role)
		VALUES ($1, $2, 0, CURRENT_TIMESTAMP, 'owner')
		ON CONFLICT (guild_id, discord_id) DO NOTHING
	`, guildID, ownerID)
	if err != nil {
		return fmt.Errorf("failed to add owner as member: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %v", err)
	}

	log.Printf("✅ Guild created: %s (%s) by %s", guildName, guildID, ownerID)
	return nil
}

// GetGuild retrieves guild treasury information
func (g *GuildTreasuryService) GetGuild(guildID string) (*GuildTreasury, error) {
	if g.db.LocalMode() {
		return nil, fmt.Errorf("guild treasury not available in local mode")
	}

	var guild GuildTreasury
	err := g.db.DB().QueryRow(`
		SELECT id, guild_id, guild_name, owner_id, balance, total_deposits, total_spent, member_count, created_at, updated_at
		FROM guild_treasury
		WHERE guild_id = $1
	`, guildID).Scan(
		&guild.ID, &guild.GuildID, &guild.GuildName, &guild.OwnerID,
		&guild.Balance, &guild.TotalDeposits, &guild.TotalSpent,
		&guild.MemberCount, &guild.CreatedAt, &guild.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, fmt.Errorf("guild not found")
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get guild: %v", err)
	}

	return &guild, nil
}

// DepositToGuild deposits GameCredits from user's personal wallet to guild treasury
// WARNING: Non-refundable. Once deposited, credits belong to the guild.
func (g *GuildTreasuryService) DepositToGuild(guildID, discordID string, amount int) error {
	if g.db.LocalMode() {
		return fmt.Errorf("guild treasury not available in local mode")
	}

	if amount <= 0 {
		return fmt.Errorf("deposit amount must be positive")
	}

	tx, err := g.db.DB().Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	// Check user balance
	var userBalance int
	err = tx.QueryRow(`SELECT credits FROM users WHERE discord_id = $1`, discordID).Scan(&userBalance)
	if err != nil {
		return fmt.Errorf("failed to get user balance: %v", err)
	}

	if userBalance < amount {
		return fmt.Errorf("insufficient credits. You have %d GC, trying to deposit %d GC", userBalance, amount)
	}

	// Check guild exists
	var guildExists bool
	err = tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM guild_treasury WHERE guild_id = $1)`, guildID).Scan(&guildExists)
	if err != nil {
		return fmt.Errorf("failed to check guild: %v", err)
	}
	if !guildExists {
		return fmt.Errorf("guild not found. Ask guild owner to create it first")
	}

	// Check membership
	var isMember bool
	err = tx.QueryRow(`SELECT EXISTS(SELECT 1 FROM guild_members WHERE guild_id = $1 AND discord_id = $2)`, guildID, discordID).Scan(&isMember)
	if err != nil {
		return fmt.Errorf("failed to check membership: %v", err)
	}
	if !isMember {
		return fmt.Errorf("you are not a member of this guild. Ask to be invited first")
	}

	// Deduct from user
	_, err = tx.Exec(`UPDATE users SET credits = credits - $1 WHERE discord_id = $2`, amount, discordID)
	if err != nil {
		return fmt.Errorf("failed to deduct credits from user: %v", err)
	}

	// Add to guild treasury
	_, err = tx.Exec(`
		UPDATE guild_treasury 
		SET balance = balance + $1, total_deposits = total_deposits + $1, updated_at = CURRENT_TIMESTAMP
		WHERE guild_id = $2
	`, amount, guildID)
	if err != nil {
		return fmt.Errorf("failed to add credits to guild: %v", err)
	}

	// Update member's contribution record
	_, err = tx.Exec(`
		UPDATE guild_members 
		SET total_deposits = total_deposits + $1, last_deposit = CURRENT_TIMESTAMP
		WHERE guild_id = $2 AND discord_id = $3
	`, amount, guildID, discordID)
	if err != nil {
		return fmt.Errorf("failed to update member record: %v", err)
	}

	// Log transaction
	_, err = tx.Exec(`
		INSERT INTO credit_transactions (from_user, to_user, amount, transaction_type, description, currency_type)
		VALUES ($1, NULL, $2, 'guild_deposit', 'Deposit to guild treasury', 'GC')
	`, discordID, amount)
	if err != nil {
		log.Printf("Warning: Failed to log transaction: %v", err)
		// Non-fatal - continue
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit deposit: %v", err)
	}

	log.Printf("💰 Guild deposit: %s deposited %d GC to guild %s", discordID, amount, guildID)
	return nil
}

// SpendFromGuild deducts credits from guild treasury (e.g., for server costs)
// Only guild owners/admins can authorize spending
func (g *GuildTreasuryService) SpendFromGuild(guildID, authorizedBy, reason string, amount int) error {
	if g.db.LocalMode() {
		return fmt.Errorf("guild treasury not available in local mode")
	}

	if amount <= 0 {
		return fmt.Errorf("spend amount must be positive")
	}

	tx, err := g.db.DB().Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	// Check guild balance
	var balance int
	err = tx.QueryRow(`SELECT balance FROM guild_treasury WHERE guild_id = $1`, guildID).Scan(&balance)
	if err == sql.ErrNoRows {
		return fmt.Errorf("guild not found")
	}
	if err != nil {
		return fmt.Errorf("failed to get guild balance: %v", err)
	}

	if balance < amount {
		return fmt.Errorf("insufficient guild funds. Treasury has %d GC, trying to spend %d GC", balance, amount)
	}

	// Verify authorization (owner or admin)
	var memberRole string
	err = tx.QueryRow(`SELECT role FROM guild_members WHERE guild_id = $1 AND discord_id = $2`, guildID, authorizedBy).Scan(&memberRole)
	if err == sql.ErrNoRows {
		return fmt.Errorf("you are not a member of this guild")
	}
	if err != nil {
		return fmt.Errorf("failed to check authorization: %v", err)
	}

	if memberRole != "owner" && memberRole != "admin" {
		return fmt.Errorf("only guild owners and admins can spend from treasury")
	}

	// Deduct from guild
	_, err = tx.Exec(`
		UPDATE guild_treasury 
		SET balance = balance - $1, total_spent = total_spent + $1, updated_at = CURRENT_TIMESTAMP
		WHERE guild_id = $2
	`, amount, guildID)
	if err != nil {
		return fmt.Errorf("failed to deduct from guild treasury: %v", err)
	}

	// Log transaction
	_, err = tx.Exec(`
		INSERT INTO credit_transactions (from_user, to_user, amount, transaction_type, description, currency_type)
		VALUES (NULL, $1, $2, 'guild_spend', $3, 'GC')
	`, authorizedBy, amount, reason)
	if err != nil {
		log.Printf("Warning: Failed to log transaction: %v", err)
		// Non-fatal - continue
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit spending: %v", err)
	}

	log.Printf("💸 Guild spend: %s authorized %d GC spending from guild %s for: %s", authorizedBy, amount, guildID, reason)
	return nil
}

// GetGuildMembers retrieves all members of a guild
func (g *GuildTreasuryService) GetGuildMembers(guildID string) ([]*GuildMember, error) {
	if g.db.LocalMode() {
		return []*GuildMember{}, nil
	}

	rows, err := g.db.DB().Query(`
		SELECT gm.guild_id, gm.discord_id, u.username, gm.total_deposits, gm.last_deposit, gm.joined_at, gm.role
		FROM guild_members gm
		JOIN users u ON gm.discord_id = u.discord_id
		WHERE gm.guild_id = $1
		ORDER BY gm.total_deposits DESC, gm.joined_at ASC
	`, guildID)
	if err != nil {
		return nil, fmt.Errorf("failed to get guild members: %v", err)
	}
	defer rows.Close()

	var members []*GuildMember
	for rows.Next() {
		member := &GuildMember{}
		var lastDeposit sql.NullTime
		err := rows.Scan(
			&member.GuildID, &member.DiscordID, &member.Username,
			&member.TotalDeposits, &lastDeposit, &member.JoinedAt, &member.Role,
		)
		if err != nil {
			continue
		}
		if lastDeposit.Valid {
			member.LastDeposit = lastDeposit.Time
		}
		members = append(members, member)
	}

	return members, nil
}

// AddMember adds a user to a guild
func (g *GuildTreasuryService) AddMember(guildID, discordID, invitedBy string) error {
	if g.db.LocalMode() {
		return fmt.Errorf("guild treasury not available in local mode")
	}

	tx, err := g.db.DB().Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	// Check inviter is owner or admin
	var inviterRole string
	err = tx.QueryRow(`SELECT role FROM guild_members WHERE guild_id = $1 AND discord_id = $2`, guildID, invitedBy).Scan(&inviterRole)
	if err == sql.ErrNoRows {
		return fmt.Errorf("you are not a member of this guild")
	}
	if err != nil {
		return fmt.Errorf("failed to check permissions: %v", err)
	}

	if inviterRole != "owner" && inviterRole != "admin" {
		return fmt.Errorf("only guild owners and admins can invite members")
	}

	// Add member
	_, err = tx.Exec(`
		INSERT INTO guild_members (guild_id, discord_id, total_deposits, joined_at, role)
		VALUES ($1, $2, 0, CURRENT_TIMESTAMP, 'member')
		ON CONFLICT (guild_id, discord_id) DO NOTHING
	`, guildID, discordID)
	if err != nil {
		return fmt.Errorf("failed to add member: %v", err)
	}

	// Increment member count
	_, err = tx.Exec(`UPDATE guild_treasury SET member_count = member_count + 1 WHERE guild_id = $1`, guildID)
	if err != nil {
		return fmt.Errorf("failed to update member count: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit: %v", err)
	}

	log.Printf("👥 Guild member added: %s joined guild %s (invited by %s)", discordID, guildID, invitedBy)
	return nil
}

// GetUserGuilds retrieves all guilds a user is a member of
func (g *GuildTreasuryService) GetUserGuilds(discordID string) ([]*GuildTreasury, error) {
	if g.db.LocalMode() {
		return []*GuildTreasury{}, nil
	}

	rows, err := g.db.DB().Query(`
		SELECT gt.id, gt.guild_id, gt.guild_name, gt.owner_id, gt.balance, gt.total_deposits, gt.total_spent, gt.member_count, gt.created_at, gt.updated_at
		FROM guild_treasury gt
		JOIN guild_members gm ON gt.guild_id = gm.guild_id
		WHERE gm.discord_id = $1
		ORDER BY gt.created_at DESC
	`, discordID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user guilds: %v", err)
	}
	defer rows.Close()

	var guilds []*GuildTreasury
	for rows.Next() {
		guild := &GuildTreasury{}
		err := rows.Scan(
			&guild.ID, &guild.GuildID, &guild.GuildName, &guild.OwnerID,
			&guild.Balance, &guild.TotalDeposits, &guild.TotalSpent,
			&guild.MemberCount, &guild.CreatedAt, &guild.UpdatedAt,
		)
		if err != nil {
			continue
		}
		guilds = append(guilds, guild)
	}

	return guilds, nil
}
