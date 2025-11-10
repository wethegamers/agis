package commands

import (
	"fmt"
	"strconv"
	"strings"

	"agis-bot/internal/bot"
	"agis-bot/internal/services"

	"github.com/bwmarrin/discordgo"
)

// ScheduleCommand manages server schedules
type ScheduleCommand struct{}

func (c *ScheduleCommand) Name() string {
	return "schedule"
}

func (c *ScheduleCommand) Description() string {
	return "Schedule automatic server start/stop/restart"
}

func (c *ScheduleCommand) RequiredPermission() bot.Permission {
	return bot.PermissionUser
}

func (c *ScheduleCommand) Execute(ctx *CommandContext) error {
	if len(ctx.Args) == 0 {
		return c.showScheduleHelp(ctx)
	}

	serverName := ctx.Args[0]

	// Get server
	servers, err := ctx.DB.GetUserServers(ctx.Message.Author.ID)
	if err != nil {
		return fmt.Errorf("failed to get servers: %v", err)
	}

	var server *services.GameServer
	for _, s := range servers {
		if strings.EqualFold(s.Name, serverName) {
			server = s
			break
		}
	}

	if server == nil {
		return fmt.Errorf("server not found: %s", serverName)
	}

	if len(ctx.Args) < 2 {
		return c.listServerSchedules(ctx, server)
	}

	action := strings.ToLower(ctx.Args[1])

	switch action {
	case "start", "stop", "restart":
		if len(ctx.Args) < 3 {
			return fmt.Errorf("cron expression required. Example: schedule %s %s \"0 8 * * *\"", serverName, action)
		}
		return c.createSchedule(ctx, server, action, ctx.Args[2])
	case "list":
		return c.listServerSchedules(ctx, server)
	case "delete", "remove":
		if len(ctx.Args) < 3 {
			return fmt.Errorf("schedule ID required. Use 'schedule %s list' to see IDs", serverName)
		}
		scheduleID, err := strconv.Atoi(ctx.Args[2])
		if err != nil {
			return fmt.Errorf("invalid schedule ID: %s", ctx.Args[2])
		}
		return c.deleteSchedule(ctx, scheduleID)
	case "enable":
		if len(ctx.Args) < 3 {
			return fmt.Errorf("schedule ID required")
		}
		scheduleID, err := strconv.Atoi(ctx.Args[2])
		if err != nil {
			return fmt.Errorf("invalid schedule ID: %s", ctx.Args[2])
		}
		return c.enableSchedule(ctx, scheduleID)
	case "disable":
		if len(ctx.Args) < 3 {
			return fmt.Errorf("schedule ID required")
		}
		scheduleID, err := strconv.Atoi(ctx.Args[2])
		if err != nil {
			return fmt.Errorf("invalid schedule ID: %s", ctx.Args[2])
		}
		return c.disableSchedule(ctx, scheduleID)
	default:
		return fmt.Errorf("unknown action: %s. Use: start, stop, restart, list, delete", action)
	}
}

func (c *ScheduleCommand) showScheduleHelp(ctx *CommandContext) error {
	embed := &discordgo.MessageEmbed{
		Title:       "📅 Server Scheduling System",
		Description: "Automate your server management with cron-like scheduling",
		Color:       0x5865F2,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name: "Create Schedule",
				Value: "```\n" +
					"schedule <server> start \"0 8 * * *\"\n" +
					"schedule <server> stop \"0 23 * * *\"\n" +
					"schedule <server> restart \"0 */6 * * *\"\n" +
					"```",
				Inline: false,
			},
			{
				Name: "Manage Schedules",
				Value: "```\n" +
					"schedule <server> list\n" +
					"schedule <server> delete <id>\n" +
					"schedule <server> enable <id>\n" +
					"schedule <server> disable <id>\n" +
					"```",
				Inline: false,
			},
			{
				Name: "Cron Format",
				Value: "```\n" +
					"┌─── minute (0-59)\n" +
					"│ ┌─── hour (0-23)\n" +
					"│ │ ┌─── day of month (1-31)\n" +
					"│ │ │ ┌─── month (1-12)\n" +
					"│ │ │ │ ┌─── day of week (0-6) (Sunday=0)\n" +
					"│ │ │ │ │\n" +
					"* * * * *\n" +
					"```",
				Inline: false,
			},
			{
				Name: "Examples",
				Value: "• `0 8 * * *` - Daily at 8:00 AM\n" +
					"• `0 23 * * *` - Daily at 11:00 PM\n" +
					"• `0 */6 * * *` - Every 6 hours\n" +
					"• `0 9 * * 1` - Every Monday at 9:00 AM\n" +
					"• `0 0 1 * *` - First day of month at midnight",
				Inline: false,
			},
			{
				Name: "Tips",
				Value: "• All times in UTC\n" +
					"• Use quotes around cron expressions\n" +
					"• Check next run time after creating\n" +
					"• Disable schedules when not needed",
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Use https://crontab.guru to help build cron expressions",
		},
	}

	_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}

