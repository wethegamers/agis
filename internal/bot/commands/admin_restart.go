package commands

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
	"time"

	"agis-bot/internal/bot"

	"github.com/bwmarrin/discordgo"
)

// AdminRestartCommand allows admins to restart the AGIS bot
type AdminRestartCommand struct{}

func (c *AdminRestartCommand) Name() string {
	return "admin-restart"
}

func (c *AdminRestartCommand) Description() string {
	return "Restart the AGIS bot"
}

func (c *AdminRestartCommand) RequiredPermission() bot.Permission {
	return bot.PermissionAdmin
}

func (c *AdminRestartCommand) Execute(ctx *CommandContext) error {
	// Check for confirmation flag
	confirmed := false
	forceFlag := false

	for _, arg := range ctx.Args {
		if strings.EqualFold(arg, "confirm") {
			confirmed = true
		}
		if strings.EqualFold(arg, "--force") || strings.EqualFold(arg, "-f") {
			forceFlag = true
		}
	}

	if !confirmed {
		return c.showConfirmation(ctx, forceFlag)
	}

	// Send a message that the bot is restarting
	embed := &discordgo.MessageEmbed{
		Title:       "🔄 Restarting AGIS Bot",
		Description: "The bot is now restarting. It will be back online shortly.",
		Color:       0xffa500,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Status",
				Value:  "⏳ Shutting down services...",
				Inline: false,
			},
			{
				Name:   "ETA",
				Value:  "Bot should be back online in 10-30 seconds",
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("Restart initiated by %s • %s", ctx.Message.Author.Username, time.Now().Format(time.RFC1123)),
		},
	}

	_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	if err != nil {
		log.Printf("Failed to send restart message: %v", err)
		// Continue with restart anyway
	}

	// Log the restart event
	log.Printf("🔄 Bot restart initiated by %s (ID: %s)", ctx.Message.Author.Username, ctx.Message.Author.ID)

	// Allow time for the message to be sent
	time.Sleep(2 * time.Second)

	// Perform the restart
	if forceFlag {
		// Force restart using exec.Command - more reliable in case of issues
		c.performForceRestart()
	} else {
		// Signal the main process to restart gracefully
		c.performGracefulRestart()
	}

	return nil
}

func (c *AdminRestartCommand) showConfirmation(ctx *CommandContext, forceFlag bool) error {
	restartType := "Normal"
	description := "This will gracefully restart the bot, allowing it to close connections properly."

	if forceFlag {
		restartType = "Force"
		description = "⚠️ This is a **force restart** that will immediately terminate and restart the bot process."
	}

	var confirmValue string
	if forceFlag {
		confirmValue = "Type `admin-restart confirm --force` to proceed"
	} else {
		confirmValue = "Type `admin-restart confirm` to proceed"
	}

	embed := &discordgo.MessageEmbed{
		Title:       "🔄 Confirm Bot Restart",
		Description: description,
		Color:       0xff9900,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Impact",
				Value:  "• Bot will be unavailable for a few seconds\n• Some in-progress operations may be interrupted\n• No data loss will occur",
				Inline: false,
			},
			{
				Name:   "Confirm Action",
				Value:  confirmValue,
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: fmt.Sprintf("%s restart • Admin permissions required", restartType),
		},
	}

	_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}

func (c *AdminRestartCommand) performGracefulRestart() {
	// Set an environment variable that the parent process can detect
	os.Setenv("AGIS_BOT_RESTART", "1")

	// Exit with a special code that can be detected by a wrapper script
	log.Println("🔄 Performing graceful restart...")
	os.Exit(42) // Special exit code for restart
}

func (c *AdminRestartCommand) performForceRestart() {
	execPath, err := os.Executable()
	if err != nil {
		log.Printf("❌ Failed to get executable path: %v", err)
		os.Exit(1)
		return
	}

	log.Println("🔄 Performing force restart...")

	// Start a new instance of the bot
	cmd := exec.Command(execPath, os.Args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()

	if err := cmd.Start(); err != nil {
		log.Printf("❌ Failed to start new process: %v", err)
		os.Exit(1)
		return
	}

	// Exit the current process
	os.Exit(0)
}
