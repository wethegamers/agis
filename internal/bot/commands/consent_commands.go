package commands

import (
	"context"
	"fmt"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/wethegamers/agis-bot/internal/bot"
	"github.com/wethegamers/agis-bot/internal/services"
)

// ConsentCommand allows users to give/revoke ad consent (GDPR compliance)
type ConsentCommand struct {
	consentService *services.ConsentService
}

func NewConsentCommand(consentService *services.ConsentService) *ConsentCommand {
	return &ConsentCommand{consentService: consentService}
}

func (c *ConsentCommand) Name() string {
	return "consent"
}

func (c *ConsentCommand) Description() string {
	return "Give or manage consent for viewing ads (required for EU users)"
}

func (c *ConsentCommand) Permission() bot.Permission {
	return bot.PermissionUser
}

func (c *ConsentCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	ctx := context.Background()
	userID := bot.GetUserID(i)

	// Check if user already has consent
	consentStatus, err := c.consentService.GetConsentStatus(ctx, userID)
	if err != nil {
		return bot.RespondError(s, i, "Failed to check consent status")
	}

	// If already consented, show current status
	if consentStatus != nil && consentStatus.Consented && consentStatus.WithdrawnTimestamp == nil {
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: fmt.Sprintf("✅ You have already given consent to view ads.\n\n"+
					"**Consent given:** <t:%d:F>\n"+
					"**Country:** %s\n\n"+
					"To withdraw consent, use `/consent-withdraw`",
					consentStatus.ConsentTimestamp.Unix(),
					consentStatus.IPCountry,
				),
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
	}

	// Show consent prompt with buttons
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: services.GetConsentPromptText(),
			Flags:   discordgo.MessageFlagsEphemeral,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "✅ I Accept",
							Style:    discordgo.SuccessButton,
							CustomID: "consent_accept",
						},
						discordgo.Button{
							Label:    "❌ I Decline",
							Style:    discordgo.DangerButton,
							CustomID: "consent_decline",
						},
						discordgo.Button{
							Label: "📄 Privacy Policy",
							Style: discordgo.LinkButton,
							URL:   services.GetPrivacyPolicyURL(),
						},
					},
				},
			},
		},
	})
}

// ConsentAcceptHandler handles the consent accept button
func ConsentAcceptHandler(consentService *services.ConsentService) func(*discordgo.Session, *discordgo.InteractionCreate) {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		ctx := context.Background()
		userID := bot.GetUserID(i)

		// TODO: In production, detect user's country from IP or Discord locale
		// For now, default to "US" - non-EU country
		userCountry := "US"
		if i.Locale != "" {
			// Extract country from locale (e.g., "en-GB" -> "GB")
			if len(i.Locale) >= 5 {
				userCountry = i.Locale[len(i.Locale)-2:]
			}
		}

		// Record consent
		err := consentService.RecordConsent(ctx, userID, true, userCountry, "discord_command")
		if err != nil {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "❌ Failed to record consent. Please try again.",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}

		// Update message to show success
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content: "✅ **Consent Recorded**\n\n" +
					"Thank you for giving consent! You can now earn Game Credits by watching ads.\n\n" +
					"Use `/watch-ad` to start earning.\n" +
					"Use `/consent-withdraw` to withdraw consent at any time.",
				Components: []discordgo.MessageComponent{}, // Remove buttons
				Flags:      discordgo.MessageFlagsEphemeral,
			},
		})
	}
}

// ConsentDeclineHandler handles the consent decline button
func ConsentDeclineHandler(consentService *services.ConsentService) func(*discordgo.Session, *discordgo.InteractionCreate) {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		ctx := context.Background()
		userID := bot.GetUserID(i)

		// Record decline
		userCountry := "US"
		if i.Locale != "" && len(i.Locale) >= 5 {
			userCountry = i.Locale[len(i.Locale)-2:]
		}

		err := consentService.RecordConsent(ctx, userID, false, userCountry, "discord_command")
		if err != nil {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseChannelMessageWithSource,
				Data: &discordgo.InteractionResponseData{
					Content: "❌ Failed to record decline. Please try again.",
					Flags:   discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}

		// Update message to show decline confirmation
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content: "ℹ️ **Consent Declined**\n\n" +
					"You have declined consent for ad viewing. You will not be able to earn Game Credits through ads.\n\n" +
					"You can change your mind at any time by using `/consent` again.",
				Components: []discordgo.MessageComponent{}, // Remove buttons
				Flags:      discordgo.MessageFlagsEphemeral,
			},
		})
	}
}

