package commands

import (
	"fmt"
	"time"

	"agis-bot/internal/bot"
	"agis-bot/internal/services"

	"github.com/bwmarrin/discordgo"
)

// ExportSaveCommand allows users to export their server save files
type ExportSaveCommand struct {
	saveFileService *services.SaveFileService
}

func NewExportSaveCommand() *ExportSaveCommand {
	return &ExportSaveCommand{
		saveFileService: services.NewSaveFileService(""),
	}
}

func (c *ExportSaveCommand) Name() string {
	return "export"
}

func (c *ExportSaveCommand) Description() string {
	return "Export save files from your game server"
}

func (c *ExportSaveCommand) RequiredPermission() bot.Permission {
	return bot.PermissionUser
}

func (c *ExportSaveCommand) Execute(ctx *CommandContext) error {
	if len(ctx.Args) == 0 {
		embed := &discordgo.MessageEmbed{
			Title:       "💾 Export Server Save Files",
			Description: "Download your game server's save files before it's removed",
			Color:       0x00ccff,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:  "Usage",
					Value: "`export <server-name>`",
				},
				{
					Name:  "What Gets Exported",
					Value: "• **Minecraft:** World data, player data, settings\n• **Terraria:** World files, player characters\n• **CS2:** Maps, configurations, settings\n• **GMod:** Saves, addons, configurations",
				},
				{
					Name:  "Examples",
					Value: "`export my-minecraft-server`\n`export cs2-pvp`",
				},
				{
					Name:  "⏰ Availability",
					Value: "Export links expire after 24 hours",
				},
			},
			Footer: &discordgo.MessageEmbedFooter{
				Text: "💡 Use 'servers' to see your server names",
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

	// Send initial processing message
	processingEmbed := &discordgo.MessageEmbed{
		Title:       "⏳ Exporting Save Files",
		Description: fmt.Sprintf("Preparing save files for **%s** (%s server)...", server.Name, server.GameType),
		Color:       0xffa500,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:  "Status",
				Value: "📦 Gathering server data...",
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "This may take a few moments depending on save file size",
		},
	}

	message, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, processingEmbed)
	if err != nil {
		return err
	}

	// Export the save files
	export, err := c.saveFileService.ExportServerSave(server)
	if err != nil {
		// Log export failure
		if ctx.Logger != nil {
			ctx.Logger.LogExport(ctx.Message.Author.ID, "export_failed", fmt.Sprintf("Save file export failed for server %s", server.Name), map[string]interface{}{
				"server_id":   server.ID,
				"server_name": server.Name,
				"game_type":   server.GameType,
				"error":       err.Error(),
			})
		}

		// Update message with error
		errorEmbed := &discordgo.MessageEmbed{
			Title:       "❌ Export Failed",
			Description: fmt.Sprintf("Failed to export save files for **%s**", server.Name),
			Color:       0xff0000,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:  "Error",
					Value: fmt.Sprintf("```%s```", err.Error()),
				},
				{
					Name:  "What to do",
					Value: "• Try again in a few minutes\n• Contact support if the problem persists\n• Use `diagnostics " + server.Name + "` for more info",
				},
			},
		}
		_, err = ctx.Session.ChannelMessageEditEmbed(ctx.Message.ChannelID, message.ID, errorEmbed)
		return err
	}

	// Log successful export
	if ctx.Logger != nil {
		ctx.Logger.LogExport(ctx.Message.Author.ID, "export_success", fmt.Sprintf("Save files exported for server %s", server.Name), map[string]interface{}{
			"server_id":   server.ID,
			"server_name": server.Name,
			"game_type":   server.GameType,
			"file_size":   export.FileSize,
			"export_path": export.FilePath,
		})
	}

	// Format file size
	var sizeStr string
	if export.FileSize < 1024 {
		sizeStr = fmt.Sprintf("%d bytes", export.FileSize)
	} else if export.FileSize < 1024*1024 {
		sizeStr = fmt.Sprintf("%.1f KB", float64(export.FileSize)/1024)
	} else {
		sizeStr = fmt.Sprintf("%.1f MB", float64(export.FileSize)/(1024*1024))
	}

	// Create success embed with download info
	successEmbed := &discordgo.MessageEmbed{
		Title:       "✅ Save Files Exported",
		Description: fmt.Sprintf("Save files for **%s** are ready for download!", server.Name),
		Color:       0x00ff00,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "📊 Export Details",
				Value:  fmt.Sprintf("**Game:** %s\n**Size:** %s\n**Exported:** %s", server.GameType, sizeStr, export.ExportedAt.Format("Jan 2, 15:04")),
				Inline: true,
			},
			{
				Name:   "⏰ Expiration",
				Value:  fmt.Sprintf("**Expires:** %s\n*(%s from now)*", export.ExpiresAt.Format("Jan 2, 15:04"), time.Until(export.ExpiresAt).Round(time.Hour)),
				Inline: true,
			},
			{
				Name:  "📥 Download",
				Value: "**Coming Soon:** Web dashboard download\n**For now:** Contact support for file access",
			},
			{
				Name:  "💡 What's Included",
				Value: c.getExportDescription(server.GameType),
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "💾 Save files are automatically cleaned up after 24 hours",
		},
	}

	_, err = ctx.Session.ChannelMessageEditEmbed(ctx.Message.ChannelID, message.ID, successEmbed)
	return err
}

// getExportDescription returns a description of what's included in the export for each game type
func (c *ExportSaveCommand) getExportDescription(gameType string) string {
	switch gameType {
	case "minecraft":
		return "World data, player inventories, server properties, plugin configs"
	case "terraria":
		return "World files (.wld), character data, server configuration"
	case "cs2":
		return "Custom maps, server configuration, workshop items"
	case "gmod":
		return "Save files, installed addons, server settings, custom content"
	default:
		return "Server configuration and save data"
	}
}
