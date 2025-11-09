package services

import (
	"context"
	"database/sql"
	"log"
	"time"
)

// RewardAlgorithm calculates dynamic ad rewards based on user context and engagement
type RewardAlgorithm struct {
	db *DatabaseService
}

// NewRewardAlgorithm creates a new reward algorithm instance
func NewRewardAlgorithm(db *DatabaseService) *RewardAlgorithm {
	return &RewardAlgorithm{db: db}
}

// RewardContext holds user context for reward calculation
type RewardContext struct {
	DiscordID           string
	IsPremium           bool
	UserTier            string
	RecentConversions   int    // Last 24 hours
	TotalConversions    int    // All time
	AverageRewardAmount int    // Historical average
	DaysSinceJoined     int
	LastConversionHours int    // Hours since last conversion
	ProviderCurrency    string
	ProviderAmount      int
	ConversionType      string // "offerwall", "surveywall", "video"
}

// RewardCalculation holds the final reward breakdown
type RewardCalculation struct {
	BaseReward       int     // Provider amount after currency conversion
	TierMultiplier   float64 // Free=1.0, Premium=1.5-2.0
	EngagementBonus  int     // Bonus for active users
	NewUserBonus     int     // Bonus for new users (first 7 days)
	TypeMultiplier   float64 // Different rates per ad type
	FinalReward      int     // Total Game Credits awarded
	Explanation      string  // Human-readable breakdown
}

// CalculateReward computes the final reward amount with all modifiers
func (r *RewardAlgorithm) CalculateReward(ctx context.Context, rewardCtx RewardContext) (*RewardCalculation, error) {
	calc := &RewardCalculation{}

	// 1. Base conversion from provider currency to Game Credits
	calc.BaseReward = r.convertCurrency(rewardCtx.ProviderCurrency, rewardCtx.ProviderAmount)

	// 2. Tier multiplier (Premium = 1.5x-2.0x)
	calc.TierMultiplier = r.getTierMultiplier(rewardCtx.UserTier, rewardCtx.IsPremium)

	// 3. Engagement bonus (reward active users, incentivize returning users)
	calc.EngagementBonus = r.calculateEngagementBonus(rewardCtx)

	// 4. New user bonus (first 7 days get 20% bonus to hook them)
	calc.NewUserBonus = r.calculateNewUserBonus(rewardCtx)

	// 5. Type multiplier (videos pay less, offers pay more)
	calc.TypeMultiplier = r.getTypeMultiplier(rewardCtx.ConversionType)

	// 6. Calculate final reward
	baseWithTier := float64(calc.BaseReward) * calc.TierMultiplier
	baseWithType := baseWithTier * calc.TypeMultiplier
	calc.FinalReward = int(baseWithType) + calc.EngagementBonus + calc.NewUserBonus

	// Apply floor and ceiling
	if calc.FinalReward < 1 {
		calc.FinalReward = 1 // Minimum 1 GC
	}
	if calc.FinalReward > 10000 {
		calc.FinalReward = 10000 // Cap at 10k GC per conversion
	}

	// 7. Build explanation
	calc.Explanation = r.buildExplanation(calc)

	return calc, nil
}

// convertCurrency converts provider currency to Game Credits
func (r *RewardAlgorithm) convertCurrency(currency string, amount int) int {
	// Base conversion rates - can be made configurable via database
	switch currency {
	case "coins", "gold", "points":
		return amount // 1:1
	case "gems":
		return amount * 2 // 1 gem = 2 GC
	case "cash", "dollars":
		return amount * 100 // $1 = 100 GC
	default:
		return amount // Default 1:1
	}
}

// getTierMultiplier returns multiplier based on user tier
func (r *RewardAlgorithm) getTierMultiplier(tier string, isPremium bool) float64 {
	// Premium subscribers get enhanced rewards
	if isPremium {
		switch tier {
		case "premium_plus":
			return 2.0 // 2x rewards
		case "premium":
			return 1.5 // 1.5x rewards
		default:
			return 1.5 // Default premium = 1.5x
		}
	}

	// Free tier multipliers
	switch tier {
	case "verified":
		return 1.1 // Verified users get small bonus
	case "free":
		return 1.0 // Standard rate
	default:
		return 1.0
	}
}

// calculateEngagementBonus rewards active users
func (r *RewardAlgorithm) calculateEngagementBonus(ctx RewardContext) int {
	bonus := 0

	// Return user bonus (24+ hours since last conversion)
	if ctx.LastConversionHours >= 24 && ctx.TotalConversions > 0 {
		bonus += 50 // Welcome back bonus
	}

	// Streak bonus (5+ conversions in last 7 days)
	if ctx.RecentConversions >= 5 {
		bonus += 25 // Active user bonus
	}

	// Long-term user bonus (30+ days since join)
	if ctx.DaysSinceJoined >= 30 {
		bonus += 10 // Loyalty bonus
	}

	return bonus
}

