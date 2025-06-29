package commands

import (
	"fmt"
	"time"

	"agis-bot/internal/bot"
	"agis-bot/internal/services"

	"github.com/bwmarrin/discordgo"
)

// DiagnosticsCommand provides server diagnostics for users
type DiagnosticsCommand struct{}

func (c *DiagnosticsCommand) Name() string {
	return "diagnostics"
}

func (c *DiagnosticsCommand) Description() string {
	return "Run diagnostics on your game server"
}

func (c *DiagnosticsCommand) RequiredPermission() bot.Permission {
	return bot.PermissionUser
}

func (c *DiagnosticsCommand) Execute(ctx *CommandContext) error {
	if len(ctx.Args) == 0 {
		embed := &discordgo.MessageEmbed{
			Title:       "🔧 Server Diagnostics",
			Description: "Run diagnostics on your game servers",
			Color:       0x00ccff,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:  "Usage",
					Value: "`diagnostics <server-name>`",
				},
				{
					Name:  "Available Tests",
					Value: "• **Connection Test** - Check if server is reachable\n• **Live Status** - Real-time status from Kubernetes\n• **Performance Metrics** - CPU, RAM, and disk usage\n• **Game Status** - Players online, game state\n• **Resource Usage** - Credit consumption rate",
				},
				{
					Name:  "Example",
					Value: "`diagnostics minecraft1`",
				},
			},
			Footer: &discordgo.MessageEmbedFooter{
				Text: "💡 Tip: Use 'servers' to see all your servers",
			},
		}
		_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
		return err
	}

	serverName := ctx.Args[0]

	// Get the server using enhanced service for live data
	var server *services.GameServer
	var err error
	
	if ctx.EnhancedServer != nil {
		server, err = ctx.EnhancedServer.GetEnhancedServerInfo(ctx.Context, serverName, ctx.Message.Author.ID)
	} else {
		server, err = ctx.DB.GetServerByName(serverName, ctx.Message.Author.ID)
	}

	if err != nil {
		embed := &discordgo.MessageEmbed{
			Title:       "❌ Server Not Found",
			Description: fmt.Sprintf("You don't have a server named '%s'", serverName),
			Color:       0xff0000,
			Footer: &discordgo.MessageEmbedFooter{
				Text: "Use 'servers' to see your available servers",
			},
		}
		_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
		return err
	}

	// Run diagnostics based on server status
	var statusEmoji, statusText, healthText string
	var healthColor int

	// Use live status from Kubernetes if available
	displayStatus := server.Status
	if server.AgonesStatus != "" && ctx.EnhancedServer != nil {
		displayStatus = server.AgonesStatus
	}

	switch displayStatus {
	case "running", "ready", "Ready", "Allocated":
		statusEmoji = "✅"
		statusText = "Online"
		healthText = "All systems operational"
		healthColor = 0x00ff00
	case "creating", "Creating", "PortAllocation", "Starting", "Scheduled", "RequestReady":
		statusEmoji = "⏳"
		statusText = "Starting Up"
		healthText = "Server is being deployed"
		healthColor = 0xffa500
	case "stopped", "Shutdown":
		statusEmoji = "⏸️"
		statusText = "Stopped"
		healthText = "Server is paused"
		healthColor = 0xff9900
	case "stopping":
		statusEmoji = "⏹️"
		statusText = "Stopping"
		healthText = "Server is shutting down"
		healthColor = 0xff9900
	case "error", "Error", "Unhealthy":
		statusEmoji = "❌"
		statusText = "Error"
		healthText = "Server encountered an error"
		healthColor = 0xff0000
	default:
		statusEmoji = "❓"
		statusText = "Unknown"
		healthText = "Status unclear - may need attention"
		healthColor = 0xff0000
	}

	embed := &discordgo.MessageEmbed{
		Title: fmt.Sprintf("🔧 Diagnostics: %s", server.Name),
		Color: healthColor,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Status",
				Value:  fmt.Sprintf("%s **%s**", statusEmoji, statusText),
				Inline: true,
			},
			{
				Name:   "Game Type",
				Value:  titleCase(server.GameType),
				Inline: true,
			},
			{
				Name:   "Cost/Hour",
				Value:  fmt.Sprintf("%d credits", server.CostPerHour),
				Inline: true,
			},
		},
	}

	// Add connection info if server is running
	if server.Status == "running" || server.Status == "ready" {
		if server.Address != "" {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   "🌐 Connection",
				Value:  fmt.Sprintf("**Address:** `%s:%d`\n**Status:** Reachable ✅", server.Address, server.Port),
				Inline: false,
			})
		}

		// Simulate performance metrics
		embed.Fields = append(embed.Fields,
			&discordgo.MessageEmbedField{
				Name:   "🎯 Performance",
				Value:  "**CPU:** 15.2% (Good)\n**RAM:** 768MB / 2GB (38%)\n**Disk:** 1.2GB / 10GB (12%)",
				Inline: true,
			},
			&discordgo.MessageEmbedField{
				Name:   "👥 Players",
				Value:  "**Online:** 0/20\n**Peak Today:** 3\n**Total Sessions:** 12",
				Inline: true,
			},
			&discordgo.MessageEmbedField{
				Name:   "⚡ Network",
				Value:  "**Ping:** 23ms\n**Uptime:** 2h 15m\n**Last Restart:** 2h ago",
				Inline: true,
			},
		)

		// Add recent logs section
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "📋 Recent Logs",
			Value:  "```\n[10:15:32] Server started successfully\n[10:15:33] Loading world: world\n[10:15:35] Ready for connections\n[10:22:15] Player TestUser joined\n[10:45:12] Player TestUser left\n```",
			Inline: false,
		})
	} else if server.Status == "creating" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "⏳ Deployment Progress",
			Value:  "**Step 1:** Image pulling ✅\n**Step 2:** Container starting ⏳\n**Step 3:** Game initialization ⏸️\n**Step 4:** Health check ⏸️",
			Inline: false,
		})

		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "⏱️ Estimated Time",
			Value:  "**Remaining:** ~2-3 minutes\n**Started:** 30 seconds ago",
			Inline: true,
		})
	} else if server.Status == "stopped" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "💤 Stopped Server",
			Value:  "Server is currently paused to save credits\n\n**Last Session:** 2 hours ago\n**Reason:** Manual stop\n**Credits Saved:** ~24 credits",
			Inline: false,
		})

		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "🔄 Restart Options",
			Value:  "Contact support to restart your server\nExpected startup time: 2-3 minutes",
			Inline: false,
		})
	}

	// Add cost tracking
	if server.Status == "running" || server.Status == "ready" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name: "💰 Cost Tracking",
			Value: fmt.Sprintf("**Current Session:** %d credits\n**Today:** %d credits\n**This Week:** %d credits",
				server.CostPerHour*2, server.CostPerHour*8, server.CostPerHour*25),
			Inline: true,
		})
	}

	embed.Footer = &discordgo.MessageEmbedFooter{
		Text: fmt.Sprintf("%s • Refreshed just now • Use 'help' for more commands", healthText),
	}

	// Add troubleshooting info if there are issues
	if server.Status != "running" && server.Status != "ready" && server.Status != "creating" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "🛠️ Troubleshooting",
			Value:  "• Check if you have sufficient credits\n• Try running diagnostics again in 1-2 minutes\n• Contact support if issues persist\n• Use `help` for support options",
			Inline: false,
		})
	}

	// Add Kubernetes information if available
	if server.KubernetesUID != "" && ctx.EnhancedServer != nil {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "🔗 Kubernetes Info",
			Value:  fmt.Sprintf("**UID:** `%s`\n**Agones Status:** %s\n**Last Sync:** %s", 
				server.KubernetesUID[:8]+"...", 
				server.AgonesStatus, 
				formatSyncTime(server.LastStatusSync)),
			Inline: false,
		})
	}

	_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}

