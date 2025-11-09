package commands

import (
	"fmt"
	"runtime"
	"strings"
	"time"

	"agis-bot/internal/services"
	"agis-bot/internal/version"
)

// ProfileCommand shows user profile with stats
type ProfileCommand struct{}

func (c *ProfileCommand) Name() string { return "profile" }
func (c *ProfileCommand) Description() string { return "View user profile and statistics" }
func (c *ProfileCommand) RequiredPermission() bot.Permission { return bot.PermissionUser }

func (c *ProfileCommand) Execute(ctx *CommandContext) error {
	targetUserID := ctx.Message.Author.ID
	targetUsername := ctx.Message.Author.Username

	// Allow viewing other users' profiles
	if len(ctx.Args) > 0 && len(ctx.Message.Mentions) > 0 {
		targetUserID = ctx.Message.Mentions[0].ID
		targetUsername = ctx.Message.Mentions[0].Username
	}

	user, err := ctx.DB.GetOrCreateUser(targetUserID)
	if err != nil {
		return fmt.Errorf("failed to get user: %v", err)
	}

	servers, err := ctx.DB.GetUserServers(targetUserID)
	if err != nil {
		return fmt.Errorf("failed to get servers: %v", err)
	}

	activeServers := 0
	for _, srv := range servers {
		if srv.Status == "running" || srv.Status == "ready" {
			activeServers++
		}
	}

	// Get stats (with fallback if table doesn't exist yet)
	var totalCreated, totalCommands int
	row := ctx.DB.DB().QueryRow(`
		SELECT COALESCE(total_servers_created, 0), COALESCE(total_commands_used, 0)
		FROM user_stats WHERE discord_id = $1
	`, targetUserID)
	_ = row.Scan(&totalCreated, &totalCommands)

	joinedAgo := time.Since(user.JoinDate)
	days := int(joinedAgo.Hours() / 24)

	profile := fmt.Sprintf(
		"👤 **Profile: %s**\n"+
			"━━━━━━━━━━━━━━━━━━━━\n"+
			"💰 **Credits:** %d\n"+
			"🎮 **Tier:** %s\n"+
			"📅 **Joined:** %d days ago\n"+
			"━━━━━━━━━━━━━━━━━━━━\n"+
			"**Server Statistics**\n"+
			"• Total Created: %d\n"+
			"• Currently Active: %d\n"+
			"• Total Owned: %d\n"+
			"━━━━━━━━━━━━━━━━━━━━\n"+
			"**Activity**\n"+
			"• Commands Used: %d\n"+
			"• Last Daily: %s ago\n"+
			"• Last Work: %s ago",
		targetUsername,
		user.Credits,
		strings.ToUpper(user.Tier),
		days,
		totalCreated,
		activeServers,
		len(servers),
		totalCommands,
		formatDurationV1_3(time.Since(user.LastDaily)),
		formatDurationV1_3(time.Since(user.LastWork)),
	)

	return ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, profile)
}

// InfoAboutCommand shows bot information
type InfoAboutCommand struct {
	startTime time.Time
}

func NewInfoAboutCommand(startTime time.Time) *InfoAboutCommand {
	return &InfoAboutCommand{startTime: startTime}
}

func (c *InfoAboutCommand) Name() string { return "about" }
func (c *InfoAboutCommand) Description() string { return "Bot information and statistics" }
func (c *InfoAboutCommand) RequiredPermission() bot.Permission { return bot.PermissionUser }

func (c *InfoAboutCommand) Execute(ctx *CommandContext) error {
	uptime := time.Since(c.startTime)
	buildInfo := version.GetBuildInfo()

	var memStats runtime.MemStats
	runtime.ReadMemStats(&memStats)

	// Get total users and servers
	var totalUsers, totalServers, activeServers int
	ctx.DB.DB().QueryRow(`SELECT COUNT(*) FROM users`).Scan(&totalUsers)
	ctx.DB.DB().QueryRow(`SELECT COUNT(*) FROM game_servers`).Scan(&totalServers)
	ctx.DB.DB().QueryRow(`SELECT COUNT(*) FROM game_servers WHERE status IN ('running', 'ready')`).Scan(&activeServers)

	info := fmt.Sprintf(
		"🤖 **AGIS Bot Information**\n"+
			"━━━━━━━━━━━━━━━━━━━━\n"+
			"**Version:** %s\n"+
			"**Build:** %s\n"+
			"**Built:** %s\n"+
			"**Uptime:** %s\n"+
			"━━━━━━━━━━━━━━━━━━━━\n"+
			"**Platform Statistics**\n"+
			"• Total Users: %d\n"+
			"• Total Servers: %d\n"+
			"• Active Servers: %d\n"+
			"━━━━━━━━━━━━━━━━━━━━\n"+
			"**System**\n"+
			"• Memory Usage: %.1f MB\n"+
			"• Goroutines: %d\n"+
			"• Go Version: %s\n"+
			"━━━━━━━━━━━━━━━━━━━━\n"+
			"Powered by Kubernetes & Agones",
		buildInfo.Version,
		buildInfo.Commit[:7],
		buildInfo.BuildDate,
		formatDurationV1_3(uptime),
		totalUsers,
		totalServers,
		activeServers,
		float64(memStats.Alloc)/1024/1024,
		runtime.NumGoroutine(),
		runtime.Version(),
	)

	return ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, info)
}

// InfoGamesCommand lists supported games
type InfoGamesCommand struct{}

func (c *InfoGamesCommand) Name() string { return "games" }
func (c *InfoGamesCommand) Description() string { return "List supported games and pricing" }
func (c *InfoGamesCommand) RequiredPermission() bot.Permission { return bot.PermissionUser }