// calculateNewUserBonus gives new users a boost
func (r *RewardAlgorithm) calculateNewUserBonus(ctx RewardContext) int {
	// First 7 days get 20% bonus on base reward
	if ctx.DaysSinceJoined <= 7 {
		return int(float64(ctx.ProviderAmount) * 0.2)
	}
	return 0
}

// getTypeMultiplier adjusts rewards based on ad type
func (r *RewardAlgorithm) getTypeMultiplier(adType string) float64 {
	switch adType {
	case "offerwall":
		return 1.0 // Standard rate
	case "surveywall":
		return 1.2 // Surveys pay more (higher engagement)
	case "video":
		return 0.8 // Videos pay less (lower effort)
	default:
		return 1.0
	}
}

// buildExplanation creates human-readable breakdown
func (r *RewardAlgorithm) buildExplanation(calc *RewardCalculation) string {
	explanation := ""
	
	if calc.TierMultiplier > 1.0 {
		explanation += "Premium bonus active. "
	}
	if calc.EngagementBonus > 0 {
		explanation += "Engagement bonus earned. "
	}
	if calc.NewUserBonus > 0 {
		explanation += "New user bonus applied. "
	}
	if calc.TypeMultiplier != 1.0 {
		explanation += "Ad type modifier applied. "
	}

	if explanation == "" {
		explanation = "Standard reward rate"
	}

	return explanation
}

// LoadRewardContext fetches user context from database
func (r *RewardAlgorithm) LoadRewardContext(ctx context.Context, discordID string, providerCurrency string, providerAmount int, conversionType string) (*RewardContext, error) {
	rewardCtx := &RewardContext{
		DiscordID:        discordID,
		ProviderCurrency: providerCurrency,
		ProviderAmount:   providerAmount,
		ConversionType:   conversionType,
	}

	// Local mode - return defaults
	if r.db.LocalMode() {
		rewardCtx.IsPremium = false
		rewardCtx.UserTier = "free"
		rewardCtx.DaysSinceJoined = 1
		return rewardCtx, nil
	}

	// Fetch user data
	var joinDate time.Time
	var lastConversion sql.NullTime
	err := r.db.DB().QueryRowContext(ctx,
		`SELECT 
			tier,
			join_date,
			COALESCE((SELECT MAX(created_at) FROM ad_conversions WHERE discord_id = $1), NULL) as last_conversion
		FROM users WHERE discord_id = $1`,
		discordID,
	).Scan(&rewardCtx.UserTier, &joinDate, &lastConversion)

	if err != nil && err != sql.ErrNoRows {
		// If user doesn't exist, use defaults (will be created on credit)
		rewardCtx.UserTier = "free"
		rewardCtx.DaysSinceJoined = 0
		rewardCtx.IsPremium = false
	} else {
		rewardCtx.DaysSinceJoined = int(time.Since(joinDate).Hours() / 24)
		rewardCtx.IsPremium = (rewardCtx.UserTier == "premium" || rewardCtx.UserTier == "premium_plus")
		
		if lastConversion.Valid {
			rewardCtx.LastConversionHours = int(time.Since(lastConversion.Time).Hours())
		} else {
			rewardCtx.LastConversionHours = 9999 // No previous conversion
		}
	}

	// Count recent conversions (last 24h)
	err = r.db.DB().QueryRowContext(ctx,
		`SELECT COUNT(*) FROM ad_conversions 
		 WHERE discord_id = $1 AND created_at > NOW() - INTERVAL '24 hours' AND status = 'completed'`,
		discordID,
	).Scan(&rewardCtx.RecentConversions)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("Warning: failed to count recent conversions: %v", err)
	}

	// Count total conversions
	err = r.db.DB().QueryRowContext(ctx,
		`SELECT COUNT(*) FROM ad_conversions 
		 WHERE discord_id = $1 AND status = 'completed'`,
		discordID,
	).Scan(&rewardCtx.TotalConversions)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("Warning: failed to count total conversions: %v", err)
	}

	// Calculate average reward
	err = r.db.DB().QueryRowContext(ctx,
		`SELECT COALESCE(AVG(amount), 0) FROM ad_conversions 
		 WHERE discord_id = $1 AND status = 'completed'`,
		discordID,
	).Scan(&rewardCtx.AverageRewardAmount)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("Warning: failed to calculate average reward: %v", err)
	}

	return rewardCtx, nil
}

// GetRewardEstimate provides an estimate without full context (for UI)
func (r *RewardAlgorithm) GetRewardEstimate(providerCurrency string, providerAmount int, userTier string, isPremium bool) int {
	baseReward := r.convertCurrency(providerCurrency, providerAmount)
	tierMultiplier := r.getTierMultiplier(userTier, isPremium)
	return int(float64(baseReward) * tierMultiplier)
}