// ConsentStatusCommand shows user's current consent status
type ConsentStatusCommand struct {
	consentService *services.ConsentService
}

func NewConsentStatusCommand(consentService *services.ConsentService) *ConsentStatusCommand {
	return &ConsentStatusCommand{consentService: consentService}
}

func (c *ConsentStatusCommand) Name() string {
	return "consent-status"
}

func (c *ConsentStatusCommand) Description() string {
	return "View your current ad consent status"
}

func (c *ConsentStatusCommand) Permission() bot.Permission {
	return bot.PermissionUser
}

func (c *ConsentStatusCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	ctx := context.Background()
	userID := bot.GetUserID(i)

	consentStatus, err := c.consentService.GetConsentStatus(ctx, userID)
	if err != nil {
		return bot.RespondError(s, i, "Failed to retrieve consent status")
	}

	if consentStatus == nil {
		return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseChannelMessageWithSource,
			Data: &discordgo.InteractionResponseData{
				Content: "ℹ️ You have not given or declined consent yet.\n\n" +
					"Use `/consent` to give consent and start earning Game Credits through ads.",
				Flags: discordgo.MessageFlagsEphemeral,
			},
		})
	}

	// Build status message
	var statusEmoji, statusText, timestamp string
	if consentStatus.WithdrawnTimestamp != nil {
		statusEmoji = "⚠️"
		statusText = "**Withdrawn**"
		timestamp = fmt.Sprintf("Withdrawn: <t:%d:F>", consentStatus.WithdrawnTimestamp.Unix())
	} else if consentStatus.Consented {
		statusEmoji = "✅"
		statusText = "**Active**"
		timestamp = fmt.Sprintf("Consented: <t:%d:F>", consentStatus.ConsentTimestamp.Unix())
	} else {
		statusEmoji = "❌"
		statusText = "**Declined**"
		timestamp = fmt.Sprintf("Declined: <t:%d:F>", consentStatus.CreatedAt.Unix())
	}

	message := fmt.Sprintf("%s **Ad Consent Status:** %s\n\n"+
		"**Country:** %s\n"+
		"**GDPR Version:** %s\n"+
		"**Method:** %s\n"+
		"%s\n\n",
		statusEmoji, statusText,
		consentStatus.IPCountry,
		consentStatus.GDPRVersion,
		consentStatus.ConsentMethod,
		timestamp,
	)

	if consentStatus.Consented && consentStatus.WithdrawnTimestamp == nil {
		message += "You can withdraw consent at any time using `/consent-withdraw`"
	} else {
		message += "Use `/consent` to give consent and start earning through ads."
	}

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: message,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}

// ConsentWithdrawCommand allows users to withdraw consent (GDPR right)
type ConsentWithdrawCommand struct {
	consentService *services.ConsentService
}

func NewConsentWithdrawCommand(consentService *services.ConsentService) *ConsentWithdrawCommand {
	return &ConsentWithdrawCommand{consentService: consentService}
}

func (c *ConsentWithdrawCommand) Name() string {
	return "consent-withdraw"
}

func (c *ConsentWithdrawCommand) Description() string {
	return "Withdraw your consent for ad viewing (GDPR right)"
}

func (c *ConsentWithdrawCommand) Permission() bot.Permission {
	return bot.PermissionUser
}

func (c *ConsentWithdrawCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	ctx := context.Background()
	userID := bot.GetUserID(i)

	// Show confirmation prompt
	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: "⚠️ **Withdraw Ad Consent?**\n\n" +
				"This will:\n" +
				"• Disable your ability to earn Game Credits through ads\n" +
				"• Be effective immediately\n\n" +
				"You can re-consent at any time using `/consent`.\n\n" +
				"**Are you sure you want to withdraw consent?**",
			Flags: discordgo.MessageFlagsEphemeral,
			Components: []discordgo.MessageComponent{
				discordgo.ActionsRow{
					Components: []discordgo.MessageComponent{
						discordgo.Button{
							Label:    "✅ Yes, Withdraw",
							Style:    discordgo.DangerButton,
							CustomID: "consent_withdraw_confirm",
						},
						discordgo.Button{
							Label:    "❌ Cancel",
							Style:    discordgo.SecondaryButton,
							CustomID: "consent_withdraw_cancel",
						},
					},
				},
			},
		},
	})
}

