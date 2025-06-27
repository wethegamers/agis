package commands

import (
	"fmt"
	"strings"

	"agis-bot/internal/bot"
	"agis-bot/internal/services"

	"github.com/bwmarrin/discordgo"
)

// PublicLobbyCommand manages public lobby functionality
type PublicLobbyCommand struct{}

func (c *PublicLobbyCommand) Name() string {
	return "lobby"
}

func (c *PublicLobbyCommand) Description() string {
	return "Manage public lobby settings"
}

func (c *PublicLobbyCommand) RequiredPermission() bot.Permission {
	return bot.PermissionUser
}

func (c *PublicLobbyCommand) Execute(ctx *CommandContext) error {
	if len(ctx.Args) == 0 {
		// Show lobby help and current listings
		return c.showLobbyHelp(ctx)
	}

	subcommand := strings.ToLower(ctx.Args[0])

	switch subcommand {
	case "list":
		return c.listPublicServers(ctx)
	case "add":
		return c.addToLobby(ctx)
	case "remove":
		return c.removeFromLobby(ctx)
	case "my":
		return c.showMyPublicServers(ctx)
	default:
		return c.showLobbyHelp(ctx)
	}
}

func (c *PublicLobbyCommand) showLobbyHelp(ctx *CommandContext) error {
	embed := &discordgo.MessageEmbed{
		Title:       "🌐 WTG Public Lobby",
		Description: "Share your servers with the WTG community!",
		Color:       0x00ccff,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "📋 Commands",
				Value:  "`lobby list` - Browse public servers\n`lobby add <server>` - Share your server\n`lobby remove <server>` - Remove from lobby\n`lobby my` - View your public servers",
				Inline: false,
			},
			{
				Name:   "✨ Benefits",
				Value:  "• **Attract Players** - Get more people on your server\n• **Community Building** - Connect with other WTG members\n• **Server Discovery** - Help others find great servers",
				Inline: false,
			},
			{
				Name:   "📜 Rules",
				Value:  "• Server must be online and stable\n• Keep description family-friendly\n• No advertising or spam\n• Follow WTG community guidelines",
				Inline: false,
			},
			{
				Name:   "🎮 What Gets Listed",
				Value:  "Your server name, game type, current players, and connection details will be visible to all WTG members.",
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "💡 Tip: Public servers get more visibility and can attract new players!",
		},
	}

	_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}

func (c *PublicLobbyCommand) listPublicServers(ctx *CommandContext) error {
	publicServers, err := ctx.DB.GetPublicServers()
	if err != nil {
		return fmt.Errorf("failed to get public servers: %v", err)
	}

	if len(publicServers) == 0 {
		embed := &discordgo.MessageEmbed{
			Title:       "🌐 WTG Public Lobby",
			Description: "No public servers available right now.",
			Color:       0xffa500,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:  "🚀 Be the First!",
					Value: "Share your server with the community using `lobby add <server-name>`",
				},
				{
					Name:  "🎮 Available Games",
					Value: "Minecraft • CS2 • Terraria • Garry's Mod • And more!",
				},
			},
			Footer: &discordgo.MessageEmbedFooter{
				Text: "Check back later or create your own server to add to the lobby!",
			},
		}
		_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
		return err
	}

	// Group servers by game type for better organization
	gameTypes := make(map[string][]*services.PublicServer)
	for _, server := range publicServers {
		gameTypes[server.GameType] = append(gameTypes[server.GameType], server)
	}

	var fields []*discordgo.MessageEmbedField
	totalPlayers := 0

	for gameType, servers := range gameTypes {
		var serverList []string
		for _, server := range servers {
			serverList = append(serverList, fmt.Sprintf("• **%s** by %s (%d/%d players)\n  `connect: %s:%d`",
				server.ServerName, server.OwnerName, server.Players, server.MaxPlayers, server.Address, server.Port))
			totalPlayers += server.Players
		}

		if len(serverList) > 0 {
			fields = append(fields, &discordgo.MessageEmbedField{
				Name:   fmt.Sprintf("🎮 %s (%d servers)", titleCase(gameType), len(servers)),
				Value:  strings.Join(serverList, "\n\n"),
				Inline: false,
			})
		}
	}

	embed := &discordgo.MessageEmbed{
		Title:  "🌐 WTG Public Lobby",
		Color:  0x00ff00,
		Fields: fields,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Total: %d servers • %d players online • Updated just now", len(publicServers), totalPlayers),
		},
	}

	_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}