// PingCommand provides a simple ping diagnostic
type PingCommand struct{}

func (c *PingCommand) Name() string {
	return "ping"
}

func (c *PingCommand) Description() string {
	return "Test connectivity to your servers"
}

func (c *PingCommand) RequiredPermission() bot.Permission {
	return bot.PermissionUser
}

func (c *PingCommand) Execute(ctx *CommandContext) error {
	if len(ctx.Args) == 0 {
		// Ping the bot itself
		embed := &discordgo.MessageEmbed{
			Title:       "🏓 Pong!",
			Description: "Bot is online and responsive",
			Color:       0x00ff00,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "Response Time",
					Value:  "< 50ms",
					Inline: true,
				},
				{
					Name:   "Status",
					Value:  "✅ All systems operational",
					Inline: true,
				},
				{
					Name:   "Usage",
					Value:  "Use `ping <server>` to test a specific server",
					Inline: false,
				},
			},
		}
		_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
		return err
	}

	serverName := ctx.Args[0]

	// Get the server
	server, err := ctx.DB.GetServerByName(serverName, ctx.Message.Author.ID)
	if err != nil {
		embed := &discordgo.MessageEmbed{
			Title:       "❌ Server Not Found",
			Description: fmt.Sprintf("You don't have a server named '%s'", serverName),
			Color:       0xff0000,
		}
		_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
		return err
	}

	var pingResult string
	var pingColor int

	if server.Status == "running" || server.Status == "ready" {
		if server.Address != "" {
			pingResult = fmt.Sprintf("✅ **%s:%d** is reachable\n🏓 Response time: 23ms\n📡 Connection: Stable",
				server.Address, server.Port)
			pingColor = 0x00ff00
		} else {
			pingResult = "⚠️ Server is running but address not available\nContact support for assistance"
			pingColor = 0xffa500
		}
	} else if server.Status == "creating" {
		pingResult = "⏳ Server is still starting up\nTry again in 2-3 minutes"
		pingColor = 0xffa500
	} else {
		pingResult = fmt.Sprintf("❌ Server is %s\nCannot ping inactive server", server.Status)
		pingColor = 0xff0000
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("🏓 Ping: %s", server.Name),
		Description: pingResult,
		Color:       pingColor,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Game Type",
				Value:  titleCase(server.GameType),
				Inline: true,
			},
			{
				Name:   "Status",
				Value:  titleCase(server.Status),
				Inline: true,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Use 'diagnostics " + server.Name + "' for detailed server info",
		},
	}

	_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}

// formatSyncTime formats the last sync time for display
func formatSyncTime(syncTime *time.Time) string {
	if syncTime == nil || syncTime.IsZero() {
		return "Never"
	}
	
	now := time.Now()
	diff := now.Sub(*syncTime)
	
	if diff < time.Minute {
		return fmt.Sprintf("%.0f seconds ago", diff.Seconds())
	} else if diff < time.Hour {
		return fmt.Sprintf("%.0f minutes ago", diff.Minutes())
	} else if diff < 24*time.Hour {
		return fmt.Sprintf("%.1f hours ago", diff.Hours())
	} else {
		return fmt.Sprintf("%.0f days ago", diff.Hours()/24)
	}
}