// ConsentWithdrawConfirmHandler handles consent withdrawal confirmation
func ConsentWithdrawConfirmHandler(consentService *services.ConsentService) func(*discordgo.Session, *discordgo.InteractionCreate) {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		ctx := context.Background()
		userID := bot.GetUserID(i)

		err := consentService.WithdrawConsent(ctx, userID)
		if err != nil {
			s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
				Type: discordgo.InteractionResponseUpdateMessage,
				Data: &discordgo.InteractionResponseData{
					Content:    fmt.Sprintf("❌ Failed to withdraw consent: %s", err.Error()),
					Components: []discordgo.MessageComponent{},
					Flags:      discordgo.MessageFlagsEphemeral,
				},
			})
			return
		}

		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content: "✅ **Consent Withdrawn**\n\n" +
					"Your consent for ad viewing has been withdrawn.\n\n" +
					"You will no longer be able to earn Game Credits through ads.\n" +
					"You can re-consent at any time using `/consent`.",
				Components: []discordgo.MessageComponent{},
				Flags:      discordgo.MessageFlagsEphemeral,
			},
		})
	}
}

// ConsentWithdrawCancelHandler handles consent withdrawal cancellation
func ConsentWithdrawCancelHandler() func(*discordgo.Session, *discordgo.InteractionCreate) {
	return func(s *discordgo.Session, i *discordgo.InteractionCreate) {
		s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
			Type: discordgo.InteractionResponseUpdateMessage,
			Data: &discordgo.InteractionResponseData{
				Content:    "ℹ️ Consent withdrawal cancelled. Your consent remains active.",
				Components: []discordgo.MessageComponent{},
				Flags:      discordgo.MessageFlagsEphemeral,
			},
		})
	}
}

// ConsentStatsCommand shows admin consent statistics (GDPR compliance reporting)
type ConsentStatsCommand struct {
	consentService *services.ConsentService
}

func NewConsentStatsCommand(consentService *services.ConsentService) *ConsentStatsCommand {
	return &ConsentStatsCommand{consentService: consentService}
}

func (c *ConsentStatsCommand) Name() string {
	return "consent-stats"
}

func (c *ConsentStatsCommand) Description() string {
	return "View GDPR consent statistics (Admin only)"
}

func (c *ConsentStatsCommand) Permission() bot.Permission {
	return bot.PermissionAdmin
}

func (c *ConsentStatsCommand) Execute(s *discordgo.Session, i *discordgo.InteractionCreate) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	stats, err := c.consentService.GetConsentStats(ctx)
	if err != nil {
		return bot.RespondError(s, i, "Failed to retrieve consent statistics")
	}

	message := fmt.Sprintf("📊 **GDPR Consent Statistics**\n\n"+
		"**Overall:**\n"+
		"• Total Users: %d\n"+
		"• Consented: %d (%.1f%%)\n"+
		"• Withdrawn: %d (%.1f%%)\n\n"+
		"**EU/EEA Users (GDPR Required):**\n"+
		"• EU Users: %d\n"+
		"• EU Consented: %d (%.1f%%)\n"+
		"• Non-EU Users: %d\n\n"+
		"**Recent Activity (24h):**\n"+
		"• New Consents: %d\n"+
		"• New Withdrawals: %d\n\n"+
		"_Stats updated: <t:%d:R>_",
		stats.TotalUsers,
		stats.ConsentedUsers, stats.ConsentRate,
		stats.WithdrawnUsers, stats.WithdrawalRate,
		stats.EUUsers,
		stats.EUConsentedUsers, stats.EUConsentRate,
		stats.NonEUUsers,
		stats.RecentConsents24h,
		stats.RecentWithdrawals24h,
		time.Now().Unix(),
	)

	return s.InteractionRespond(i.Interaction, &discordgo.InteractionResponse{
		Type: discordgo.InteractionResponseChannelMessageWithSource,
		Data: &discordgo.InteractionResponseData{
			Content: message,
			Flags:   discordgo.MessageFlagsEphemeral,
		},
	})
}
