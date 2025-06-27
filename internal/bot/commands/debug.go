package commands

import (
	"fmt"

	"agis-bot/internal/bot"

	"github.com/bwmarrin/discordgo"
)

// DebugPermissionsCommand shows current user permissions for debugging
type DebugPermissionsCommand struct{}

func (c *DebugPermissionsCommand) Name() string {
	return "debug-perms"
}

func (c *DebugPermissionsCommand) Description() string {
	return "Debug: Show current user permissions"
}

func (c *DebugPermissionsCommand) RequiredPermission() bot.Permission {
	return bot.PermissionUser
}

func (c *DebugPermissionsCommand) Execute(ctx *CommandContext) error {
	embed := &discordgo.MessageEmbed{
		Title:       "🔍 Permission Debug Information",
		Description: "Current user permission details",
		Color:       0x0099ff,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:   "Your Discord ID",
				Value:  ctx.Message.Author.ID,
				Inline: true,
			},
			{
				Name:   "Your Permission Level",
				Value:  fmt.Sprintf("%s (%d)", bot.GetPermissionString(ctx.UserPerm), int(ctx.UserPerm)),
				Inline: true,
			},
			{
				Name:   "Bot Owner ID",
				Value:  "290955794172739584",
				Inline: true,
			},
			{
				Name:   "Is Owner?",
				Value:  fmt.Sprintf("%t", ctx.Message.Author.ID == "290955794172739584"),
				Inline: true,
			},
		},
	}

	// Add guild info if available
	if ctx.Message.GuildID != "" {
		embed.Fields = append(embed.Fields, &discordgo.MessageEmbedField{
			Name:   "Guild ID",
			Value:  ctx.Message.GuildID,
			Inline: true,
		})
	}

	_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}
