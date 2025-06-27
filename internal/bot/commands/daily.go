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

	// Calculate daily reward (base + tier bonus)
	baseReward := 25
	tierBonus := 0
	if user.Tier == "premium" {
		tierBonus = 15 // Premium users get +15 bonus
	}
	totalReward := baseReward + tierBonus

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
				Name:   "💰 Reward Breakdown",
				Value:  fmt.Sprintf("Base: %d credits\nTier Bonus: %d credits\n**Total: %d credits**", baseReward, tierBonus, totalReward),
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
			Text: "💡 Premium members get +15 bonus credits daily! Upgrade with 'upgrade premium'",
		},
	}

	if user.Tier == "premium" {
		embed.Footer.Text = "⭐ Premium bonus applied! Thanks for supporting WTG!"
	}

	_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}
