package commands

import (
	"context"
	"fmt"
	"strings"

	"agis-bot/internal/services"
)

// GuildServerCommand handles guild server provisioning
type GuildServerCommand struct {
	provisioningService *services.GuildProvisioningService
}

// NewGuildServerCommand creates a new guild server command handler
func NewGuildServerCommand(provisioningService *services.GuildProvisioningService) *GuildServerCommand {
	return &GuildServerCommand{
		provisioningService: provisioningService,
	}
}

// HandleTemplates lists available server templates
func (c *GuildServerCommand) HandleTemplates(userID string) (string, error) {
	templates, err := c.provisioningService.GetAvailableTemplates()
	if err != nil {
		return "", fmt.Errorf("failed to get templates: %w", err)
	}

	message := "🎮 **Available Server Templates**\n\n"
	for _, tmpl := range templates {
		message += fmt.Sprintf("**%s** (`%s`)\n", tmpl.Name, tmpl.ID)
		message += fmt.Sprintf("  Game: %s | Size: %s | Players: %d\n", tmpl.GameType, tmpl.Size, tmpl.MaxPlayers)
		message += fmt.Sprintf("  Cost: **%d GC/hour** | Setup: **%d GC**\n", tmpl.Cost, tmpl.SetupCost)
		message += fmt.Sprintf("  Resources: %s CPU, %s RAM\n", tmpl.CPURequest, tmpl.MemoryRequest)
		message += fmt.Sprintf("  %s\n\n", tmpl.Description)
	}

	message += "Use `/guild-server create <template_id> <name> <hours>` to provision a server"

	return message, nil
}

// HandleCreate creates a new server provisioning request
// Usage: /guild-server create <template_id> <name> <hours> [auto-renew]
func (c *GuildServerCommand) HandleCreate(guildID, userID string, args []string) (string, error) {
	if len(args) < 3 {
		return "", fmt.Errorf("usage: /guild-server create <template_id> <name> <hours> [auto-renew]")
	}

	templateID := args[0]
	serverName := args[1]
	
	var durationHours int
	fmt.Sscanf(args[2], "%d", &durationHours)

	autoRenew := false
	if len(args) > 3 && strings.ToLower(args[3]) == "auto-renew" {
		autoRenew = true
	}

	req := &services.ProvisionRequest{
		GuildID:      guildID,
		RequestedBy:  userID,
		TemplateID:   templateID,
		ServerName:   serverName,
		DurationHours: durationHours,
		AutoRenew:    autoRenew,
	}

	ctx := context.Background()
	if err := c.provisioningService.RequestProvisioning(ctx, req); err != nil {
		return "", fmt.Errorf("failed to provision server: %w", err)
	}

	message := fmt.Sprintf("✅ Server provisioning requested!\n\n"+
		"**Server Name**: %s\n"+
		"**Template**: %s\n"+
		"**Duration**: %d hours\n"+
		"**Auto-Renew**: %v\n\n"+
		"Server will be created shortly. Check status with `/guild-server list`",
		serverName, templateID, durationHours, autoRenew)

	return message, nil
}

// HandleList lists active guild servers
func (c *GuildServerCommand) HandleList(guildID string) (string, error) {
	ctx := context.Background()
	servers, err := c.provisioningService.GetGuildServers(ctx, guildID)
	if err != nil {
		return "", fmt.Errorf("failed to get servers: %w", err)
	}

	if len(servers) == 0 {
		return "No active servers for this guild.\n\nCreate one with `/guild-server create`", nil
	}

	message := "🖥️  **Guild Servers**\n\n"
	for i, server := range servers {
		statusEmoji := map[string]string{
			"pending":      "⏳",
			"provisioning": "🔧",
			"active":       "✅",
			"terminated":   "🛑",
			"failed":       "❌",
		}[server.Status]

		message += fmt.Sprintf("%d. %s **%s** (`%s`)\n", i+1, statusEmoji, server.ServerName, server.ServerID)
		message += fmt.Sprintf("   Template: %s | Duration: %dh\n", server.TemplateID, server.DurationHours)
		message += fmt.Sprintf("   Status: %s | Auto-Renew: %v\n", server.Status, server.AutoRenew)
		message += fmt.Sprintf("   Requested: %s\n\n", server.RequestedAt.Format("2006-01-02 15:04"))
	}

	return message, nil
}

// HandleTerminate manually terminates a running server
func (c *GuildServerCommand) HandleTerminate(guildID, userID, serverID string) (string, error) {
	ctx := context.Background()
	if err := c.provisioningService.TerminateServer(ctx, guildID, serverID); err != nil {
		return "", fmt.Errorf("failed to terminate server: %w", err)
	}

	return fmt.Sprintf("🛑 Server **%s** has been terminated.\nResources released back to guild treasury.", serverID), nil
}

// HandleTreasury shows guild treasury balance
func (c *GuildServerCommand) HandleTreasury(guildID string) (string, error) {
	// TODO: Query guild treasury balance from database
	// For now, return placeholder
	return "💰 **Guild Treasury**\n\n"+
		"Balance: 5,000 GC\n"+
		"Total Earned: 15,000 GC\n"+
		"Total Spent: 10,000 GC\n\n"+
		"Members can contribute by earning Game Credits through `/earn` commands.\n"+
		"Server costs are deducted hourly from the treasury.", nil
}

// HandleInfo shows detailed server information
func (c *GuildServerCommand) HandleInfo(guildID, serverID string) (string, error) {
	// TODO: Query server details from database
	return fmt.Sprintf("📊 **Server Information: %s**\n\n"+
		"Status: Active\n"+
		"Uptime: 12h 34m\n"+
		"Players: 5/10\n"+
		"Cost: 100 GC/hour\n"+
		"Next Renewal: 47 minutes\n\n"+
		"Connection: `play.example.com:25565`", serverID), nil
}
