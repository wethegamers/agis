package commands

import (
	"fmt"

	"agis-bot/internal/bot"

	"github.com/bwmarrin/discordgo"
)

// DeleteServerCommand allows users to delete their own servers
type DeleteServerCommand struct{}

func (c *DeleteServerCommand) Name() string {
	return "delete"
}

func (c *DeleteServerCommand) Description() string {
	return "Delete one of your own game servers"
}

func (c *DeleteServerCommand) RequiredPermission() bot.Permission {
	return bot.PermissionUser
}

func (c *DeleteServerCommand) Execute(ctx *CommandContext) error {
	if len(ctx.Args) == 0 {
		return c.showHelp(ctx)
	}

	serverName := ctx.Args[0]

	// Get the server
	server, err := ctx.DB.GetServerByName(serverName, ctx.Message.Author.ID)
	if err != nil {
		// Log deletion attempt failure
		if ctx.Logger != nil {
			ctx.Logger.LogUser(ctx.Message.Author.ID, "delete_attempt_failed", fmt.Sprintf("Failed to find server for deletion: %s", serverName), map[string]interface{}{
				"server_name": serverName,
				"error":       err.Error(),
			})
		}

		embed := &discordgo.MessageEmbed{
			Title:       "❌ Server Not Found",
			Description: fmt.Sprintf("Could not find a server named '%s' in your account", serverName),
			Color:       0xff0000,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:  "What to do",
					Value: "• Use `servers` to see your servers\n• Check the spelling of the server name\n• Make sure you own this server",
				},
			},
		}
		_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
		return err
	}

	// Check if server exists
	if server == nil {
		embed := &discordgo.MessageEmbed{
			Title:       "❌ Server Not Found",
			Description: fmt.Sprintf("Server '%s' was not found", serverName),
			Color:       0xff0000,
		}
		_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
		return err
	}

	// Log deletion initiation
	if ctx.Logger != nil {
		ctx.Logger.LogUser(ctx.Message.Author.ID, "delete_initiated", fmt.Sprintf("User initiated deletion for server: %s", server.Name), map[string]interface{}{
			"server_id":   server.ID,
			"server_name": server.Name,
			"game_type":   server.GameType,
			"status":      server.Status,
		})
	}

	// Confirmation message
	embed := &discordgo.MessageEmbed{
		Title: "⚠️ Confirm Server Deletion",
		Description: fmt.Sprintf("You are about to delete your server:\n\n**Name**: %s\n**Game**: %s\n**Status**: %s\n**Created**: %s",
			server.Name, server.GameType, server.Status, server.CreatedAt.Format("Jan 2, 2006 15:04")),
		Color: 0xff9900,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "❗ Warning",
				Value:  "This action is **irreversible**. All server data will be permanently deleted.",
				Inline: false,
			},
			{
				Name:   "💾 Save Files",
				Value:  "Consider using `export " + server.Name + "` to download your save files first!",
				Inline: false,
			},
			{
				Name:   "Confirm",
				Value:  fmt.Sprintf("To confirm deletion, type `confirm-delete-mine %s`", server.Name),
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "You can cancel by simply not responding",
		},
	}

	_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}

func (c *DeleteServerCommand) showHelp(ctx *CommandContext) error {
	embed := &discordgo.MessageEmbed{
		Title:       "🗑️ Delete Your Server",
		Description: "Delete one of your own game servers",
		Color:       0x4287f5,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Usage",
				Value:  "`delete <server-name>`\nExample: `delete my-minecraft-server`",
				Inline: false,
			},
			{
				Name:   "💡 Tips",
				Value:  "• Use `servers` to see your server names\n• Use `export <server-name>` to backup saves first\n• Deletion requires confirmation",
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Only you can delete your own servers",
		},
	}

	_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}