func (c *PublicLobbyCommand) addToLobby(ctx *CommandContext) error {
	if len(ctx.Args) < 2 {
		embed := &discordgo.MessageEmbed{
			Title:       "🌐 Add Server to Public Lobby",
			Description: "Share your server with the WTG community",
			Color:       0x00ccff,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:  "Usage",
					Value: "`lobby add <server-name> [description]`",
				},
				{
					Name:  "Requirements",
					Value: "• Server must be running\n• Must be your server\n• Family-friendly content only",
				},
				{
					Name:  "Example",
					Value: "`lobby add minecraft1 \"Survival server with friendly community\"`",
				},
			},
		}
		_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
		return err
	}

	serverName := ctx.Args[1]
	description := "A great server to play on!"
	if len(ctx.Args) > 2 {
		description = strings.Join(ctx.Args[2:], " ")
		description = strings.Trim(description, "\"'")
	}

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

	// Check if server is running
	if server.Status != "running" && server.Status != "ready" {
		embed := &discordgo.MessageEmbed{
			Title:       "❌ Server Not Running",
			Description: fmt.Sprintf("Server '%s' must be online to be added to the public lobby", serverName),
			Color:       0xff0000,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:  "Current Status",
					Value: titleCase(server.Status),
				},
				{
					Name:  "What to do",
					Value: "• Wait for server to finish starting\n• Contact support if server won't start\n• Try diagnostics for more info",
				},
			},
		}
		_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
		return err
	}

	// Add to public lobby
	err = ctx.DB.AddToPublicLobby(server, ctx.Message.Author.Username)
	if err != nil {
		return fmt.Errorf("failed to add to public lobby: %v", err)
	}

	// Mark server as public in the game_servers table
	err = ctx.DB.UpdateServerPublicStatus(server.Name, server.DiscordID, true)
	if err != nil {
		return fmt.Errorf("failed to update server public status: %v", err)
	}

	embed := &discordgo.MessageEmbed{
		Title:       "✅ Added to Public Lobby!",
		Description: fmt.Sprintf("Your server **%s** is now listed in the WTG Public Lobby", serverName),
		Color:       0x00ff00,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "🎮 Server Details",
				Value:  fmt.Sprintf("**Name:** %s\n**Game:** %s\n**Address:** `%s:%d`", server.Name, titleCase(server.GameType), server.Address, server.Port),
				Inline: false,
			},
			{
				Name:   "📝 Description",
				Value:  description,
				Inline: false,
			},
			{
				Name:   "🌟 What's Next",
				Value:  "• Players can now discover your server\n• Your server appears in `lobby list`\n• Monitor with `lobby my` or `diagnostics`",
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "💡 Use 'lobby remove " + serverName + "' to remove from public lobby",
		},
	}

	_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}

func (c *PublicLobbyCommand) removeFromLobby(ctx *CommandContext) error {
	if len(ctx.Args) < 2 {
		embed := &discordgo.MessageEmbed{
			Title:       "🌐 Remove Server from Public Lobby",
			Description: "Make your server private again",
			Color:       0x00ccff,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:  "Usage",
					Value: "`lobby remove <server-name>`",
				},
				{
					Name:  "Example",
					Value: "`lobby remove minecraft1`",
				},
			},
		}
		_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
		return err
	}

	serverName := ctx.Args[1]

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

	// Remove from public lobby
	err = ctx.DB.RemoveFromPublicLobby(server.Name, server.DiscordID)
	if err != nil {
		return fmt.Errorf("failed to remove from public lobby: %v", err)
	}

	// Mark server as private
	err = ctx.DB.UpdateServerPublicStatus(server.Name, server.DiscordID, false)
	if err != nil {
		return fmt.Errorf("failed to update server public status: %v", err)
	}

	embed := &discordgo.MessageEmbed{
		Title:       "✅ Removed from Public Lobby",
		Description: fmt.Sprintf("Your server **%s** is now private", serverName),
		Color:       0x00ff00,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:  "🔒 Privacy",
				Value: "Your server is no longer visible in the public lobby",
			},
			{
				Name:  "🎮 Server Status",
				Value: fmt.Sprintf("Still running and accessible at `%s:%d`", server.Address, server.Port),
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "💡 Use 'lobby add " + serverName + "' to make it public again",
		},
	}

	_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}

func (c *PublicLobbyCommand) showMyPublicServers(ctx *CommandContext) error {
	servers, err := ctx.DB.GetUserServers(ctx.Message.Author.ID)
	if err != nil {
		return fmt.Errorf("failed to get servers: %v", err)
	}

	var publicServers []*services.GameServer
	for _, server := range servers {
		if server.IsPublic {
			publicServers = append(publicServers, server)
		}
	}

	if len(publicServers) == 0 {
		embed := &discordgo.MessageEmbed{
			Title:       "🌐 Your Public Servers",
			Description: "You don't have any servers in the public lobby yet.",
			Color:       0xffa500,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:  "🚀 Share Your Servers",
					Value: "Use `lobby add <server>` to add a server to the public lobby",
				},
				{
					Name:  "📋 Your Servers",
					Value: "Use `servers` to see all your servers",
				},
			},
		}
		_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
		return err
	}

	var fields []*discordgo.MessageEmbedField
	for _, server := range publicServers {
		var statusEmoji string
		switch server.Status {
		case "running", "ready":
			statusEmoji = "✅"
		case "creating":
			statusEmoji = "⏳"
		case "stopped":
			statusEmoji = "⏸️"
		default:
			statusEmoji = "❓"
		}

		value := fmt.Sprintf("**Game:** %s\n**Status:** %s %s\n**Address:** `%s:%d`",
			titleCase(server.GameType), statusEmoji, titleCase(server.Status), server.Address, server.Port)

		fields = append(fields, &discordgo.MessageEmbedField{
			Name:   server.Name,
			Value:  value,
			Inline: true,
		})
	}

	embed := &discordgo.MessageEmbed{
		Title:  "🌐 Your Public Servers",
		Color:  0x00ff00,
		Fields: fields,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("You have %d servers in the public lobby", len(publicServers)),
		},
	}

	_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}
