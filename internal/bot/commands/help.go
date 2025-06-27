package commands

import (
	"agis-bot/internal/bot"

	"github.com/bwmarrin/discordgo"
)

type HelpCommand struct{}

func (c *HelpCommand) Name() string {
	return "help"
}

func (c *HelpCommand) Description() string {
	return "Shows available commands"
}

func (c *HelpCommand) RequiredPermission() bot.Permission {
	return bot.PermissionUser
}

func (c *HelpCommand) Execute(ctx *CommandContext) error {
	embed := &discordgo.MessageEmbed{
		Title:       "🤖 Agis - WTG Cluster Management Bot",
		Description: "Your gateway to the We The Gamers platform! Available commands:",
		Color:       0xffd700,
	}

	// Add fields based on user permission level
	userFields := []*discordgo.MessageEmbedField{
		{
			Name:   "**🎮 Quick Start**",
			Value:  "`help` - Show this help menu\n`credits` - Check your balance\n`credits earn` - 🎥 **Watch ads for 50-150 credits each!**\n`create minecraft` - Deploy your first server",
			Inline: false,
		},
		{
			Name:   "**💰 Earn Credits**",
			Value:  "🎥 `credits earn` - **Ad dashboard (best earnings!)**\n🔧 `work` - Infrastructure tasks (1h cooldown)",
			Inline: false,
		},
		{
			Name:   "**🎮 Server Management**",
			Value:  "`servers` - List your servers\n`create <game> [name]` - Deploy new server (minecraft/cs2/terraria/gmod)\n`stop <server>` - Stop server to save credits\n`delete <server>` - Delete your own server permanently\n`export <server>` - Export save files before cleanup",
			Inline: false,
		},
		{
			Name:   "**🔧 Diagnostics & Testing**",
			Value:  "`diagnostics <server>` - Complete server health check\n`ping [server]` - Test connectivity to bot or server",
			Inline: false,
		},
		{
			Name:   "**🌐 Public Lobby**",
			Value:  "`lobby list` - Browse all public servers\n`lobby add <server> [description]` - Share your server publicly\n`lobby remove <server>` - Make server private\n`lobby my` - View your public servers",
			Inline: false,
		},
	}

	modFields := []*discordgo.MessageEmbedField{
		{
			Name:   "**🛡️ Moderator Commands**",
			Value:  "`mod-servers` - View all user servers across platform\n`mod-control <user> <server> <action>` - Control any user's server\n• Actions: stop, restart, info, logs\n`mod-delete <server-id>` - Delete a user's server\n`confirm-delete <server-id>` - Confirm server deletion",
			Inline: false,
		},
	}

	adminFields := []*discordgo.MessageEmbedField{
		{
			Name:   "**⚙️ Admin Commands**",
			Value:  "`admin status` - Backend cluster health and status\n`admin pods` - List Kubernetes pods\n`admin nodes` - List cluster nodes\n`admin-restart` - Restart the AGIS bot\n`admin-restart confirm` - Confirm restart\n`admin-restart confirm --force` - Force restart",
			Inline: false,
		},
		{
			Name:   "**💰 Credit Management**",
			Value:  "`admin credits add @user <amount>` - Add credits to user\n`admin credits remove @user <amount>` - Remove credits\n`admin credits check @user` - Check user balance",
			Inline: false,
		},
	}

	ownerFields := []*discordgo.MessageEmbedField{
		{
			Name:   "**👑 Owner Commands**",
			Value:  "`owner set-admin <@role>` - Set admin role\n`owner set-mod <@role>` - Set moderator role\n`owner list-roles` - Show configured roles\n`owner remove-admin <@role>` - Remove admin role\n`owner remove-mod <@role>` - Remove moderator role",
			Inline: false,
		},
	}

	// Add user fields (everyone can see these)
	embed.Fields = append(embed.Fields, userFields...)

	// Add mod fields if user is mod or admin
	if ctx.UserPerm >= bot.PermissionMod {
		embed.Fields = append(embed.Fields, modFields...)
	}

	// Add admin fields if user is admin
	if ctx.UserPerm >= bot.PermissionAdmin {
		embed.Fields = append(embed.Fields, adminFields...)
	}

	// Add owner fields if user is owner
	if ctx.UserPerm >= bot.PermissionOwner {
		embed.Fields = append(embed.Fields, ownerFields...)
	}

	// Add game types and costs info
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
		Name:   "**🎮 Game Types & Costs**",
		Value:  "• **Minecraft:** 5 credits/hour\n• **CS2:** 8 credits/hour\n• **Terraria:** 3 credits/hour\n• **GMod:** 6 credits/hour",
		Inline: true,
	})

	// Add business model info
	embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
		Name:   "**💎 WTG Business Model**",
		Value:  "🆓 **Free Tier:** Earn credits through ads & work\n💰 **Premium ($0.99/month):** Unlimited servers + 2x ad earnings + 100 monthly credits",
		Inline: true,
	})

	embed.Footer = &discordgo.MessageEmbedFooter{
		Text: "💡 Best way to earn credits: 'credits earn' for ad dashboard • Your permission: " + bot.GetPermissionString(ctx.UserPerm),
	}

	_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}