func (c *ScheduleCommand) createSchedule(ctx *CommandContext, server *services.GameServer, action, cronExpr string) error {
	if ctx.SchedulerService == nil {
		return fmt.Errorf("scheduler service not available - contact administrator")
	}

	// Remove quotes if present
	cronExpr = strings.Trim(cronExpr, "\"'")

	schedule, err := ctx.SchedulerService.CreateSchedule(
		server.ID,
		ctx.Message.Author.ID,
		action,
		cronExpr,
		"UTC",
	)

	if err != nil {
		return fmt.Errorf("failed to create schedule: %v", err)
	}

	embed := &discordgo.MessageEmbed{
		Title:       "✅ Schedule Created",
		Description: fmt.Sprintf("Scheduled **%s** for server **%s**", action, server.Name),
		Color:       0x00ff00,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Schedule ID",
				Value:  fmt.Sprintf("%d", schedule.ID),
				Inline: true,
			},
			{
				Name:   "Action",
				Value:  strings.Title(action),
				Inline: true,
			},
			{
				Name:   "Cron Expression",
				Value:  fmt.Sprintf("`%s`", cronExpr),
				Inline: true,
			},
			{
				Name:   "Next Run",
				Value:  schedule.NextRun.Format("2006-01-02 15:04 MST"),
				Inline: false,
			},
			{
				Name:   "Status",
				Value:  "✅ Enabled",
				Inline: true,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Use 'schedule %s list' to view all schedules", server.Name),
		},
	}

	_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}

func (c *ScheduleCommand) listServerSchedules(ctx *CommandContext, server *services.GameServer) error {
	if ctx.SchedulerService == nil {
		return fmt.Errorf("scheduler service not available")
	}

	schedules, err := ctx.SchedulerService.GetServerSchedules(server.ID, ctx.Message.Author.ID)
	if err != nil {
		return fmt.Errorf("failed to list schedules: %v", err)
	}

	if len(schedules) == 0 {
		embed := &discordgo.MessageEmbed{
			Title:       "📅 No Schedules",
			Description: fmt.Sprintf("No schedules found for server **%s**", server.Name),
			Color:       0xFFA500,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name: "Create One",
					Value: "```\n" +
						fmt.Sprintf("schedule %s start \"0 8 * * *\"\n", server.Name) +
						fmt.Sprintf("schedule %s stop \"0 23 * * *\"\n", server.Name) +
						"```",
					Inline: false,
				},
			},
		}
		_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
		return err
	}

	fields := make([]*discordgo.MessageEmbedField, 0, len(schedules))
	for _, schedule := range schedules {
		status := "✅ Enabled"
		if !schedule.Enabled {
			status = "⏸️ Disabled"
		}

		nextRun := "Not scheduled"
		if schedule.NextRun != nil {
			nextRun = schedule.NextRun.Format("2006-01-02 15:04 MST")
		}

		lastRun := "Never"
		if schedule.LastRun != nil {
			lastRun = schedule.LastRun.Format("2006-01-02 15:04 MST")
		}

		fields = append(fields, &discordgo.MessageEmbedField{
			Name: fmt.Sprintf("Schedule #%d - %s", schedule.ID, strings.Title(schedule.Action)),
			Value: fmt.Sprintf(
				"**Cron:** `%s`\n**Status:** %s\n**Next Run:** %s\n**Last Run:** %s",
				schedule.CronExpression,
				status,
				nextRun,
				lastRun,
			),
			Inline: false,
		})
	}

	embed := &discordgo.MessageEmbed{
		Title:       fmt.Sprintf("📅 Schedules for %s", server.Name),
		Description: fmt.Sprintf("Found %d schedule(s)", len(schedules)),
		Color:       0x5865F2,
		Fields:      fields,
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Use 'schedule %s delete <id>' to remove a schedule", server.Name),
		},
	}

	_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}

func (c *ScheduleCommand) deleteSchedule(ctx *CommandContext, scheduleID int) error {
	if ctx.SchedulerService == nil {
		return fmt.Errorf("scheduler service not available")
	}

	err := ctx.SchedulerService.DeleteSchedule(scheduleID, ctx.Message.Author.ID)
	if err != nil {
		return err
	}

	embed := &discordgo.MessageEmbed{
		Title:       "🗑️ Schedule Deleted",
		Description: fmt.Sprintf("Schedule #%d has been deleted", scheduleID),
		Color:       0xFF0000,
	}

	_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}

func (c *ScheduleCommand) enableSchedule(ctx *CommandContext, scheduleID int) error {
	if ctx.SchedulerService == nil {
		return fmt.Errorf("scheduler service not available")
	}

	err := ctx.SchedulerService.EnableSchedule(scheduleID, ctx.Message.Author.ID)
	if err != nil {
		return err
	}

	embed := &discordgo.MessageEmbed{
		Title:       "✅ Schedule Enabled",
		Description: fmt.Sprintf("Schedule #%d is now active", scheduleID),
		Color:       0x00ff00,
	}

	_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}

func (c *ScheduleCommand) disableSchedule(ctx *CommandContext, scheduleID int) error {
	if ctx.SchedulerService == nil {
		return fmt.Errorf("scheduler service not available")
	}

	err := ctx.SchedulerService.DisableSchedule(scheduleID, ctx.Message.Author.ID)
	if err != nil {
		return err
	}

	embed := &discordgo.MessageEmbed{
		Title:       "⏸️ Schedule Disabled",
		Description: fmt.Sprintf("Schedule #%d has been paused", scheduleID),
		Color:       0xFFA500,
	}

	_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}
