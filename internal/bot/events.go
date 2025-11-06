package bot

import (
	"fmt"
	"log"

	"agis-bot/internal/services"

	"github.com/bwmarrin/discordgo"
)

// EventHandlers manages Discord event handlers
type EventHandlers struct {
	loggingService *services.LoggingService
	verifiedRoleID string
	guildID        string
}

// NewEventHandlers creates a new event handlers instance
func NewEventHandlers(loggingService *services.LoggingService, verifiedRoleID, guildID string) *EventHandlers {
	return &EventHandlers{
		loggingService: loggingService,
		verifiedRoleID: verifiedRoleID,
		guildID:        guildID,
	}
}

// HandleGuildMemberUpdate monitors role changes and makes verified role sticky
// If a verified user has their role removed, it will be automatically re-added
func (eh *EventHandlers) HandleGuildMemberUpdate(s *discordgo.Session, event *discordgo.GuildMemberUpdate) {
	log.Printf("[DEBUG] GuildMemberUpdate event received for user %s in guild %s", event.User.ID, event.GuildID)
	
	// Skip if verified role is not configured
	if eh.verifiedRoleID == "" {
		log.Printf("[DEBUG] Verified role ID not configured, skipping")
		return
	}

	// Skip if not in configured guild
	if event.GuildID != eh.guildID {
		log.Printf("[DEBUG] Event from different guild (%s vs %s), skipping", event.GuildID, eh.guildID)
		return
	}
	
	log.Printf("[DEBUG] BeforeUpdate: %+v", event.BeforeUpdate)
	log.Printf("[DEBUG] Current roles: %v", event.Member.Roles)

	// Check if user previously had verified role
	hadVerifiedRole := false
	for _, roleID := range event.BeforeUpdate.Roles {
		if roleID == eh.verifiedRoleID {
			hadVerifiedRole = true
			break
		}
	}

	// Check if user currently has verified role
	hasVerifiedRole := false
	for _, roleID := range event.Member.Roles {
		if roleID == eh.verifiedRoleID {
			hasVerifiedRole = true
			break
		}
	}

	// If user had verified role but now doesn't, re-add it (make it sticky)
	if hadVerifiedRole && !hasVerifiedRole {
		userTag := "Unknown"
		if event.User != nil {
			userTag = fmt.Sprintf("%s#%s", event.User.Username, event.User.Discriminator)
		}

		log.Printf("[RoleProtection] Verified role removed from %s (%s), re-adding (sticky)", userTag, event.User.ID)

		// Re-add the verified role
		err := s.GuildMemberRoleAdd(event.GuildID, event.User.ID, eh.verifiedRoleID)
		if err != nil {
			log.Printf("[RoleProtection] ERROR: Failed to re-add verified role to %s: %v", event.User.ID, err)
			
			// Log the failure to audit channel
			if eh.loggingService != nil {
				eh.loggingService.LogAudit(
					event.User.ID,
					"verified_role_protection_failed",
					fmt.Sprintf("Failed to re-add verified role to user %s", userTag),
					map[string]interface{}{
						"user_id":  event.User.ID,
						"username": userTag,
						"error":    err.Error(),
					},
				)
			}
			return
		}

		log.Printf("[RoleProtection] Successfully re-added verified role to %s", userTag)

		// Log successful protection to audit channel
		if eh.loggingService != nil {
			eh.loggingService.LogAudit(
				event.User.ID,
				"verified_role_protected",
				fmt.Sprintf("✅ Verified role automatically restored for %s", userTag),
				map[string]interface{}{
					"user_id":  event.User.ID,
					"username": userTag,
					"action":   "role_restored",
					"reason":   "sticky_verified_role",
				},
			)
		}

		// Optionally send a DM to the user notifying them
		// (Commented out to avoid potential spam)
		/*
		dmChannel, err := s.UserChannelCreate(event.User.ID)
		if err == nil {
			_, _ = s.ChannelMessageSend(dmChannel.ID, 
				"✅ Your verified status has been automatically restored. The verified role cannot be removed.")
		}
		*/
	}
}
