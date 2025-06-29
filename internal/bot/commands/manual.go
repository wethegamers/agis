package commands

import (
	"agis-bot/internal/bot"

	"github.com/bwmarrin/discordgo"
)

type ManualCommand struct{}

func (c *ManualCommand) Name() string {
	return "manual"
}

func (c *ManualCommand) Description() string {
	return "Show detailed command manual and examples"
}

func (c *ManualCommand) RequiredPermission() bot.Permission {
	return bot.PermissionUser
}

func (c *ManualCommand) Execute(ctx *CommandContext) error {
	embed := &discordgo.MessageEmbed{
		Title:       "📖 AGIS Bot - Complete Manual",
		Description: "Comprehensive documentation for all commands and features",
		Color:       0x4169e1,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "📚 **Documentation**",
				Value:  "Full manual available at: **COMMANDS.md**\n[View on GitHub](https://github.com/wethegamers/agis-bot/blob/main/COMMANDS.md)",
				Inline: false,
			},
			{
				Name:   "🎯 **Quick Start Guide**",
				Value:  "1. `credits` - Check your balance\n2. `create minecraft` - Deploy your first server\n3. `diagnostics <server>` - Monitor deployment\n4. `servers` - View all your servers\n5. `lobby add <server>` - Share with community",
				Inline: false,
			},
			{
				Name:   "🎮 **Supported Games**",
				Value:  "• **Minecraft** (5 credits/hour)\n• **CS2** (8 credits/hour)\n• **Terraria** (3 credits/hour)\n• **Garry's Mod** (6 credits/hour)",
				Inline: true,
			},
			{
				Name:   "🔧 **Key Features**",
				Value:  "• Live Kubernetes integration\n• Real-time server status\n• Automated Agones deployment\n• Enhanced diagnostics\n• Public lobby system",
				Inline: true,
			},
			{
				Name:   "💡 **Pro Tips**",
				Value:  "• Use `diagnostics` for detailed server health\n• `credits earn` provides best earnings\n• `stop` servers when not playing to save credits\n• `export` saves before server cleanup\n• `lobby` to discover community servers",
				Inline: false,
			},
			{
				Name:   "🆘 **Need Help?**",
				Value:  "• `help` - Quick command overview\n• `ping` - Test connectivity\n• `diagnostics <server>` - Server troubleshooting\n• Contact admins for technical support",
				Inline: false,
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "AGIS Bot - Powered by Kubernetes & Agones | Your permission: " + bot.GetPermissionString(ctx.UserPerm),
		},
	}

	_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}

// ManCommand is an alias for ManualCommand
type ManCommand struct{}

func (c *ManCommand) Name() string {
	return "man"
}

func (c *ManCommand) Description() string {
	return "Show detailed command manual (alias for manual)"
}

func (c *ManCommand) RequiredPermission() bot.Permission {
	return bot.PermissionUser
}

func (c *ManCommand) Execute(ctx *CommandContext) error {
	manualCmd := &ManualCommand{}
	return manualCmd.Execute(ctx)
}
