package commands

import (
	"fmt"
	"strings"

	"agis-bot/internal/bot"
	"agis-bot/internal/services"

	"github.com/bwmarrin/discordgo"
)

// ModServersCommand lists all servers for moderation purposes
type ModServersCommand struct{}

func (c *ModServersCommand) Name() string {
	return "mod-servers"
}

func (c *ModServersCommand) Description() string {
	return "List all user servers (moderator view)"
}

func (c *ModServersCommand) RequiredPermission() bot.Permission {
	return bot.PermissionMod
}

func (c *ModServersCommand) Execute(ctx *CommandContext) error {
	servers, err := ctx.DB.GetAllServers()
	if err != nil {
		return fmt.Errorf("failed to get all servers: %v", err)
	}

	if len(servers) == 0 {
		embed := &discordgo.MessageEmbed{
			Title:       "🛡️ Moderator View - All Servers",
			Description: "No servers found in the system.",
			Color:       0x4169e1,
		}
		_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
		return err
	}

	// Group servers by user
	userServers := make(map[string][]*services.GameServer)

	for _, server := range servers {
		userServers[server.DiscordID] = append(userServers[server.DiscordID], server)
	}

	var fields []*discordgo.MessageEmbedField
	totalServers := 0
	runningServers := 0

	for userID, servers := range userServers {
		// Get user info from Discord
		user, err := ctx.Session.User(userID)
		var userName string
		if err != nil {
			userName = fmt.Sprintf("Unknown User (%s)", userID[:8])
		} else {
			userName = user.Username
		}

		var serverInfo []string
		for _, server := range servers {
			totalServers++
			statusEmoji := "❓"
			switch server.Status {
			case "running", "ready":
				statusEmoji = "✅"
				runningServers++
			case "stopped":
				statusEmoji = "⏸️"
			case "creating":
				statusEmoji = "⏳"
			case "stopping":
				statusEmoji = "⏹️"
			}

			publicStatus := ""
			if server.IsPublic {
				publicStatus = " 🌐"
			}

			serverInfo = append(serverInfo, fmt.Sprintf("%s **%s** (%s)%s",
				statusEmoji, server.Name, server.GameType, publicStatus))
		}

		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   fmt.Sprintf("👤 %s (%d servers)", userName, len(servers)),
			Value:  strings.Join(serverInfo, "\n"),
			Inline: false,
		})
	}

	embed := &discordgo.MessageEmbed{
		Title:  "🛡️ Moderator View - All Servers",
		Color:  0x4169e1,
		Fields: fields,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Total: %d servers • %d running • Use 'mod-control <user> <server>' to manage",
				totalServers, runningServers),
		},
	}

	_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}

// ModControlCommand allows mods to control user servers
type ModControlCommand struct{}

func (c *ModControlCommand) Name() string {
	return "mod-control"
}

func (c *ModControlCommand) Description() string {
	return "Control user servers (moderator only)"
}

func (c *ModControlCommand) RequiredPermission() bot.Permission {
	return bot.PermissionMod
}

