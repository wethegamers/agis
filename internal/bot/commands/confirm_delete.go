package commands

import (
	"fmt"
	"strconv"
	"time"

	"agis-bot/internal/bot"
	"agis-bot/internal/services"

	"github.com/bwmarrin/discordgo"
)

// ConfirmDeleteCommand handles server deletion confirmation
type ConfirmDeleteCommand struct{}

func (c *ConfirmDeleteCommand) Name() string {
	return "confirm-delete"
}

func (c *ConfirmDeleteCommand) Description() string {
	return "Confirm deletion of a game server (used after mod-delete)"
}

func (c *ConfirmDeleteCommand) RequiredPermission() bot.Permission {
	return bot.PermissionMod
}

func (c *ConfirmDeleteCommand) Execute(ctx *CommandContext) error {
	if len(ctx.Args) == 0 {
		return c.sendErrorMessage(ctx, "Missing server ID. Usage: `confirm-delete <server-id>`")
	}

	// Parse server ID
	serverID, err := strconv.Atoi(ctx.Args[0])
	if err != nil {
		return c.sendErrorMessage(ctx, fmt.Sprintf("Invalid server ID: %s", ctx.Args[0]))
	}

	// Get all servers
	servers, err := ctx.DB.GetAllServers()
	if err != nil {
		return c.sendErrorMessage(ctx, fmt.Sprintf("Failed to get servers: %v", err))
	}

	// Find the server
	var server *services.GameServer
	for _, s := range servers {
		if s.ID == serverID {
			server = s
			break
		}
	}

	if server == nil {
		return c.sendErrorMessage(ctx, fmt.Sprintf("No server found with ID %d", serverID))
	}

	// Send processing message
	embed := &discordgo.MessageEmbed{
		Title:       "⏳ Processing Server Deletion",
		Description: fmt.Sprintf("Deleting server **%s** (ID: %d)...", server.Name, server.ID),
		Color:       0xffa500,
	}
	message, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	if err != nil {
		return err
	}

	// Export save files first (best effort)
	var exportResult string
	exportService := services.NewSaveFileService("")
	export, exportErr := exportService.ExportServerSave(server)

	if exportErr != nil {
		exportResult = fmt.Sprintf("❌ **Export Failed**: %v", exportErr)
	} else {
		// Format file size nicely
		var sizeStr string
		if export.FileSize < 1024 {
			sizeStr = fmt.Sprintf("%d bytes", export.FileSize)
		} else if export.FileSize < 1024*1024 {
			sizeStr = fmt.Sprintf("%.1f KB", float64(export.FileSize)/1024)
		} else {
			sizeStr = fmt.Sprintf("%.1f MB", float64(export.FileSize)/(1024*1024))
		}
		exportResult = fmt.Sprintf("✅ **Save Files Exported**: %s (%s)", export.FilePath, sizeStr)
	}

	// Delete the server
	err = ctx.DB.DeleteGameServer(server.ID)
	if err != nil {
		// Update message with error
		errorEmbed := &discordgo.MessageEmbed{
			Title:       "❌ Deletion Failed",
			Description: fmt.Sprintf("Failed to delete server **%s** (ID: %d)", server.Name, server.ID),
			Color:       0xff0000,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "Error",
					Value:  fmt.Sprintf("```%v```", err),
					Inline: false,
				},
			},
		}
		_, _ = ctx.Session.ChannelMessageEditEmbed(ctx.Message.ChannelID, message.ID, errorEmbed)
		return err
	}

	// Update message with success
	successEmbed := &discordgo.MessageEmbed{
		Title:       "✅ Server Deleted",
		Description: fmt.Sprintf("Server **%s** (ID: %d) has been successfully deleted.", server.Name, server.ID),
		Color:       0x00ff00,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Server Details",
				Value:  fmt.Sprintf("**Game:** %s\n**Owner:** <@%s>\n**Created:** %s", server.GameType, server.DiscordID, server.CreatedAt.Format("Jan 2, 2006 15:04:05")),
				Inline: false,
			},
			{
				Name:   "Save Files",
				Value:  exportResult,
				Inline: false,
			},
			{
				Name:   "Moderation Log",
				Value:  fmt.Sprintf("**Deleted By:** %s (%s)\n**Timestamp:** %s", ctx.Message.Author.Username, ctx.Message.Author.ID, time.Now().Format("Jan 2, 2006 15:04:05")),
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "This action has been logged",
		},
	}
	_, err = ctx.Session.ChannelMessageEditEmbed(ctx.Message.ChannelID, message.ID, successEmbed)
	return err
}

func (c *ConfirmDeleteCommand) sendErrorMessage(ctx *CommandContext, errorMessage string) error {
	embed := &discordgo.MessageEmbed{
		Title:       "❌ Error",
		Description: errorMessage,
		Color:       0xff0000,
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Moderator permissions required",
		},
	}
	_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}
