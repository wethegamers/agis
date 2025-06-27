package commands

import (
	"fmt"
	"strings"

	"agis-bot/internal/bot"

	"github.com/bwmarrin/discordgo"
)

type CreditsEarnCommand struct{}

func (c *CreditsEarnCommand) Name() string {
	return "earn"
}

func (c *CreditsEarnCommand) Description() string {
	return "Access the ad dashboard to earn credits"
}

func (c *CreditsEarnCommand) RequiredPermission() bot.Permission {
	return bot.PermissionUser
}

func (c *CreditsEarnCommand) Execute(ctx *CommandContext) error {
	user, err := ctx.DB.GetOrCreateUser(ctx.Message.Author.ID)
	if err != nil {
		return fmt.Errorf("failed to get user: %v", err)
	}

	// Calculate subscriber benefits
	var subscriberBenefits string
	var adEarningRate string

	if user.Tier == "free" {
		subscriberBenefits = "• **Monthly Credits:** None\n• **Ad Multiplier:** 1x\n• **Server Time:** Limited by credits"
		adEarningRate = "**50-75 credits** per ad"
	} else {
		subscriberBenefits = "• **Monthly Credits:** 100 free credits\n• **Ad Multiplier:** 2x earnings\n• **Server Time:** Unlimited for $0.99/month"
		adEarningRate = "**100-150 credits** per ad (2x multiplier)"
	}

	embed := &discordgo.MessageEmbed{
		Title:       "🎥 Earn Credits - Ad Dashboard",
		Description: "Watch short video advertisements to earn credits for game servers!",
		Color:       0x00ff00,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "💰 Current Balance",
				Value:  fmt.Sprintf("%d credits", user.Credits),
				Inline: true,
			},
			{
				Name:   "🎯 Earning Rate",
				Value:  adEarningRate,
				Inline: true,
			},
			{
				Name:   "⏱️ Ad Duration",
				Value:  "15-30 seconds each",
				Inline: true,
			},
			{
				Name:   "🌐 Access Dashboard",
				Value:  fmt.Sprintf("**[🎥 Open Ad Dashboard](%s/ads?user=%s)**", ctx.Config.WTG.DashboardURL, user.DiscordID),
				Inline: false,
			},
			{
				Name:   "📋 How It Works",
				Value:  "1️⃣ Click the dashboard link above\n2️⃣ Watch short video advertisements\n3️⃣ Earn credits automatically\n4️⃣ Return to Discord to deploy servers!",
				Inline: false,
			},
			{
				Name:   fmt.Sprintf("👑 %s Tier Benefits", strings.ToUpper(user.Tier)),
				Value:  subscriberBenefits,
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "💡 Credits earned from ads are processed instantly • Upgrade for 2x earnings!",
		},
	}

	// Add upgrade call-to-action for free users
	if user.Tier == "free" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "🚀 Want More?",
			Value:  "Upgrade to **Premium** for just **$0.99/month** to get:\n• **100 free credits monthly**\n• **2x ad earnings**\n• **Unlimited server time**\n• **No credit limits**",
			Inline: false,
		})
	}

	_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}
