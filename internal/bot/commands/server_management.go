package commands

import (
	"fmt"
	"strings"

	"agis-bot/internal/bot"
	"agis-bot/internal/services"

	"github.com/bwmarrin/discordgo"
)

// CreateServerCommand creates a new game server
type CreateServerCommand struct{}

func (c *CreateServerCommand) Name() string {
	return "create"
}

func (c *CreateServerCommand) Description() string {
	return "Create a new game server"
}

func (c *CreateServerCommand) RequiredPermission() bot.Permission {
	return bot.PermissionUser
}

func (c *CreateServerCommand) Execute(ctx *CommandContext) error {
	if len(ctx.Args) == 0 {
		embed := &discordgo.MessageEmbed{
			Title:       "🚀 Create Game Server",
			Description: "Deploy a new game server in the WTG cluster",
			Color:       0x00ccff,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:  "Usage",
					Value: "`create <game-type> [server-name] [--here]`",
				},
				{
					Name:  "Available Games",
					Value: "• **minecraft** - Minecraft Java Edition\n• **cs2** - Counter-Strike 2\n• **terraria** - Terraria\n• **gmod** - Garry's Mod",
				},
				{
					Name:  "Examples",
					Value: "`create minecraft`\n`create cs2 my-cs-server`\n`create terraria survival-world --here`",
				},
				{
					Name:  "Flags",
					Value: "• `--here` - Show all updates in this channel instead of DMs",
				},
				{
					Name:  "💰 Costs",
					Value: "• Minecraft: 5 credits/hour\n• CS2: 8 credits/hour\n• Terraria: 3 credits/hour\n• GMod: 6 credits/hour",
				},
				{
					Name:  "⏱️ Deployment Time",
					Value: "Most servers deploy in 2-5 minutes",
				},
			},
			Footer: &discordgo.MessageEmbedFooter{
				Text: "💡 Check your credits with 'credits' before creating a server",
			},
		}
		_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
		return err
	}

	// Parse arguments and flags
	var gameType, serverName string
	var notifyInChannel bool

	args := make([]string, 0)
	for _, arg := range ctx.Args {
		if arg == "--here" {
			notifyInChannel = true
		} else {
			args = append(args, arg)
		}
	}

	if len(args) == 0 {
		return fmt.Errorf("game type is required")
	}

	gameType = strings.ToLower(args[0])
	serverName = fmt.Sprintf("%s-%s", gameType, ctx.Message.Author.Username)
	if len(args) > 1 {
		serverName = args[1]
	}

	// BLOCKER 1: Validate game type using dynamic pricing
	if ctx.PricingService == nil {
		return fmt.Errorf("pricing service not available - contact administrator")
	}

	pricing, err := ctx.PricingService.GetPricing(gameType)
	if err != nil {
		// Game type not found or inactive
		allPricing := ctx.PricingService.GetAllPricing()
		availableGames := make([]string, 0, len(allPricing))
		for _, p := range allPricing {
			if !p.RequiresGuild {
				availableGames = append(availableGames, fmt.Sprintf("%s (%d GC/hr)", p.GameType, p.CostPerHour))
			}
		}

		embed := &discordgo.MessageEmbed{
			Title:       "❌ Invalid Game Type",
			Description: fmt.Sprintf("Game type '%s' is not supported", gameType),
			Color:       0xff0000,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:  "Available Games",
					Value: strings.Join(availableGames, ", "),
				},
				{
					Name:  "💡 Titan-Tier Servers",
					Value: "High-resource games like ARK require guild pooling. Use `guild-create` first.",
				},
			},
		}
		_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
		return err
	}

	// CRITICAL: Enforce requires_guild for Titan-tier servers
	if pricing.RequiresGuild {
		embed := &discordgo.MessageEmbed{
			Title:       "🏰 Guild Required",
			Description: fmt.Sprintf("**%s** is a Titan-tier game that requires guild pooling", pricing.DisplayName),
			Color:       0xff9900,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:  "Why Guild-Only?",
					Value: fmt.Sprintf("This game costs **%d GC/hour**—too expensive for individual users. Guilds pool resources from multiple members.", pricing.CostPerHour),
				},
				{
					Name:  "How to Create This Server",
					Value: "1. Create a guild: `guild-create <name>`\n2. Invite members: `guild-invite @user <guild_id>`\n3. Pool credits: `guild-deposit <guild_id> <amount>`\n4. Create server: Use guild-server commands",
				},
				{
					Name:  "💡 Alternative",
					Value: "Consider lighter games like Minecraft (30 GC/hr) or CS2 (120 GC/hr) for individual play.",
				},
			},
			Footer: &discordgo.MessageEmbedFooter{
				Text: "Guild treasuries enable premium experiences impossible for competitors",
			},
		}
		_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
		return err
	}

	costPerHour := pricing.CostPerHour

	// Check user credits
	user, err := ctx.DB.GetOrCreateUser(ctx.Message.Author.ID)
	if err != nil {
		return fmt.Errorf("failed to get user: %v", err)
	}

	if user.Credits < costPerHour {
		embed := &discordgo.MessageEmbed{
			Title:       "💰 Insufficient Credits",
			Description: fmt.Sprintf("You need %d credits to create a %s server", costPerHour, gameType),
			Color:       0xff0000,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:  "Your Balance",
					Value: fmt.Sprintf("%d credits", user.Credits),
				},
				{
					Name:  "Earn More Credits",
					Value: "• Use `work` (every hour)\n• Use `credits earn` for ad rewards\n• Daily bonuses coming soon!",
				},
			},
		}
		_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
		return err
	}

	// Check if server name already exists
	existingServer, _ := ctx.DB.GetServerByName(serverName, ctx.Message.Author.ID)
	if existingServer != nil {
		embed := &discordgo.MessageEmbed{
			Title:       "❌ Server Name Exists",
			Description: fmt.Sprintf("You already have a server named '%s'", serverName),
			Color:       0xff0000,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:  "What to do",
					Value: "• Choose a different name\n• Use `servers` to see your existing servers\n• Stop the existing server if no longer needed",
				},
			},
		}
		_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
		return err
	}

	// Create the server record using enhanced service
	enhancedService := ctx.EnhancedServer
	if enhancedService == nil {
		// Fallback to old method if enhanced service not available
		server := &services.GameServer{
			DiscordID:   ctx.Message.Author.ID,
			Name:        serverName,
			GameType:    gameType,
			Status:      "creating",
			CostPerHour: costPerHour,
			IsPublic:    false,
			Description: fmt.Sprintf("A %s server", gameType),
		}

		err = ctx.DB.SaveGameServer(server)
		if err != nil {
			return fmt.Errorf("failed to create server: %v", err)
		}
	} else {
		// Use enhanced service for full lifecycle management
		if notifyInChannel {
			_, err = enhancedService.CreateGameServer(ctx.Context, ctx.Message.Author.ID, gameType, serverName, costPerHour, ctx.Message.ChannelID)
		} else {
			_, err = enhancedService.CreateGameServer(ctx.Context, ctx.Message.Author.ID, gameType, serverName, costPerHour)
		}
		if err != nil {
			return fmt.Errorf("failed to create server: %v", err)
		}
	}

	// Deduct initial credits (1 hour worth)
	err = ctx.DB.AddCredits(ctx.Message.Author.ID, -costPerHour)
	if err != nil {
		return fmt.Errorf("failed to deduct credits: %v", err)
	}

	// Get updated user balance after deduction
	updatedUser, err := ctx.DB.GetOrCreateUser(ctx.Message.Author.ID)
	if err != nil {
		return fmt.Errorf("failed to get updated user: %v", err)
	}

	// Send success message
	embed := &discordgo.MessageEmbed{
		Title:       "🚀 Server Creation Started!",
		Description: fmt.Sprintf("Deploying **%s** (%s server)", serverName, gameType),
		Color:       0x00ff00,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "📋 Server Details",
				Value:  fmt.Sprintf("**Name:** %s\n**Game:** %s\n**Cost:** %d credits/hour", serverName, titleCase(gameType), costPerHour),
				Inline: false,
			},
			{
				Name:   "⏱️ Deployment Progress",
				Value:  "🔄 **Starting deployment...**\n⏳ Estimated time: 2-5 minutes",
				Inline: false,
			},
			{
				Name:   "💰 Credits",
				Value:  fmt.Sprintf("Deducted: %d credits\nRemaining: %d credits", costPerHour, updatedUser.Credits),
				Inline: true,
			},
			{
				Name:   "📊 What's Next",
				Value:  "• Monitor with `diagnostics " + serverName + "`\n• Check status with `servers`\n• Get connection info when ready",
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "💡 Your server will automatically start billing when deployment completes",
		},
	}

	_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}

