package commands

import (
	"fmt"
	"time"
)

type RestartServerCommand struct{}

func (c *RestartServerCommand) Name() string {
	return "restart"
}

func (c *RestartServerCommand) Description() string {
	return "Restart a running server"
}

func (c *RestartServerCommand) RequiredPermission() PermissionLevel {
	return PermissionUser
}

func (c *RestartServerCommand) Execute(ctx *CommandContext) error {
	if len(ctx.Args) == 0 {
		return fmt.Errorf("usage: restart <server-name>")
	}

	serverName := ctx.Args[0]
	user, err := ctx.DB.GetOrCreateUser(ctx.Message.Author.ID)
	if err != nil {
		return fmt.Errorf("failed to get user: %v", err)
	}

	// Get user's servers
	servers, err := ctx.DB.GetUserServers(ctx.Message.Author.ID)
	if err != nil {
		return fmt.Errorf("failed to get your servers: %v", err)
	}

	// Find the server
	var targetServer *GameServer
	for _, srv := range servers {
		if srv.Name == serverName {
			targetServer = srv
			break
		}
	}

	if targetServer == nil {
		return fmt.Errorf("server '%s' not found. Use `servers` to list your servers", serverName)
	}

	// Check if server is actually running
	if targetServer.Status == "stopped" {
		return fmt.Errorf("server '%s' is stopped. Use `start %s` to start it", serverName, serverName)
	}

	if targetServer.Status != "running" && targetServer.Status != "ready" {
		return fmt.Errorf("server '%s' is currently %s. Can only restart running servers", serverName, targetServer.Status)
	}

	// Deduct credits for restart (1 credit administrative action)
	if user.Credits < 1 {
		return fmt.Errorf("insufficient credits. Need 1 credit for restart. You have %d credits", user.Credits)
	}

	// Stop the server first
	if err := ctx.DB.UpdateServerStatus(targetServer.ID, "stopping"); err != nil {
		return fmt.Errorf("failed to update server status: %v", err)
	}

	// Update stopped timestamp
	now := time.Now()
	if err := ctx.DB.UpdateServerField(targetServer.ID, "stopped_at", now); err != nil {
		ctx.Logger.Printf("Failed to update stopped_at: %v", err)
	}

	// Send stopping notification
	ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, fmt.Sprintf("⏸️ Stopping server '%s' for restart...", serverName))

	// Wait a moment for graceful shutdown
	time.Sleep(3 * time.Second)

	// Start the server again
	if err := ctx.DB.UpdateServerStatus(targetServer.ID, "creating"); err != nil {
		return fmt.Errorf("failed to restart server: %v", err)
	}

	// Clear stopped timestamp
	if err := ctx.DB.UpdateServerField(targetServer.ID, "stopped_at", nil); err != nil {
		ctx.Logger.Printf("Failed to clear stopped_at: %v", err)
	}

	// Deduct credit
	if err := ctx.DB.DeductCredits(ctx.Message.Author.ID, 1); err != nil {
		ctx.Logger.Printf("Failed to deduct restart credit: %v", err)
	}

	// Log restart action
	ctx.Logger.LogAction(ctx.Message.Author.ID, "server_restarted", map[string]interface{}{
		"server_id":   targetServer.ID,
		"server_name": serverName,
		"game_type":   targetServer.GameType,
	})

	return ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, fmt.Sprintf(
		"🔄 **Server Restarted**\n"+
			"Server: `%s`\n"+
			"Game: %s\n"+
			"Status: Restarting\n"+
			"Cost: 1 credit\n"+
			"New Balance: %d credits\n\n"+
			"Server will be online shortly. Use `diagnostics %s` to check status.",
		serverName, targetServer.GameType, user.Credits-1, serverName,
	))
}