func (c *ModControlCommand) Execute(ctx *CommandContext) error {
	if len(ctx.Args) < 3 {
		embed := &discordgo.MessageEmbed{
			Title:       "🛡️ Moderator Server Control",
			Description: "Manage user servers as a moderator",
			Color:       0x4169e1,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:  "Usage",
					Value: "`mod-control <user> <server> <action>`",
				},
				{
					Name:  "Actions",
					Value: "• `stop` - Stop the server\n• `restart` - Restart the server\n• `info` - Get detailed info\n• `logs` - Get recent logs",
				},
				{
					Name:  "Examples",
					Value: "`mod-control @user minecraft1 stop`\n`mod-control john terraria info`",
				},
			},
		}
		_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
		return err
	}

	targetUser := ctx.Args[0]
	serverName := ctx.Args[1]
	action := ctx.Args[2]

	// Try to resolve user mention to user ID
	var userID string
	if strings.HasPrefix(targetUser, "<@") && strings.HasSuffix(targetUser, ">") {
		userID = strings.Trim(targetUser, "<@!>")
	} else {
		// Try to find user by username
		guild, err := ctx.Session.Guild(ctx.Message.GuildID)
		if err == nil {
			for _, member := range guild.Members {
				if strings.EqualFold(member.User.Username, targetUser) {
					userID = member.User.ID
					break
				}
			}
		}
		if userID == "" {
			embed := &discordgo.MessageEmbed{
				Title:       "❌ User Not Found",
				Description: fmt.Sprintf("Could not find user: %s\nTry using @mention or exact username", targetUser),
				Color:       0xff0000,
			}
			_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
			return err
		}
	}

	// Get the server
	server, err := ctx.DB.GetServerByName(serverName, userID)
	if err != nil {
		embed := &discordgo.MessageEmbed{
			Title:       "❌ Server Not Found",
			Description: fmt.Sprintf("Server '%s' not found for user", serverName),
			Color:       0xff0000,
		}
		_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
		return err
	}

	// Get user info for logging
	user, _ := ctx.Session.User(userID)
	userName := "Unknown User"
	if user != nil {
		userName = user.Username
	}

	switch strings.ToLower(action) {
	case "stop":
		// Here you would integrate with your Kubernetes API to stop the server
		// For now, simulate the action
		err := ctx.DB.UpdateServerStatus(server.Name, server.DiscordID, "stopping")
		if err != nil {
			return err
		}

		embed := &discordgo.MessageEmbed{
			Title:       "🛡️ Server Stop Initiated",
			Description: fmt.Sprintf("Stopping server **%s** owned by **%s**", serverName, userName),
			Color:       0xff9900,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:  "Action By",
					Value: fmt.Sprintf("%s (Moderator)", ctx.Message.Author.Username),
				},
				{
					Name:  "Server",
					Value: fmt.Sprintf("%s (%s)", serverName, server.GameType),
				},
			},
		}
		_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
		return err

	case "restart":
		embed := &discordgo.MessageEmbed{
			Title:       "🛡️ Server Restart Initiated",
			Description: fmt.Sprintf("Restarting server **%s** owned by **%s**", serverName, userName),
			Color:       0xff9900,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:  "Action By",
					Value: fmt.Sprintf("%s (Moderator)", ctx.Message.Author.Username),
				},
				{
					Name:  "Server",
					Value: fmt.Sprintf("%s (%s)", serverName, server.GameType),
				},
			},
		}
		_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
		return err

	case "info":
		embed := &discordgo.MessageEmbed{
			Title: "🛡️ Server Information",
			Color: 0x4169e1,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "Server Name",
					Value:  server.Name,
					Inline: true,
				},
				{
					Name:   "Owner",
					Value:  userName,
					Inline: true,
				},
				{
					Name:   "Game Type",
					Value:  server.GameType,
					Inline: true,
				},
				{
					Name:   "Status",
					Value:  server.Status,
					Inline: true,
				},
				{
					Name:   "Cost/Hour",
					Value:  fmt.Sprintf("%d credits", server.CostPerHour),
					Inline: true,
				},
				{
					Name:   "Public",
					Value:  fmt.Sprintf("%t", server.IsPublic),
					Inline: true,
				},
			},
		}
		if server.Address != "" {
			embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
				Name:   "Address",
				Value:  fmt.Sprintf("%s:%d", server.Address, server.Port),
				Inline: true,
			})
		}
		_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
		return err

	case "logs":
		embed := &discordgo.MessageEmbed{
			Title:       "🛡️ Server Logs",
			Description: fmt.Sprintf("Recent logs for **%s** (owned by **%s**)", serverName, userName),
			Color:       0x4169e1,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:  "Log Output",
					Value: "```\n[2024-01-20 10:30:15] Server started\n[2024-01-20 10:30:16] Listening on port 25565\n[2024-01-20 10:32:45] Player joined: TestPlayer\n[2024-01-20 10:45:12] Player left: TestPlayer\n```",
				},
			},
			Footer: &discordgo.MessageEmbedFooter{
				Text: "Last 10 lines • Use kubectl for full logs",
			},
		}
		_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
		return err

	default:
		embed := &discordgo.MessageEmbed{
			Title:       "❌ Invalid Action",
			Description: fmt.Sprintf("Unknown action: %s\nValid actions: stop, restart, info, logs", action),
			Color:       0xff0000,
		}
		_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
		return err
	}
}
