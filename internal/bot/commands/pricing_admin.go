package commands

import (
	"fmt"
	"strconv"
	"strings"

	"agis-bot/internal/bot"

	"github.com/bwmarrin/discordgo"
)

// PricingCommand allows admins to view and manage game pricing
type PricingCommand struct{}

func (c *PricingCommand) Name() string {
	return "pricing"
}

func (c *PricingCommand) Description() string {
	return "Manage game server pricing (Admin only)"
}

func (c *PricingCommand) RequiredPermission() bot.Permission {
	return bot.PermissionAdmin
}

func (c *PricingCommand) Execute(ctx *CommandContext) error {
	if ctx.PricingService == nil {
		return fmt.Errorf("pricing service not available")
	}

	if len(ctx.Args) == 0 {
		return c.showPricing(ctx)
	}

	subcommand := strings.ToLower(ctx.Args[0])

	switch subcommand {
	case "list":
		return c.showPricing(ctx)
	case "update":
		return c.updatePricing(ctx)
	case "add":
		return c.addGameType(ctx)
	case "disable":
		return c.disableGameType(ctx)
	default:
		return fmt.Errorf("unknown subcommand. Use: list, update, add, disable")
	}
}

func (c *PricingCommand) showPricing(ctx *CommandContext) error {
	allPricing := ctx.PricingService.GetAllPricing()

	if len(allPricing) == 0 {
		embed := &discordgo.MessageEmbed{
			Title:       "💰 Game Pricing",
			Description: "No active game types configured",
			Color:       0xffaa00,
		}
		_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
		return err
	}

	fields := make([]*discordgo.MessageEmbedField, 0, len(allPricing))
	for _, pricing := range allPricing {
		fields = append(fields, &discordgo.MessageEmbedField{
			Name: fmt.Sprintf("%s (%s)", pricing.DisplayName, pricing.GameType),
			Value: fmt.Sprintf(
				"**Cost:** %d GC/hour\n**Min Credits:** %d GC\n**Status:** %s\n**Description:** %s",
				pricing.CostPerHour,
				pricing.MinCredits,
				map[bool]string{true: "Active ✅", false: "Inactive ❌"}[pricing.IsActive],
				pricing.Description,
			),
			Inline: true,
		})
	}

	embed := &discordgo.MessageEmbed{
		Title:       "💰 Current Game Pricing",
		Description: "Database-backed pricing configuration",
		Color:       0x00ff00,
		Fields:      fields,
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Use 'pricing update <game> <cost> [min]' to change pricing",
		},
	}

	_, err := ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}

func (c *PricingCommand) updatePricing(ctx *CommandContext) error {
	// Usage: pricing update minecraft 10 10
	if len(ctx.Args) < 3 {
		return fmt.Errorf("usage: pricing update <game-type> <cost-per-hour> [min-credits]")
	}

	gameType := strings.ToLower(ctx.Args[1])
	costPerHour, err := strconv.Atoi(ctx.Args[2])
	if err != nil {
		return fmt.Errorf("invalid cost-per-hour: %s", ctx.Args[2])
	}

	minCredits := costPerHour // Default to same as cost
	if len(ctx.Args) >= 4 {
		minCredits, err = strconv.Atoi(ctx.Args[3])
		if err != nil {
			return fmt.Errorf("invalid min-credits: %s", ctx.Args[3])
		}
	}

	// Validate game exists
	if !ctx.PricingService.IsValidGameType(gameType) {
		return fmt.Errorf("game type '%s' does not exist. Use 'pricing add' to create it", gameType)
	}

	// Update pricing
	err = ctx.PricingService.UpdatePricing(gameType, costPerHour, minCredits)
	if err != nil {
		return fmt.Errorf("failed to update pricing: %v", err)
	}

	embed := &discordgo.MessageEmbed{
		Title:       "✅ Pricing Updated",
		Description: fmt.Sprintf("Pricing for '%s' has been updated", gameType),
		Color:       0x00ff00,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:  "Game Type",
				Value: gameType,
			},
			{
				Name:  "New Cost Per Hour",
				Value: fmt.Sprintf("%d GC", costPerHour),
			},
			{
				Name:  "Minimum Credits",
				Value: fmt.Sprintf("%d GC", minCredits),
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "⚠️ Changes take effect immediately. Existing servers unaffected.",
		},
	}

	_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}

func (c *PricingCommand) addGameType(ctx *CommandContext) error {
	// Usage: pricing add valheim "Valheim" "Viking survival game" 10 10
	if len(ctx.Args) < 5 {
		return fmt.Errorf("usage: pricing add <game-type> <display-name> <description> <cost-per-hour> [min-credits]")
	}

	gameType := strings.ToLower(ctx.Args[1])
	displayName := ctx.Args[2]
	description := ctx.Args[3]
	costPerHour, err := strconv.Atoi(ctx.Args[4])
	if err != nil {
		return fmt.Errorf("invalid cost-per-hour: %s", ctx.Args[4])
	}

	minCredits := costPerHour
	if len(ctx.Args) >= 6 {
		minCredits, err = strconv.Atoi(ctx.Args[5])
		if err != nil {
			return fmt.Errorf("invalid min-credits: %s", ctx.Args[5])
		}
	}

	// Add game type
	err = ctx.PricingService.AddGameType(gameType, displayName, description, costPerHour, minCredits)
	if err != nil {
		return fmt.Errorf("failed to add game type: %v", err)
	}

	embed := &discordgo.MessageEmbed{
		Title:       "✅ Game Type Added",
		Description: fmt.Sprintf("New game type '%s' has been added to pricing", gameType),
		Color:       0x00ff00,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:  "Game Type",
				Value: gameType,
			},
			{
				Name:  "Display Name",
				Value: displayName,
			},
			{
				Name:  "Description",
				Value: description,
			},
			{
				Name:  "Cost Per Hour",
				Value: fmt.Sprintf("%d GC", costPerHour),
			},
			{
				Name:  "Minimum Credits",
				Value: fmt.Sprintf("%d GC", minCredits),
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "⚠️ Don't forget to deploy the Docker image and Agones GameServer manifest!",
		},
	}

	_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}

func (c *PricingCommand) disableGameType(ctx *CommandContext) error {
	// Usage: pricing disable minecraft
	if len(ctx.Args) < 2 {
		return fmt.Errorf("usage: pricing disable <game-type>")
	}

	gameType := strings.ToLower(ctx.Args[1])

	// Validate game exists
	if !ctx.PricingService.IsValidGameType(gameType) {
		return fmt.Errorf("game type '%s' does not exist or is already disabled", gameType)
	}

	// Disable game type
	err := ctx.PricingService.DisableGameType(gameType)
	if err != nil {
		return fmt.Errorf("failed to disable game type: %v", err)
	}

	embed := &discordgo.MessageEmbed{
		Title:       "✅ Game Type Disabled",
		Description: fmt.Sprintf("Game type '%s' has been disabled", gameType),
		Color:       0xffaa00,
		Fields: []*discordgo.MessageEmbedField{
			{
				Name:  "Game Type",
				Value: gameType,
			},
			{
				Name:  "Status",
				Value: "Inactive ❌",
			},
		},
		Footer: &discordgo.MessageEmbedFooter{
			Text: "Existing servers will continue to run. New servers cannot be created.",
		},
	}

	_, err = ctx.Session.ChannelMessageSendEmbed(ctx.Message.ChannelID, embed)
	return err
}
