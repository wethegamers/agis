package commands

import (
	"fmt"
	"strconv"

	"agis-bot/internal/bot"
	"agis-bot/internal/services"

	"github.com/bwmarrin/discordgo"
)

// ModDeleteCommand allows moderators to delete game servers
type ModDeleteCommand struct{}

func (c *ModDeleteCommand) Name() string {
	return "mod-delete"
}

func (c *ModDeleteCommand) Description() string {
	return "Delete a user's game server by name or ID"
}

func (c *ModDeleteCommand) RequiredPermission() bot.Permission {
	return bot.PermissionMod
}

func (c *ModDeleteCommand) Execute(ctx *CommandContext) error {
	if len(ctx.Args) == 0 {
		return c.showHelp(ctx)
	}

	// Check if first argument is a server ID (numeric)
	var serverID int
	var err error
	var server *services.GameServer

	if serverID, err = strconv.Atoi(ctx.Args[0]); err == nil {
		// Get server by ID
		servers, err := ctx.DB.GetAllServers()
		if err != nil {
			return fmt.Errorf("failed to get servers: %v", err)
		}

		found := false
		for _, s := range servers {
			if s.ID == serverID {
				server = s
				found = true
				break
			}
		}

		if !found {
			return c.sendErrorMessage(ctx, fmt.Sprintf("No server found with ID %d", serverID))
		}
	} else {
		// Assume it's a server name
		if len(ctx.Args) < 2 {
			return c.sendErrorMessage(ctx, "When deleting by name, you must specify both server name and Discord ID")
		}

		serverName := ctx.Args[0]
		discordID := ctx.Args[1]

		// Get server by name and Discord ID
		server, err = ctx.DB.GetServerByName(serverName, discordID)
		if err != nil {
			return c.sendErrorMessage(ctx, fmt.Sprintf("No server found with name '%s' for user %s", serverName, discordID))
		}
	}

	if server == nil {
		return c.sendErrorMessage(ctx, "Failed to find the specified server")
	}

	// Confirmation message
	embed := &discordgo.MessageEmbed{
		Title: "⚠️ Confirm Server Deletion",
		Description: fmt.Sprintf("You are about to delete the following server:\n\n**Name**: %s\n**ID**: %d\n**Owner**: <@%s>\n**Game**: %s\n**Status**: %s",
			server.Name, server.ID, server.DiscordID, server.GameType, server.Status),
		Color: 0xff9900,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "❗ Warning",
				Value:  "This action is **irreversible**. All server data will be permanently deleted.",
				Inline: false,
			},
			{
				Name:   "Confirm",
				Value:  fmt.Sprintf("To confirm deletion, type `confirm-delete %d`", server.ID),
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Moderation action - This will be logged",
		},
	}

	_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}

func (c *ModDeleteCommand) showHelp(ctx *CommandContext) error {
	embed := &discordgo.MessageEmbed{
		Title:       "🔧 Mod Server Deletion",
		Description: "Delete a user's game server by ID or name",
		Color:       0x4287f5,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Usage (by ID)",
				Value:  "`mod-delete <server-id>`\nExample: `mod-delete 123`",
				Inline: false,
			},
			{
				Name:   "Usage (by name)",
				Value:  "`mod-delete <server-name> <discord-id>`\nExample: `mod-delete minecraft-server 290955794172739584`",
				Inline: false,
			},
			{
				Name:   "Note",
				Value:  "After issuing the command, you will be asked to confirm the deletion.",
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Moderator permissions required",
		},
	}

	_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}

func (c *ModDeleteCommand) sendErrorMessage(ctx *CommandContext, errorMessage string) error {
	embed := &discordgo.MessageEmbed{
		Title:       "❌ Error",
		Description: errorMessage,
		Color:       0xff0000,
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Use 'mod-delete' without arguments for help",
		},
	}

	_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}
