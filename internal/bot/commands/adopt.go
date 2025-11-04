package commands

import (
	"fmt"

	"agis-bot/internal/bot"
	"agis-bot/internal/services"

	"github.com/bwmarrin/discordgo"
)

// AdoptCommand links an existing Agones GameServer into the DB
// Usage: adopt <server-name> <discord_id>
type AdoptCommand struct{}

func (c *AdoptCommand) Name() string { return "adopt" }
func (c *AdoptCommand) Description() string {
	return "Admin: adopt an existing Agones GameServer into the database"
}
func (c *AdoptCommand) RequiredPermission() bot.Permission { return bot.PermissionAdmin }

func (c *AdoptCommand) Execute(ctx *CommandContext) error {
	if len(ctx.Args) < 2 {
		embed := &discordgo.MessageEmbed{
			Title:       "❌ Usage",
			Description: "`adopt <server-name> <discord_id>`",
			Color:       0xff0000,
		}
		_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
		return err
	}

	serverName := ctx.Args[0]
	userID := ctx.Args[1]

	if ctx.Agones == nil {
		return fmt.Errorf("Agones service not available")
	}

	info, err := ctx.Agones.FindGameServerByServerName(ctx.Context, serverName)
	if err != nil {
		return fmt.Errorf("failed to locate GameServer: %v", err)
	}

	// Upsert DB record
	if _, err := ctx.DB.GetServerByName(serverName, userID); err != nil {
		// Create if not found
		gs := &services.GameServer{
			DiscordID:   userID,
			Name:        serverName,
			GameType:    "minecraft",
			Status:      "ready",
			Address:     info.Address,
			Port:        int(info.Port),
			CostPerHour: 5,
			IsPublic:    false,
			Description: fmt.Sprintf("Adopted server %s", serverName),
		}
		_ = ctx.DB.SaveGameServer(gs) // ignore duplicate errors
	}
	// Update k8s/Agones fields
	_ = ctx.DB.UpdateServerKubernetesInfo(serverName, userID, info.UID, string(info.Status))
	if info.Address != "" {
		_ = ctx.DB.UpdateServerAddress(serverName, userID, info.Address, int(info.Port))
	}

	embed := &discordgo.MessageEmbed{
		Title:       "✅ Adopted GameServer",
		Description: fmt.Sprintf("Linked %s to <@%s>", serverName, userID),
		Color:       0x00ff99,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "State", Value: string(info.Status), Inline: true},
			{Name: "Address", Value: fmt.Sprintf("%s:%d", info.Address, info.Port), Inline: true},
			{Name: "UID", Value: info.UID, Inline: false},
		},
	}
	_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}
