package commands

import (
	"fmt"
	"log"
	"time"

	"agis-bot/internal/bot"

	"github.com/bwmarrin/discordgo"
)

type WorkCommand struct{}

func (c *WorkCommand) Name() string {
	return "work"
}

func (c *WorkCommand) Description() string {
	return "Earn credits through infrastructure work (1 hour cooldown)"
}

func (c *WorkCommand) RequiredPermission() bot.Permission {
	return bot.PermissionUser
}

func (c *WorkCommand) Execute(ctx *CommandContext) error {
	user, err := ctx.DB.GetOrCreateUser(ctx.Message.Author.ID)
	if err != nil {
		return fmt.Errorf("failed to get user: %v", err)
	}

	// Debug log for troubleshooting cooldown issues
	log.Printf("🔍 Work Debug - User: %s, LastWork: %v, TimeSince: %v, Cooldown: %v",
		ctx.Message.Author.ID, user.LastWork, time.Since(user.LastWork), 1*time.Hour)

	// Check work cooldown (1 hour) - ensure we handle zero time properly
	cooldownDuration := 1 * time.Hour
	timeSinceLastWork := time.Since(user.LastWork)

	// Handle zero time (when last_work is '1970-01-01' or unset)
	if user.LastWork.IsZero() || user.LastWork.Year() < 2000 {
		log.Printf("🔍 Work Debug - LastWork is zero/invalid, allowing work")
	} else if timeSinceLastWork < cooldownDuration {
		timeLeft := cooldownDuration - timeSinceLastWork

		embed := &discordgo.MessageEmbed{
			Title:       "⏰ Work Cooldown Active",
			Description: "You need to wait before earning more credits through work.",
			Color:       0xffa500,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "Time Remaining",
					Value:  fmt.Sprintf("%d minutes", int(timeLeft.Minutes())+1),
					Inline: true,
				},
				{
					Name:   "🎥 Earn More Credits Now!",
					Value:  "Watch short video ads on our dashboard to earn **50-100 credits** per ad with no cooldown!",
					Inline: false,
				},
				{
					Name:   "Alternative Options",
					Value:  "• Use `daily` for free credits (24h cooldown)\n• Use `credits earn` for instant ad rewards",
					Inline: false,
				},
			},
			Footer: &discordgo.MessageEmbedFooter{
				Text: "💡 Premium subscribers earn 2x credits from ads! Upgrade for just $0.99/month",
			},
		}
		_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
		return err
	}

	// Work command now directs users to the ad-reward dashboard
	// Based on business plan: "Users watch short, rewarded video advertisements on a simple, dedicated web dashboard"

	// Simulate work completion and give base credits
	workCredits := 15 // Base work credits (smaller than ad rewards)

	// Apply premium 2x multiplier
	multiplier := GetUserMultiplier(ctx.DB.DB(), ctx.Message.Author.ID)
	finalCredits := workCredits * multiplier

	user.Credits += finalCredits
	user.LastWork = time.Now()

	// Update user in database
	if err := ctx.DB.UpdateUserWork(user.DiscordID, user.Credits, user.LastWork); err != nil {
		log.Printf("Failed to update user work: %v", err)
	}

	embed := &discordgo.MessageEmbed{
		Title:       "💼 Work Complete!",
		Description: "You helped maintain the WTG cluster infrastructure and earned credits!",
		Color:       0x00ffff,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Credits Earned",
				Value:  fmt.Sprintf("+%d credits", finalCredits),
				Inline: true,
			},
			{
				Name:   "Multiplier",
				Value:  fmt.Sprintf("%dx%s", multiplier, func() string { if multiplier > 1 { return " 👑" } else { return "" } }()),
				Inline: true,
			},
			{
				Name:   "New Balance",
				Value:  fmt.Sprintf("%d credits", user.Credits),
				Inline: true,
			},
			{
				Name:   "🎥 Want More Credits?",
				Value:  fmt.Sprintf("Visit our [Ad Dashboard](%s/ads?user=%s) to watch short videos and earn **50-100 credits** per ad!", ctx.Config.WTG.DashboardURL, user.DiscordID),
				Inline: false,
			},
			{
				Name:   "💡 Pro Tip",
				Value:  "Premium subscribers earn **2x credits** from ads and get **100 free credits monthly**!",
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "⏰ Work available again in 1 hour • Use 'credits earn' for ad dashboard",
		},
	}

	_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}