// StopServerCommand stops a user's game server
type StopServerCommand struct{}

func (c *StopServerCommand) Name() string {
	return "stop"
}

func (c *StopServerCommand) Description() string {
	return "Stop one of your game servers"
}

func (c *StopServerCommand) RequiredPermission() bot.Permission {
	return bot.PermissionUser
}

func (c *StopServerCommand) Execute(ctx *CommandContext) error {
	if len(ctx.Args) == 0 {
		embed := &discordgo.MessageEmbed{
			Title:       "⏹️ Stop Server",
			Description: "Stop one of your running game servers",
			Color:       0xff9900,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:  "Usage",
					Value: "`stop <server-name>`",
				},
				{
					Name:  "Example",
					Value: "`stop minecraft1`",
				},
				{
					Name:  "💰 Credit Savings",
					Value: "Stopping servers prevents credit consumption while preserving your world/progress",
				},
			},
			Footer: &discordgo.MessageEmbedFooter{
				Text: "💡 Use 'servers' to see your running servers",
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
			Footer: &discordgo.MessageEmbedFooter{
				Text: "Use 'servers' to see your available servers",
			},
		}
		_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
		return err
	}

	if server.Status == "stopped" {
		embed := &discordgo.MessageEmbed{
			Title:       "⏸️ Server Already Stopped",
			Description: fmt.Sprintf("Server '%s' is already stopped", serverName),
			Color:       0xffa500,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:  "Current Status",
					Value: "⏸️ Stopped",
				},
				{
					Name:  "💰 Credit Consumption",
					Value: "✅ Not consuming credits",
				},
			},
		}
		_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
		return err
	}

	if server.Status == "stopping" {
		embed := &discordgo.MessageEmbed{
			Title:       "⏹️ Server Already Stopping",
			Description: fmt.Sprintf("Server '%s' is already being stopped", serverName),
			Color:       0xffa500,
		}
		_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
		return err
	}

	// Update server status to stopping
	err = ctx.DB.UpdateServerStatus(server.Name, server.DiscordID, "stopping")
	if err != nil {
		return fmt.Errorf("failed to update server status: %v", err)
	}

	// Remove from public lobby if it was listed
	if server.IsPublic {
		ctx.DB.RemoveFromPublicLobby(server.Name, server.DiscordID)
		ctx.DB.UpdateServerPublicStatus(server.Name, server.DiscordID, false)
	}

	embed := &discordgo.MessageEmbed{
		Title:       "⏹️ Server Stop Initiated",
		Description: fmt.Sprintf("Stopping server **%s**", serverName),
		Color:       0xff9900,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "🎮 Server Details",
				Value:  fmt.Sprintf("**Name:** %s\n**Game:** %s\n**Previous Status:** %s", server.Name, titleCase(server.GameType), titleCase(server.Status)),
				Inline: false,
			},
			{
				Name:   "💰 Credit Savings",
				Value:  fmt.Sprintf("Will stop consuming %d credits/hour\nEstimated savings: ~%d credits/day", server.CostPerHour, server.CostPerHour*24),
				Inline: false,
			},
			{
				Name:   "📊 What Happens Next",
				Value:  "• Server will safely shutdown\n• World/progress is preserved\n• Credit billing stops\n• Can be restarted later (contact support)",
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "⏱️ Shutdown typically completes in 30-60 seconds",
		},
	}

	if server.IsPublic {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "🌐 Public Lobby",
			Value:  "Server has been removed from the public lobby",
			Inline: false,
		})
	}

	_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}
