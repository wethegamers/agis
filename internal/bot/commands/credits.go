package commands

import (
	"fmt"
	"strings"

	"agis-bot/internal/bot"

	"github.com/bwmarrin/discordgo"
)

type CreditsCommand struct{}

func (c *CreditsCommand) Name() string {
	return "credits"
}

func (c *CreditsCommand) Description() string {
	return "Check your credit balance"
}

func (c *CreditsCommand) RequiredPermission() bot.Permission {
	return bot.PermissionUser
}

func (c *CreditsCommand) Execute(ctx *CommandContext) error {
	// Check if this is a subcommand
	if len(ctx.Args) > 0 && strings.ToLower(ctx.Args[0]) == "earn" {
		earnCmd := &CreditsEarnCommand{}
		return earnCmd.Execute(ctx)
	}

	user, err := ctx.DB.GetOrCreateUser(ctx.Message.Author.ID)
	if err != nil {
		return fmt.Errorf("failed to get user: %v", err)
	}

	// Calculate estimated server time based on current credits
	minecraftHours := user.Credits / 50 // 50 credits per hour for Minecraft
	arkHours := user.Credits / 75       // 75 credits per hour for ARK

	embed := &discordgo.MessageEmbed{
		Title: "💰 Your WTG Credits",
		Color: 0x00ff00,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Current Balance",
				Value:  fmt.Sprintf("**%d credits**", user.Credits),
				Inline: true,
			},
			{
				Name:   "Account Tier",
				Value:  strings.Title(user.Tier),
				Inline: true,
			},
			{
				Name:   "Active Servers",
				Value:  fmt.Sprintf("%d deployed", user.ServersUsed),
				Inline: true,
			},
			{
				Name:   "⏱️ Estimated Server Time",
				Value:  fmt.Sprintf("• **Minecraft:** %d hours\n• **ARK:** %d hours", minecraftHours, arkHours),
				Inline: false,
			},
			{
				Name:   "🎥 Earn Credits Fast",
				Value:  "**[Watch Ads - 50-150 credits each!]** Use `credits earn`",
				Inline: false,
			},
			{
				Name:   "🔄 Other Ways to Earn",
				Value:  "• `daily` - Free daily credits (24h cooldown)\n• `work` - Infrastructure work (1h cooldown)",
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "💰 Premium subscribers get 100 free credits monthly + 2x ad earnings!",
		},
	}

	_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}
