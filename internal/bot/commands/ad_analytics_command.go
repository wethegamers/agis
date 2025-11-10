package commands

import (
	"context"
	"fmt"
	"time"

	"agis-bot/internal/bot"
	"agis-bot/internal/services"
)

// AdAnalyticsCommand shows admin analytics for ad conversions
type AdAnalyticsCommand struct {
	adService *services.AdConversionService
}

// NewAdAnalyticsCommand creates a new ad analytics command
func NewAdAnalyticsCommand(adService *services.AdConversionService) *AdAnalyticsCommand {
	return &AdAnalyticsCommand{
		adService: adService,
	}
}

func (c *AdAnalyticsCommand) Name() string {
	return "ad-analytics"
}

func (c *AdAnalyticsCommand) Description() string {
	return "View ad conversion analytics and statistics (Admin only)"
}

func (c *AdAnalyticsCommand) RequiredPermission() bot.Permission {
	return bot.PermissionAdmin
}

func (c *AdAnalyticsCommand) Execute(ctx *CommandContext) error {
	// For slash command compatibility
	if ctx.Session == nil {
		return fmt.Errorf("no session available")
	}
	
	// Send ephemeral message with analytics
	response := c.buildAnalyticsResponse(ctx)
	_, err := ctx.Session.ChannelMessageSend(ctx.Message.ChannelID, response)
	return err
}

func (c *AdAnalyticsCommand) buildAnalyticsResponse(ctx *CommandContext) string {
	loadCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Get conversion statistics
	stats, err := c.adService.GetConversionStats(loadCtx)
	if err != nil {
		return "❌ Failed to retrieve ad analytics: " + err.Error()
	}

	// Build analytics message
	return c.buildAnalyticsMessage(stats)
}

func (c *AdAnalyticsCommand) buildAnalyticsMessage(stats *services.ConversionStats) string {
	// Calculate rates
	var conversionRate float64
	if stats.TotalConversions > 0 {
		conversionRate = 100.0 // Assume 100% for now (can add callback attempts later)
	}

	var fraudRate float64
	totalAttempts := stats.TotalConversions + stats.FraudAttempts
	if totalAttempts > 0 {
		fraudRate = float64(stats.FraudAttempts) / float64(totalAttempts) * 100
	}

	var avgReward float64
	if stats.TotalConversions > 0 {
		avgReward = float64(stats.TotalRewards) / float64(stats.TotalConversions)
	}

	// Build message
	message := "📊 **Ad Conversion Analytics**\n\n"

	// Overall stats
	message += "**📈 Overall Performance**\n"
	message += fmt.Sprintf("• Total Conversions: **%d**\n", stats.TotalConversions)
	message += fmt.Sprintf("• Total Rewards: **%d GC**\n", stats.TotalRewards)
	message += fmt.Sprintf("• Unique Users: **%d**\n", stats.UniqueUsers)
	message += fmt.Sprintf("• Average Reward: **%.1f GC**\n", avgReward)
	message += fmt.Sprintf("• Conversion Rate: **%.1f%%**\n\n", conversionRate)

	// Security stats
	message += "**🛡️ Security & Fraud**\n"
	message += fmt.Sprintf("• Fraud Attempts Blocked: **%d**\n", stats.FraudAttempts)
	message += fmt.Sprintf("• Fraud Rate: **%.2f%%**\n", fraudRate)
	message += fmt.Sprintf("• Clean Conversions: **%d** (%.1f%%)\n\n", stats.TotalConversions, 100.0-fraudRate)

	// 24h activity
	message += "**⏱️ Last 24 Hours**\n"
	message += fmt.Sprintf("• Conversions: **%d**\n", stats.Conversions24h)
	message += fmt.Sprintf("• Rewards Issued: **%d GC**\n", stats.Rewards24h)

	var avgReward24h float64
	if stats.Conversions24h > 0 {
		avgReward24h = float64(stats.Rewards24h) / float64(stats.Conversions24h)
	}
	message += fmt.Sprintf("• Average: **%.1f GC**\n\n", avgReward24h)

	// By type breakdown
	if len(stats.ByType) > 0 {
		message += "**🎯 By Ad Type**\n"
		for adType, typeStats := range stats.ByType {
			var typeAvg float64
			if typeStats.Count > 0 {
				typeAvg = float64(typeStats.Rewards) / float64(typeStats.Count)
			}
			emoji := getAdTypeEmoji(adType)
			message += fmt.Sprintf("%s **%s**: %d conversions, %d GC (%.1f avg)\n",
				emoji, capitalizeFirst(adType), typeStats.Count, typeStats.Rewards, typeAvg)
		}
		message += "\n"
	}

	// Revenue insights (placeholder for future)
	message += "**💰 Revenue Insights**\n"
	message += "• Estimated Provider Revenue: *Data not available*\n"
	message += "• Estimated Fill Rate: *Data not available*\n"
	message += "• Top Performing Type: " + c.getTopPerformingType(stats) + "\n\n"

	// Timestamp
	message += fmt.Sprintf("_Updated: <t:%d:R>_", time.Now().Unix())

	return message
}

func (c *AdAnalyticsCommand) getTopPerformingType(stats *services.ConversionStats) string {
	if len(stats.ByType) == 0 {
		return "No data"
	}

	maxCount := 0
	topType := ""
	for adType, typeStats := range stats.ByType {
		if typeStats.Count > maxCount {
			maxCount = typeStats.Count
			topType = adType
		}
	}

	if topType == "" {
		return "No data"
	}

	return fmt.Sprintf("**%s** (%d conversions)", capitalizeFirst(topType), maxCount)
}

func getAdTypeEmoji(adType string) string {
	switch adType {
	case "offerwall":
		return "🎁"
	case "surveywall":
		return "📋"
	case "video":
		return "🎬"
	default:
		return "📊"
	}
}

func capitalizeFirst(s string) string {
	if len(s) == 0 {
		return s
	}
	return string(s[0]-32) + s[1:]
}

func stringPtr(s string) *string {
	return &s
}
