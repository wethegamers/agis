package services

import (
	"database/sql"
	"log"
	"time"

	"github.com/bwmarrin/discordgo"
)

// RoleSyncService periodically syncs verified roles from database to Discord
type RoleSyncService struct {
	db             *sql.DB
	session        *discordgo.Session
	guildID        string
	verifiedRoleID string
	interval       time.Duration
	stopChan       chan struct{}
}

// NewRoleSyncService creates a new role sync service
func NewRoleSyncService(db *sql.DB, session *discordgo.Session, guildID, verifiedRoleID string, interval time.Duration) *RoleSyncService {
	return &RoleSyncService{
		db:             db,
		session:        session,
		guildID:        guildID,
		verifiedRoleID: verifiedRoleID,
		interval:       interval,
		stopChan:       make(chan struct{}),
	}
}

// Start begins the periodic role sync
func (rs *RoleSyncService) Start() {
	log.Printf("✅ Role sync service started (interval: %v)", rs.interval)
	
	// Run immediately on start
	rs.syncRoles()
	
	ticker := time.NewTicker(rs.interval)
	defer ticker.Stop()
	
	for {
		select {
		case <-ticker.C:
			rs.syncRoles()
		case <-rs.stopChan:
			log.Println("🛑 Role sync service stopped")
			return
		}
	}
}

// Stop stops the role sync service
func (rs *RoleSyncService) Stop() {
	close(rs.stopChan)
}

// syncRoles checks database for verified users and ensures they have the Discord role
func (rs *RoleSyncService) syncRoles() {
	if rs.verifiedRoleID == "" {
		log.Println("[RoleSync] Verified role ID not configured, skipping sync")
		return
	}
	
	if rs.guildID == "" {
		log.Println("[RoleSync] Guild ID not configured, skipping sync")
		return
	}
	
	log.Println("[RoleSync] Starting role sync...")
	
	// Query all users who have bot_access_granted in WordPress (via audit logs or separate table)
	// For now, we'll query users from the verification audit logs
	query := `
		SELECT DISTINCT user_id 
		FROM audit_logs 
		WHERE event_type = 'user_verified' 
		AND created_at > NOW() - INTERVAL '90 days'
	`
	
	rows, err := rs.db.Query(query)
	if err != nil {
		log.Printf("[RoleSync] ERROR querying verified users: %v", err)
		return
	}
	defer rows.Close()
	
	verifiedUsers := make([]string, 0)
	for rows.Next() {
		var userID string
		if err := rows.Scan(&userID); err != nil {
			log.Printf("[RoleSync] ERROR scanning user ID: %v", err)
			continue
		}
		verifiedUsers = append(verifiedUsers, userID)
	}
	
	if len(verifiedUsers) == 0 {
		log.Println("[RoleSync] No verified users found in database")
		return
	}
	
	log.Printf("[RoleSync] Found %d verified users in database", len(verifiedUsers))
	
	// Get all members in the guild
	members, err := rs.getAllGuildMembers()
	if err != nil {
		log.Printf("[RoleSync] ERROR fetching guild members: %v", err)
		return
	}
	
	log.Printf("[RoleSync] Checking %d guild members", len(members))
	
	synced := 0
	skipped := 0
	errors := 0
	
	// Check each verified user
	for _, userID := range verifiedUsers {
		member := rs.findMember(members, userID)
		if member == nil {
			// User not in server, skip
			skipped++
			continue
		}
		
		// Check if member already has verified role
		hasRole := rs.hasRole(member, rs.verifiedRoleID)
		if hasRole {
			skipped++
			continue
		}
		
		// Add verified role
		err := rs.session.GuildMemberRoleAdd(rs.guildID, userID, rs.verifiedRoleID)
		if err != nil {
			log.Printf("[RoleSync] ERROR adding verified role to user %s (%s): %v", member.User.Username, userID, err)
			errors++
			continue
		}
		
		log.Printf("[RoleSync] ✅ Added verified role to %s#%s (%s)", member.User.Username, member.User.Discriminator, userID)
		synced++
		
		// Rate limit: Discord allows 50 requests per second, but be conservative
		time.Sleep(100 * time.Millisecond)
	}
	
	log.Printf("[RoleSync] Sync complete: %d synced, %d skipped, %d errors", synced, skipped, errors)
}

// getAllGuildMembers fetches all members from the guild (handles pagination)
func (rs *RoleSyncService) getAllGuildMembers() ([]*discordgo.Member, error) {
	allMembers := make([]*discordgo.Member, 0)
	after := ""
	
	for {
		members, err := rs.session.GuildMembers(rs.guildID, after, 1000)
		if err != nil {
			return nil, err
		}
		
		if len(members) == 0 {
			break
		}
		
		allMembers = append(allMembers, members...)
		
		// If we got less than 1000, we've reached the end
		if len(members) < 1000 {
			break
		}
		
		// Set after to last user ID for pagination
		after = members[len(members)-1].User.ID
	}
	
	return allMembers, nil
}

// findMember finds a member by user ID in the members slice
func (rs *RoleSyncService) findMember(members []*discordgo.Member, userID string) *discordgo.Member {
	for _, member := range members {
		if member.User.ID == userID {
			return member
		}
	}
	return nil
}

// hasRole checks if a member has a specific role
func (rs *RoleSyncService) hasRole(member *discordgo.Member, roleID string) bool {
	for _, r := range member.Roles {
		if r == roleID {
			return true
		}
	}
	return false
}
