package commands

import (
	"fmt"
	"strings"

	"agis-bot/internal/bot"
	"agis-bot/internal/services"

	"github.com/bwmarrin/discordgo"
)

type LogChannelCommand struct{}

func (c *LogChannelCommand) Name() string {
	return "log-channel"
}

func (c *LogChannelCommand) Description() string {
	return "Configure Discord logging channels"
}

func (c *LogChannelCommand) RequiredPermission() bot.Permission {
	return bot.PermissionAdmin
}

func (c *LogChannelCommand) Execute(ctx *CommandContext) error {
	if len(ctx.Args) == 0 {
		return c.showUsage(ctx)
	}

	subcommand := strings.ToLower(ctx.Args[0])

	switch subcommand {
	case "set":
		return c.setChannel(ctx)
	case "list":
		return c.listChannels(ctx)
	case "test":
		return c.testChannel(ctx)
	case "help":
		return c.showUsage(ctx)
	default:
		return c.showUsage(ctx)
	}
}

func (c *LogChannelCommand) setChannel(ctx *CommandContext) error {
	if len(ctx.Args) < 3 {
		embed := &discordgo.MessageEmbed{
			Title:       "❌ Invalid Usage",
			Description: "Usage: `log-channel set <category> <#channel>`\n\nCategories: general, user, mod, error, cluster, export, cleanup, audit",
			Color:       0xff0000,
		}
		_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
		return err
	}

	category := strings.ToLower(ctx.Args[1])
	channelMention := ctx.Args[2]

	// Extract channel ID from mention
	channelID := strings.TrimPrefix(channelMention, "<#")
	channelID = strings.TrimSuffix(channelID, ">")

	// Validate category
	validCategories := map[string]services.LogCategory{
		"general": services.LogCategoryBot,
		"user":    services.LogCategoryUser,
		"mod":     services.LogCategoryMod,
		"error":   services.LogCategoryError,
		"cluster": services.LogCategoryCluster,
		"export":  services.LogCategoryExport,
		"cleanup": services.LogCategoryCleanup,
		"audit":   services.LogCategoryAudit,
	}

	logCategory, valid := validCategories[category]
	if !valid {
		embed := &discordgo.MessageEmbed{
			Title:       "❌ Invalid Category",
			Description: "Valid categories: general, user, mod, error, cluster, export, cleanup, audit",
			Color:       0xff0000,
		}
		_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
		return err
	}

	// Verify channel exists and bot has access
	_, err := ctx.Session.Channel(channelID)
	if err != nil {
		embed := &discordgo.MessageEmbed{
			Title:       "❌ Channel Access Error",
			Description: "Cannot access the specified channel. Make sure the bot has permission to send messages there.",
			Color:       0xff0000,
		}
		_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
		return err
	}

	// Set the channel in the logging service
	if ctx.Logger != nil {
		ctx.Logger.SetLogChannel(logCategory, channelID)
	}

	embed := &discordgo.MessageEmbed{
		Title:       "✅ Log Channel Set",
		Description: fmt.Sprintf("**%s** logs will now be sent to <#%s>", strings.ToUpper(category[:1])+category[1:], channelID),
		Color:       0x00ff00,
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Note: This setting is temporary and will reset when the bot restarts. Update the deployment config for permanent settings.",
		},
	}

	_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}

func (c *LogChannelCommand) listChannels(ctx *CommandContext) error {
	embed := &discordgo.MessageEmbed{
		Title:       "📋 Discord Log Channels",
		Description: "Current log channel configuration:",
		Color:       0x0099ff,
		Fields:      []*discordgo.MessageEmbedField{},
	}

	if ctx.Logger != nil {
		config := ctx.Logger.GetChannelConfig()
		channels := map[string]string{
			"General": config.GeneralLogs,
			"User":    config.UserLogs,
			"Mod":     config.ModLogs,
			"Error":   config.ErrorLogs,
			"Cluster": config.ClusterLogs,
			"Export":  config.ExportLogs,
			"Cleanup": config.CleanupLogs,
			"Audit":   config.AuditLogs,
		}

		for category, channelID := range channels {
			value := "Not configured"
			if channelID != "" {
				value = fmt.Sprintf("<#%s>", channelID)
			}
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   category,
				Value:  value,
				Inline: true,
			})
		}
	}

	embed.Footer = &discordgo.MessageEmbedFooter{
		Text: "Use 'log-channel set <category> <#channel>' to configure channels",
	}

	_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}

