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
	PermissionMod
	PermissionAdmin
	PermissionOwner
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

	// Check for admin roles
	for _, roleID := range member.Roles {
		if p.isAdminRole(roleID) {
			return PermissionAdmin
		}
	}

	// Check for mod roles
	for _, roleID := range member.Roles {
		if p.isModRole(roleID) {
			return PermissionMod
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

// IsAdmin checks if user has admin permissions
func (p *PermissionChecker) IsAdmin(s *discordgo.Session, guildID, userID string) bool {
	return p.HasPermission(s, guildID, userID, PermissionAdmin)
}

// IsMod checks if user has mod permissions or higher
func (p *PermissionChecker) IsMod(s *discordgo.Session, guildID, userID string) bool {
	return p.HasPermission(s, guildID, userID, PermissionMod)
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

// GetPermissionString returns a human-readable permission level
func GetPermissionString(perm Permission) string {
	switch perm {
	case PermissionOwner:
		return "Owner"
	case PermissionAdmin:
		return "Admin"
	case PermissionMod:
		return "Moderator"
	default:
		return "User"
	}
}