func (c *InfoGamesCommand) Execute(ctx *CommandContext) error {
	games := `🎮 **Supported Games**
━━━━━━━━━━━━━━━━━━━━

**Minecraft** - Java Edition
• Cost: 5 credits/hour
• Default Port: 25565
• Features: Mods, Plugins, Custom worlds

**Counter-Strike 2 (CS2)**
• Cost: 8 credits/hour
• Default Port: 27015
• Features: Custom maps, Competitive mode

**Terraria**
• Cost: 3 credits/hour
• Default Port: 7777
• Features: Multiplayer worlds, Mods support

**Garry's Mod**
• Cost: 6 credits/hour
• Default Port: 27015
• Features: Custom gamemodes, Addons

━━━━━━━━━━━━━━━━━━━━
Use ` + "`create <game> [name]`" + ` to deploy a server`

	return ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, games)
}

// LeaderboardCommand shows leaderboards
type LeaderboardCommand struct{}

func (c *LeaderboardCommand) Name() string { return "leaderboard" }
func (c *LeaderboardCommand) Description() string { return "View leaderboards (credits, servers)" }
func (c *LeaderboardCommand) RequiredPermission() bot.Permission { return bot.PermissionUser }

func (c *LeaderboardCommand) Execute(ctx *CommandContext) error {
	lbType := "credits"
	if len(ctx.Args) > 0 {
		lbType = strings.ToLower(ctx.Args[0])
	}

	switch lbType {
	case "credits", "credit":
		return c.showCreditsLeaderboard(ctx)
	case "servers", "server":
		return c.showServersLeaderboard(ctx)
	default:
		return fmt.Errorf("usage: leaderboard [credits|servers]")
	}
}

func (c *LeaderboardCommand) showCreditsLeaderboard(ctx *CommandContext) error {
	rows, err := ctx.DB.DB().Query(`
		SELECT discord_id, credits 
		FROM users 
		ORDER BY credits DESC 
		LIMIT 10
	`)
	if err != nil {
		return fmt.Errorf("failed to fetch leaderboard: %v", err)
	}
	defer rows.Close()

	var board strings.Builder
	board.WriteString("🏆 **Credits Leaderboard**\n━━━━━━━━━━━━━━━━━━━━\n")

	position := 1
	for rows.Next() {
		var userID string
		var credits int
		if err := rows.Scan(&userID, &credits); err != nil {
			continue
		}

		// Get username
		user, err := ctx.Session.User(userID)
		username := userID
		if err == nil && user != nil {
			username = user.Username
		}

		medal := getMedal(position)
		board.WriteString(fmt.Sprintf("%s **#%d** %s - %d credits\n", medal, position, username, credits))
		position++
	}

	return ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, board.String())
}

func (c *LeaderboardCommand) showServersLeaderboard(ctx *CommandContext) error {
	rows, err := ctx.DB.DB().Query(`
		SELECT discord_id, COUNT(*) as server_count
		FROM game_servers
		GROUP BY discord_id
		ORDER BY server_count DESC
		LIMIT 10
	`)
	if err != nil {
		return fmt.Errorf("failed to fetch leaderboard: %v", err)
	}
	defer rows.Close()

	var board strings.Builder
	board.WriteString("🏆 **Servers Leaderboard**\n━━━━━━━━━━━━━━━━━━━━\n")

	position := 1
	for rows.Next() {
		var userID string
		var count int
		if err := rows.Scan(&userID, &count); err != nil {
			continue
		}

		user, err := ctx.Session.User(userID)
		username := userID
		if err == nil && user != nil {
			username = user.Username
		}

		medal := getMedal(position)
		board.WriteString(fmt.Sprintf("%s **#%d** %s - %d servers\n", medal, position, username, count))
		position++
	}

	return ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, board.String())
}

// StartServerCommand starts a stopped server
type StartServerCommand struct{}

func (c *StartServerCommand) Name() string { return "start" }
func (c *StartServerCommand) Description() string { return "Start a stopped server" }
func (c *StartServerCommand) RequiredPermission() bot.Permission { return bot.PermissionUser }

func (c *StartServerCommand) Execute(ctx *CommandContext) error {
	if len(ctx.Args) == 0 {
		return fmt.Errorf("usage: start <server-name>")
	}

	serverName := ctx.Args[0]
	servers, err := ctx.DB.GetUserServers(ctx.Message.Author.ID)
	if err != nil {
		return fmt.Errorf("failed to get servers: %v", err)
	}

	var targetServer *services.GameServer
	for _, srv := range servers {
		if srv.Name == serverName {
			targetServer = srv
			break
		}
	}

	if targetServer == nil {
		return fmt.Errorf("server '%s' not found", serverName)
	}

	if targetServer.Status != "stopped" {
		return fmt.Errorf("server is already %s", targetServer.Status)
	}

	// Update server status
	if err := ctx.DB.UpdateServerStatus(targetServer.ID, "creating"); err != nil {
		return fmt.Errorf("failed to start server: %v", err)
	}

	// Clear stopped timestamp
	ctx.DB.UpdateServerField(targetServer.ID, "stopped_at", nil)

	return ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, fmt.Sprintf(
		"▶️ Starting server `%s`...\nUse `diagnostics %s` to check status.",
		serverName, serverName,
	))
}

// ServerLogsCommand - DEPRECATED: Replaced by K8sLogsCommand in v1.6.0
// This is kept for backwards compatibility but should not be registered

// Helper functions
func formatDurationV1_3(d time.Duration) string {
	if d < time.Minute {
		return "just now"
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}

func getMedal(position int) string {
	switch position {
	case 1:
		return "🥇"
	case 2:
		return "🥈"
	case 3:
		return "🥉"
	default:
		return "  "
	}
}
