package commands

import (
	"fmt"
	"time"

	"agis-bot/internal/bot"

	"github.com/bwmarrin/discordgo"
)

// DailyCommand provides daily credit rewards
type DailyCommand struct{}

func (c *DailyCommand) Name() string {
	return "daily"
}

func (c *DailyCommand) Description() string {
	return "Claim your daily credit bonus"
}

func (c *DailyCommand) RequiredPermission() bot.Permission {
	return bot.PermissionUser
}

func (c *DailyCommand) Execute(ctx *CommandContext) error {
	user, err := ctx.DB.GetOrCreateUser(ctx.Message.Author.ID)
	if err != nil {
		return fmt.Errorf("failed to get user: %v", err)
	}

	now := time.Now()
	timeSinceLastDaily := now.Sub(user.LastDaily)

	if timeSinceLastDaily < 24*time.Hour && !user.LastDaily.IsZero() {
		timeUntilNext := 24*time.Hour - timeSinceLastDaily
		hours := int(timeUntilNext.Hours())
		minutes := int(timeUntilNext.Minutes()) % 60

		embed := &discordgo.MessageEmbed{
			Title:       "⏰ Daily Cooldown Active",
			Description: "You've already claimed your daily bonus today!",
			Color:       0xffa500,
			Fields: []*discordgo.MessageEmbedField{
				{
					Name:   "⏱️ Next Daily Available",
					Value:  fmt.Sprintf("In %d hours and %d minutes", hours, minutes),
					Inline: true,
				},
				{
					Name:   "💰 Current Balance",
					Value:  fmt.Sprintf("%d credits", user.Credits),
					Inline: true,
				},
				{
					Name:   "💡 Pro Tip",
					Value:  "Try `work` or `credits earn` while you wait!",
					Inline: false,
				},
			},
			Footer: &discordgo.MessageEmbedFooter{
				Text: "Daily bonuses reset every 24 hours",
			},
		}

		_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
		return err
	}

	// Calculate daily reward (50 GC base, 100 GC for premium)
	baseReward := 50
	totalReward := baseReward
	
	// Check for active premium subscription
	isPremium := HasActivePremium(ctx.DB.DB(), ctx.Message.Author.ID)
	if isPremium {
		totalReward = 100 // Premium gets double
	}

	// Add credits
	err = ctx.DB.AddCredits(ctx.Message.Author.ID, totalReward)
	if err != nil {
		return fmt.Errorf("failed to add daily credits: %v", err)
	}

	// Update last daily time
	err = ctx.DB.UpdateLastDaily(ctx.Message.Author.ID)
	if err != nil {
		return fmt.Errorf("failed to update daily timestamp: %v", err)
	}

	// Get updated user balance
	updatedUser, err := ctx.DB.GetOrCreateUser(ctx.Message.Author.ID)
	if err != nil {
		return fmt.Errorf("failed to get updated user: %v", err)
	}

	embed := &discordgo.MessageEmbed{
		Title:       "🎁 Daily Bonus Claimed!",
		Description: fmt.Sprintf("You've earned **%d credits** from your daily bonus!", totalReward),
		Color:       0x00ff00,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "💰 Daily Reward",
				Value:  fmt.Sprintf("**%d GameCredits**%s", totalReward, func() string { if isPremium { return " 👑" } else { return "" } }()),
				Inline: true,
			},
			{
				Name:   "📊 Your Balance",
				Value:  fmt.Sprintf("New Balance: **%d credits**\nPrevious: %d credits", updatedUser.Credits, user.Credits),
				Inline: true,
			},
			{
				Name:   "⏰ Next Daily",
				Value:  "Available in 24 hours",
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: func() string {
				if isPremium {
					return "⭐ Premium 2x multiplier applied! Thanks for supporting WTG!"
				}
				return "💎 Premium members get 100 GC daily (2x)! Use 'subscribe' to upgrade"
			}(),
		},
	}

	_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}