func (c *LogChannelCommand) testChannel(ctx *CommandContext) error {
	if len(ctx.Args) < 2 {
		embed := &discordgo.MessageEmbed{
			Title:       "❌ Invalid Usage",
			Description: "Usage: `log-channel test <category>`\n\nCategories: general, user, mod, error, cluster, export, cleanup, audit",
			Color:       0xff0000,
		}
		_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
		return err
	}

	category := strings.ToLower(ctx.Args[1])

	// Send a test log entry
	if ctx.Logger != nil {
		switch category {
		case "general":
			ctx.Logger.LogBot(services.LogLevelInfo, "test_log", "Test log message for general channel", map[string]interface{}{
				"test":      true,
				"triggered": "admin_command",
				"user":      ctx.Message.Author.ID,
			})
		case "user":
			ctx.Logger.LogUser(ctx.Message.Author.ID, "test_log", "Test log message for user channel", map[string]interface{}{
				"test":      true,
				"triggered": "admin_command",
			})
		case "mod":
			ctx.Logger.LogMod(ctx.Message.Author.ID, "test_log", "Test log message for mod channel", map[string]interface{}{
				"test":      true,
				"triggered": "admin_command",
			})
		case "error":
			ctx.Logger.LogError("test_log", "Test log message for error channel", map[string]interface{}{
				"test":      true,
				"triggered": "admin_command",
				"user":      ctx.Message.Author.ID,
			})
		case "cluster":
			ctx.Logger.LogCluster("test_log", "Test log message for cluster channel", map[string]interface{}{
				"test":      true,
				"triggered": "admin_command",
				"user":      ctx.Message.Author.ID,
			})
		case "export":
			ctx.Logger.LogExport(ctx.Message.Author.ID, "test_log", "Test log message for export channel", map[string]interface{}{
				"test":      true,
				"triggered": "admin_command",
			})
		case "cleanup":
			ctx.Logger.LogCleanup("test_log", "Test log message for cleanup channel", map[string]interface{}{
				"test":      true,
				"triggered": "admin_command",
				"user":      ctx.Message.Author.ID,
			})
		case "audit":
			ctx.Logger.LogAudit(ctx.Message.Author.ID, "test_log", "Test log message for audit channel", map[string]interface{}{
				"test":      true,
				"triggered": "admin_command",
			})
		default:
			embed := &discordgo.MessageEmbed{
				Title:       "❌ Invalid Category",
				Description: "Valid categories: general, user, mod, error, cluster, export, cleanup, audit",
				Color:       0xff0000,
			}
			_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
			return err
		}
	}

	embed := &discordgo.MessageEmbed{
		Title:       "🧪 Test Log Sent",
		Description: fmt.Sprintf("Sent a test log entry for **%s** category. Check the configured channel and database for the log entry.", strings.ToUpper(category[:1])+category[1:]),
		Color:       0x00ff00,
	}

	_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}

func (c *LogChannelCommand) showUsage(ctx *CommandContext) error {
	embed := &discordgo.MessageEmbed{
		Title:       "📋 Log Channel Management",
		Description: "Configure Discord logging channels for the AGIS bot",
		Color:       0x0099ff,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "**📝 Available Commands**",
				Value:  "`log-channel list` - Show current channel configuration\n`log-channel set <category> <#channel>` - Set log channel\n`log-channel test <category>` - Send test log message",
				Inline: false,
			},
			{
				Name:   "**📊 Log Categories**",
				Value:  "• **general** - General system logs\n• **user** - User actions (server creation, deletion)\n• **mod** - Moderation actions\n• **error** - Error logs\n• **cluster** - Kubernetes cluster events\n• **export** - Save file exports\n• **cleanup** - Cleanup operations\n• **audit** - Security/audit events",
				Inline: false,
			},
			{
				Name:   "**💡 Examples**",
				Value:  "`log-channel set user #user-logs`\n`log-channel set error #error-logs`\n`log-channel test user`",
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Admin permission required • Changes are temporary until deployment is updated",
		},
	}

	_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}
