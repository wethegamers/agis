package commands

import (
	"fmt"
	"log"
	"time"

	"agis-bot/internal/bot"
	"agis-bot/internal/services"
)

type RestartServerCommand struct{}

func (c *RestartServerCommand) Name() string {
	return "restart"
}

func (c *RestartServerCommand) Description() string {
	return "Restart a running server"
}

func (c *RestartServerCommand) RequiredPermission() bot.Permission {
	return bot.PermissionUser
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

	servers, err := ctx.DB.GetUserServers(ctx.Message.Author.ID)
	if err != nil {
		return fmt.Errorf("failed to get your servers: %v", err)
	}

	var targetServer *services.GameServer
	for _, srv := range servers {
		if srv.Name == serverName {
			targetServer = srv
			break
		}
	}

	if targetServer == nil {
		return fmt.Errorf("server '%s' not found. Use `servers` to list your servers", serverName)
	}

	if targetServer.Status == "stopped" {
		return fmt.Errorf("server '%s' is stopped. Use `start %s` to start it", serverName, serverName)
	}

	if targetServer.Status != "running" && targetServer.Status != "ready" {
		return fmt.Errorf("server '%s' is currently %s. Can only restart running servers", serverName, targetServer.Status)
	}

	if user.Credits < 1 {
		return fmt.Errorf("insufficient credits. Need 1 credit for restart. You have %d credits", user.Credits)
	}

	if err := ctx.DB.UpdateServerStatus(targetServer.Name, targetServer.DiscordID, "stopping"); err != nil {
		return fmt.Errorf("failed to update server status: %v", err)
	}

	now := time.Now()
	if err := ctx.DB.UpdateServerStoppedAt(targetServer.ID, &now); err != nil {
		if ctx.Logger != nil {
			ctx.Logger.LogError("restart_update_stopped_at_failed", "Failed to set stopped_at during restart", map[string]interface{}{
				"server_id":   targetServer.ID,
				"server_name": serverName,
				"error":       err.Error(),
			})
		} else {
			log.Printf("Failed to update stopped_at for server %s: %v", serverName, err)
		}
	}

	ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, fmt.Sprintf("⏸️ Stopping server '%s' for restart...", serverName))

	time.Sleep(3 * time.Second)

	if err := ctx.DB.UpdateServerStatus(targetServer.Name, targetServer.DiscordID, "creating"); err != nil {
		return fmt.Errorf("failed to restart server: %v", err)
	}

	if err := ctx.DB.UpdateServerStoppedAt(targetServer.ID, nil); err != nil {
		if ctx.Logger != nil {
			ctx.Logger.LogError("restart_clear_stopped_at_failed", "Failed to clear stopped_at during restart", map[string]interface{}{
				"server_id":   targetServer.ID,
				"server_name": serverName,
				"error":       err.Error(),
			})
		} else {
			log.Printf("Failed to clear stopped_at for server %s: %v", serverName, err)
		}
	}

	if err := ctx.DB.DeductCredits(ctx.Message.Author.ID, 1); err != nil {
		if ctx.Logger != nil {
			ctx.Logger.LogError("restart_credit_deduct_failed", "Failed to deduct restart credit", map[string]interface{}{
				"server_id":   targetServer.ID,
				"server_name": serverName,
				"user_id":     ctx.Message.Author.ID,
				"error":       err.Error(),
			})
		} else {
			log.Printf("Failed to deduct restart credit for server %s: %v", serverName, err)
		}
	} else {
		user.Credits--
	}

	if ctx.Logger != nil {
		ctx.Logger.LogUser(ctx.Message.Author.ID, "server_restarted", fmt.Sprintf("User restarted server %s", serverName), map[string]interface{}{
			"server_id":   targetServer.ID,
			"server_name": serverName,
			"game_type":   targetServer.GameType,
		})
	} else {
		log.Printf("User %s restarted server %s", ctx.Message.Author.ID, serverName)
	}

	_, err = ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, fmt.Sprintf(
		"🔄 **Server Restarted**\n"+
			"Server: `%s`\n"+
			"Game: %s\n"+
			"Status: Restarting\n"+
			"Cost: 1 credit\n"+
			"New Balance: %d credits\n\n"+
			"Server will be online shortly. Use `diagnostics %s` to check status.",
		serverName, targetServer.GameType, user.Credits, serverName,
	))
	return err
}
