package bot

import (
	"strings"

	"agis-bot/internal/config"

	"github.com/bwmarrin/discordgo"
)

// Permission levels
type Permission int

const (
	PermissionUser Permission = iota
	PermissionGameServerMod     // Can moderate game servers
	PermissionCommunityAmbassador // Community engagement
	PermissionDiscordMod        // Discord moderation
	PermissionDiscordAdmin      // Discord administration
	PermissionBackendDev        // Backend developer access
	PermissionClusterAdmin      // Kubernetes cluster admin
	PermissionOwner             // Bot owner
)

// Bot owner Discord ID (hardcoded for security)
const BotOwnerID = "290955794172739584"

// PermissionChecker handles role-based permissions
type PermissionChecker struct {
	config *config.Config
}

func NewPermissionChecker(cfg *config.Config) *PermissionChecker {
	return &PermissionChecker{config: cfg}
}

// GetUserPermission determines the permission level of a user
func (p *PermissionChecker) GetUserPermission(s *discordgo.Session, guildID, userID string) Permission {
	// Check for bot owner first (highest priority)
	if userID == BotOwnerID {
		return PermissionOwner
	}

	// Get guild member to check roles
	member, err := s.GuildMember(guildID, userID)
	if err != nil {
		return PermissionUser // Default to user permission if we can't get member info
	}

	// Check roles in priority order (highest to lowest)
	for _, roleID := range member.Roles {
		if p.isRoleType(roleID, "cluster-admin") {
			return PermissionClusterAdmin
		}
	}

	for _, roleID := range member.Roles {
		if p.isRoleType(roleID, "backend-dev") {
			return PermissionBackendDev
		}
	}

	for _, roleID := range member.Roles {
		if p.isRoleType(roleID, "discord-admin") || p.isAdminRole(roleID) {
			return PermissionDiscordAdmin
		}
	}

	for _, roleID := range member.Roles {
		if p.isRoleType(roleID, "discord-mod") || p.isModRole(roleID) {
			return PermissionDiscordMod
		}
	}

	for _, roleID := range member.Roles {
		if p.isRoleType(roleID, "community-ambassador") {
			return PermissionCommunityAmbassador
		}
	}

	for _, roleID := range member.Roles {
		if p.isRoleType(roleID, "gameserver-mod") {
			return PermissionGameServerMod
		}
	}

	return PermissionUser
}

// HasPermission checks if user has required permission level
func (p *PermissionChecker) HasPermission(s *discordgo.Session, guildID, userID string, required Permission) bool {
	userPerm := p.GetUserPermission(s, guildID, userID)
	return userPerm >= required
}

// IsOwner checks if user is the bot owner
func (p *PermissionChecker) IsOwner(userID string) bool {
	return userID == BotOwnerID
}

// IsAdmin checks if user has admin permissions (DiscordAdmin or higher)
func (p *PermissionChecker) IsAdmin(s *discordgo.Session, guildID, userID string) bool {
	return p.HasPermission(s, guildID, userID, PermissionDiscordAdmin)
}

// IsMod checks if user has mod permissions or higher (DiscordMod or higher)
func (p *PermissionChecker) IsMod(s *discordgo.Session, guildID, userID string) bool {
	return p.HasPermission(s, guildID, userID, PermissionDiscordMod)
}

func (p *PermissionChecker) isAdminRole(roleID string) bool {
	for _, adminRole := range p.config.Roles.AdminRoles {
		if strings.EqualFold(roleID, adminRole) {
			return true
		}
	}
	return false
}

func (p *PermissionChecker) isModRole(roleID string) bool {
	for _, modRole := range p.config.Roles.ModRoles {
		if strings.EqualFold(roleID, modRole) {
			return true
		}
	}
	return false
}

// isRoleType checks if a role matches a specific type in the database
func (p *PermissionChecker) isRoleType(roleID, roleType string) bool {
	// This is a placeholder - actual implementation would query the database
	// For now, we'll use config-based checking
	// TODO: Implement database role lookup
	return false
}

// IsVerified checks if user has the verified member role
func (p *PermissionChecker) IsVerified(s *discordgo.Session, guildID, userID string) bool {
	if p.config.Roles.VerifiedRoleID == "" {
		return false
	}
	member, err := s.GuildMember(guildID, userID)
	if err != nil {
		return false
	}
	for _, roleID := range member.Roles {
		if strings.EqualFold(roleID, p.config.Roles.VerifiedRoleID) {
			return true
		}
	}
	return false
}

// GetPermissionString returns a human-readable permission level
func GetPermissionString(perm Permission) string {
	switch perm {
	case PermissionOwner:
		return "Owner"
	case PermissionClusterAdmin:
		return "Cluster Admin"
	case PermissionBackendDev:
		return "Backend Developer"
	case PermissionDiscordAdmin:
		return "Discord Admin"
	case PermissionDiscordMod:
		return "Discord Moderator"
	case PermissionCommunityAmbassador:
		return "Community Ambassador"
	case PermissionGameServerMod:
		return "Game Server Mod"
	default:
		return "User"
	}
}
