package commands

import (
	"fmt"
	"strings"
	"time"

	"agis-bot/internal/bot"

	"github.com/bwmarrin/discordgo"
)

type ServersCommand struct{}

func (c *ServersCommand) Name() string {
	return "servers"
}

func (c *ServersCommand) Description() string {
	return "List your game servers"
}

func (c *ServersCommand) RequiredPermission() bot.Permission {
	return bot.PermissionUser
}

func (c *ServersCommand) Execute(ctx *CommandContext) error {
	servers, err := ctx.DB.GetUserServers(ctx.Message.Author.ID)
	if err != nil {
		return fmt.Errorf("failed to get servers: %v", err)
	}

	if len(servers) == 0 {
		embed := &discordgo.MessageEmbed{
			Title:       "🎮 Your Game Servers",
			Description: "You don't have any servers yet! Ready to create your first one?",
			Color:       0xffa500,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "🚀 Get Started",
					Value:  "Use `create <game>` to deploy a server\nExample: `create minecraft`",
					Inline: false,
				},
				{
					Name:   "🎲 Available Games",
					Value:  "• Minecraft\n• CS2\n• Terraria\n• Garry's Mod",
					Inline: true,
				},
				{
					Name:   "💰 Check Credits",
					Value:  "Use `credits` to see your balance",
					Inline: true,
				},
			},
			Footer: &discordgo.MessageEmbedFooter{
				Text: "💡 Tip: Most servers take 2-5 minutes to deploy. Need help? Try 'help'",
			},
		}
		_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
		return err
	}

	var fields []*discordgo.MessageEmbedField
	for _, server := range servers {
		var statusEmoji, statusText, helpText string

		switch server.Status {
		case "creating":
			statusEmoji = "⏳"
			statusText = "Starting Up"
			helpText = "⏳ **Server is deploying** - Check back in 2-3 minutes"
		case "running", "ready":
			statusEmoji = "✅"
			statusText = "Online"
			helpText = "✅ **Ready to play!** - Use `diagnostics " + server.Name + "` for details"
		case "stopped":
			statusEmoji = "⏸️"
			statusText = "Stopped"
			helpText = "⏸️ **Server paused** - Contact support if this persists"
		case "stopping":
			statusEmoji = "⏹️"
			statusText = "Stopping"
			helpText = "⏹️ **Server shutting down** - Will be ready to restart soon"
		case "error":
			statusEmoji = "❌"
			statusText = "Error"
			helpText = "❌ **Server encountered an error** - Use `diagnostics " + server.Name + "` for details"
		default:
			statusEmoji = "❓"
			statusText = "Unknown"
			helpText = "❓ **Status unclear** - Use `diagnostics " + server.Name + "` for details"
		}

		value := fmt.Sprintf("%s **%s**\n**Type:** %s\n**Cost:** %d credits/hour",
			statusEmoji, statusText, strings.Title(server.GameType), server.CostPerHour)

		// Add connection info for running servers
		if server.Address != "" && (server.Status == "running" || server.Status == "ready") {
			value += fmt.Sprintf("\n🌐 **Connect:** `%s:%d`", server.Address, server.Port)
		}

		// Add error message if present
		if server.ErrorMessage != "" {
			value += fmt.Sprintf("\n⚠️ **Error:** %s", server.ErrorMessage)
		}

		// Add lifespan information
		now := time.Now()
		uptime := now.Sub(server.CreatedAt)
		value += fmt.Sprintf("\n⏰ **Uptime:** %s", formatDuration(uptime))

		// Add cleanup information for stopped servers
		if server.Status == "stopped" && server.StoppedAt != nil {
			stoppedDuration := now.Sub(*server.StoppedAt)
			value += fmt.Sprintf("\n⏹️ **Stopped:** %s ago", formatDuration(stoppedDuration))

			// Calculate cleanup time (2 hours after stopped)
			cleanupTime := server.StoppedAt.Add(2 * time.Hour)
			if now.Before(cleanupTime) {
				timeUntilCleanup := cleanupTime.Sub(now)
				value += fmt.Sprintf("\n🧹 **Cleanup in:** %s", formatDuration(timeUntilCleanup))
				value += "\n💾 **Tip:** Use `export " + server.Name + "` to save your data"
			} else {
				value += "\n⚠️ **Scheduled for cleanup** - Export saves now!"
			}
		}

		if server.IsPublic {
			value += "\n🌐 **Listed in Public Lobby**"
		}

		value += fmt.Sprintf("\n%s", helpText)

		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   server.Name,
			Value:  value,
			Inline: true,
		})
	}

	// Count servers by status for footer info
	running := 0
	stopped := 0
	creating := 0
	for _, server := range servers {
		switch server.Status {
		case "running", "ready":
			running++
		case "stopped":
			stopped++
		case "creating":
			creating++
		}
	}

	footerText := fmt.Sprintf("Total: %d servers", len(servers))
	if creating > 0 {
		footerText += fmt.Sprintf(" • %d starting up", creating)
	}
	if running > 0 {
		footerText += fmt.Sprintf(" • %d online", running)
	}
	if stopped > 0 {
		footerText += fmt.Sprintf(" • %d offline", stopped)
	}

	embed := &discordgo.MessageEmbed{
		Title:  "🎮 Your Game Servers",
		Color:  0x00ff00,
		Fields: fields,
		Footer: &discordgo.MessageEmbedFooter{
			Text: footerText + " • Use 'diagnostics <name>' for details",
		},
	}

	_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}

// formatDuration formats a duration into a human-readable string
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.0f seconds", d.Seconds())
	} else if d < time.Hour {
		return fmt.Sprintf("%.0f minutes", d.Minutes())
	} else if d < 24*time.Hour {
		hours := int(d.Hours())
		minutes := int((d - time.Duration(hours)*time.Hour).Minutes())
		if minutes == 0 {
			return fmt.Sprintf("%d hours", hours)
		}
		return fmt.Sprintf("%d hours, %d minutes", hours, minutes)
	} else {
		days := int(d.Hours() / 24)
		hours := int(d.Hours()) % 24
		if hours == 0 {
			return fmt.Sprintf("%d days", days)
		}
		return fmt.Sprintf("%d days, %d hours", days, hours)
	}
}
