package commands

import (
	"fmt"
	"time"

	"agis-bot/internal/bot"
	"agis-bot/internal/services"

	"github.com/bwmarrin/discordgo"
)

// ConfirmDeleteMineCommand handles user's own server deletion confirmation
type ConfirmDeleteMineCommand struct{}

func (c *ConfirmDeleteMineCommand) Name() string {
	return "confirm-delete-mine"
}

func (c *ConfirmDeleteMineCommand) Description() string {
	return "Confirm deletion of your own game server"
}

func (c *ConfirmDeleteMineCommand) RequiredPermission() bot.Permission {
	return bot.PermissionUser
}

func (c *ConfirmDeleteMineCommand) Execute(ctx *CommandContext) error {
	if len(ctx.Args) == 0 {
		return c.sendErrorMessage(ctx, "Missing server name. Usage: `confirm-delete-mine <server-name>`")
	}

	serverName := ctx.Args[0]

	// Get the server and verify ownership
	server, err := ctx.DB.GetServerByName(serverName, ctx.Message.Author.ID)
	if err != nil {
		return c.sendErrorMessage(ctx, fmt.Sprintf("Could not find server '%s' in your account", serverName))
	}

	if server == nil {
		return c.sendErrorMessage(ctx, fmt.Sprintf("Server '%s' was not found in your account", serverName))
	}

	// Send processing message
	embed := &discordgo.MessageEmbed{
		Title:       "⏳ Deleting Your Server",
		Description: fmt.Sprintf("Deleting server **%s**...", server.Name),
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
		// Log export failure during deletion
		if ctx.Logger != nil {
			ctx.Logger.LogExport(ctx.Message.Author.ID, "export_failed_during_deletion", fmt.Sprintf("Save file export failed during server deletion: %s", server.Name), map[string]interface{}{
				"server_id":   server.ID,
				"server_name": server.Name,
				"game_type":   server.GameType,
				"error":       exportErr.Error(),
			})
		}
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

		// Log successful export during deletion
		if ctx.Logger != nil {
			ctx.Logger.LogExport(ctx.Message.Author.ID, "export_success_during_deletion", fmt.Sprintf("Save files exported during server deletion: %s", server.Name), map[string]interface{}{
				"server_id":   server.ID,
				"server_name": server.Name,
				"game_type":   server.GameType,
				"file_size":   export.FileSize,
				"export_path": export.FilePath,
			})
		}
	}

	// Delete the server using EnhancedServerService (handles both DB and Kubernetes)
	err = ctx.EnhancedServer.DeleteGameServer(ctx.Context, server.Name, ctx.Message.Author.ID)
	if err != nil {
		// Log deletion failure
		if ctx.Logger != nil {
			ctx.Logger.LogUser(ctx.Message.Author.ID, "server_deletion_failed", fmt.Sprintf("Server deletion failed: %s", server.Name), map[string]interface{}{
				"server_id":   server.ID,
				"server_name": server.Name,
				"game_type":   server.GameType,
				"error":       err.Error(),
			})
		}

		// Update message with error
		errorEmbed := &discordgo.MessageEmbed{
			Title:       "❌ Deletion Failed",
			Description: fmt.Sprintf("Failed to delete server **%s**", server.Name),
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

	// Log successful deletion
	if ctx.Logger != nil {
		logDetails := map[string]interface{}{
			"server_id":     server.ID,
			"server_name":   server.Name,
			"game_type":     server.GameType,
			"server_uptime": time.Since(server.CreatedAt).String(),
		}
		if export != nil {
			logDetails["export_size"] = export.FileSize
		}
		ctx.Logger.LogUser(ctx.Message.Author.ID, "server_deleted", fmt.Sprintf("User successfully deleted server: %s", server.Name), logDetails)
	}

	// Update message with success
	successEmbed := &discordgo.MessageEmbed{
		Title:       "✅ Server Deleted Successfully",
		Description: fmt.Sprintf("Your server **%s** has been permanently deleted.", server.Name),
		Color:       0x00ff00,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Server Details",
				Value:  fmt.Sprintf("**Game:** %s\n**Created:** %s\n**Total Uptime:** %s", server.GameType, server.CreatedAt.Format("Jan 2, 2006 15:04"), time.Since(server.CreatedAt).Round(time.Hour)),
				Inline: false,
			},
			{
				Name:   "Save Files",
				Value:  exportResult,
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Thank you for using WTG! Create a new server anytime with 'create'",
		},
	}
	_, err = ctx.Session.ChannelMessageEditEmbed(ctx.Message.ChannelID, message.ID, successEmbed)
	return err
}

func (c *ConfirmDeleteMineCommand) sendErrorMessage(ctx *CommandContext, errorMessage string) error {
	embed := &discordgo.MessageEmbed{
		Title:       "❌ Error",
		Description: errorMessage,
		Color:       0xff0000,
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Use 'servers' to see your server names",
		},
	}
	_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}
