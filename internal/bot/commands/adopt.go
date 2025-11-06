package commands

import (
	"fmt"
	"log"

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

	log.Printf("[adopt][%s] Starting adoption for server '%s' by user '%s'", ctx.Message.ID, serverName, userID)

	// Step 1: Find the GameServer in Agones
	info, err := ctx.Agones.FindGameServerByServerName(ctx.Context, serverName)
	if err != nil {
		log.Printf("[adopt][%s] ERROR: Failed to locate GameServer '%s': %v", ctx.Message.ID, serverName, err)
		return fmt.Errorf("failed to locate GameServer: %v", err)
	}
	log.Printf("[adopt][%s] Found GameServer - UID: %s, Status: %s, Address: %s:%d", 
		ctx.Message.ID, info.UID, info.Status, info.Address, info.Port)

	// Step 2: Ensure user exists in database
	log.Printf("[adopt][%s] Ensuring user %s exists in database", ctx.Message.ID, userID)
	user, err := ctx.DB.GetOrCreateUser(userID)
	if err != nil {
		log.Printf("[adopt][%s] ERROR: Failed to ensure user exists: %v", ctx.Message.ID, err)
		return fmt.Errorf("failed to ensure user exists: %v", err)
	}
	log.Printf("[adopt][%s] User verified - Credits: %d, Tier: %s", ctx.Message.ID, user.Credits, user.Tier)

	// Step 3: Check if server already exists in database
	log.Printf("[adopt][%s] Checking if server already exists in database", ctx.Message.ID)
	existingServer, err := ctx.DB.GetServerByName(serverName, userID)
	
	if err != nil {
		// Server doesn't exist, create it
		log.Printf("[adopt][%s] Server not found in database, creating new record", ctx.Message.ID)
		
		// Determine status based on Agones status
		status := "running"
		if info.Status == "Allocated" {
			status = "running"
		} else if info.Status == "Ready" {
			status = "ready"
		} else {
			status = "error"
		}

		gs := &services.GameServer{
			DiscordID:      userID,
			Name:           serverName,
			GameType:       "minecraft", // Could be inferred from labels if available
			Status:         status,
			Address:        info.Address,
			Port:           int(info.Port),
			KubernetesUID:  info.UID,
			AgonesStatus:   string(info.Status),
			CostPerHour:    5,
			IsPublic:       false,
			Description:    fmt.Sprintf("Adopted server %s", serverName),
		}

		if err := ctx.DB.SaveGameServer(gs); err != nil {
			log.Printf("[adopt][%s] ERROR: Failed to save GameServer to database: %v", ctx.Message.ID, err)
			return fmt.Errorf("failed to save GameServer to database: %v", err)
		}
		log.Printf("[adopt][%s] ✅ GameServer record created in database", ctx.Message.ID)
	} else {
		// Server exists, update it
		log.Printf("[adopt][%s] Server already exists (ID: %d), updating information", ctx.Message.ID, existingServer.ID)
	}

	// Step 4: Update Kubernetes/Agones fields
	log.Printf("[adopt][%s] Updating Kubernetes/Agones metadata", ctx.Message.ID)
	if err := ctx.DB.UpdateServerKubernetesInfo(serverName, userID, info.UID, string(info.Status)); err != nil {
		log.Printf("[adopt][%s] WARNING: Failed to update Kubernetes info: %v", ctx.Message.ID, err)
		// Don't fail the whole operation if this fails
	} else {
		log.Printf("[adopt][%s] Updated Kubernetes UID and Agones status", ctx.Message.ID)
	}

	// Step 5: Update address if available
	if info.Address != "" {
		log.Printf("[adopt][%s] Updating server address: %s:%d", ctx.Message.ID, info.Address, info.Port)
		if err := ctx.DB.UpdateServerAddress(serverName, userID, info.Address, int(info.Port)); err != nil {
			log.Printf("[adopt][%s] WARNING: Failed to update address: %v", ctx.Message.ID, err)
		} else {
			log.Printf("[adopt][%s] Address updated successfully", ctx.Message.ID)
		}
	}

	log.Printf("[adopt][%s] ✅ Adoption completed successfully", ctx.Message.ID)

	// Step 6: Send success message to Discord
	embed := &discordgo.MessageEmbed{
		Title:       "✅ Adopted GameServer",
		Description: fmt.Sprintf("Linked %s to <@%s>", serverName, userID),
		Color:       0x00ff99,
		Fields: []*discordgo.MessageEmbedField{
			{Name: "State", Value: string(info.Status), Inline: true},
			{Name: "Address", Value: fmt.Sprintf("%s:%d", info.Address, info.Port), Inline: true},
			{Name: "UID", Value: info.UID, Inline: false},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Server is now tracked in database",
		},
	}
	_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}
