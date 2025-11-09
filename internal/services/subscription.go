package services

import (
	"database/sql"
	"fmt"
	"log"
	"time"
)

// SubscriptionService manages premium subscriptions and automatic benefit application
// BLOCKER 8: Zero-touch subscription management
type SubscriptionService struct {
	db *DatabaseService
}

// NewSubscriptionService creates a new subscription service
func NewSubscriptionService(db *DatabaseService) *SubscriptionService {
	return &SubscriptionService{db: db}
}

// SubscriptionTier represents a user's subscription level
type SubscriptionTier string

const (
	TierFree    SubscriptionTier = "free"
	TierPremium SubscriptionTier = "premium"
)

// PremiumBenefits defines the benefits of premium subscription (Economy Plan v4.0)
const (
	PremiumPrice        = 3.99 // USD per month
	PremiumWTGAllowance = 5    // WTG coins granted monthly
	PremiumGCMultiplier = 3    // 3x multiplier for ads/work (was 2x in old docs)
	PremiumDailyBonus   = 100  // GameCredits (vs 50 for free)
	PremiumFreeServer   = 3000 // GC/month server rent waived
)

// ActivateSubscription activates premium subscription for a user
// Called automatically when Stripe webhook confirms payment
func (s *SubscriptionService) ActivateSubscription(discordID string, durationDays int) error {
	if s.db.LocalMode() {
		return fmt.Errorf("subscriptions not available in local mode")
	}

	tx, err := s.db.DB().Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	expiresAt := time.Now().Add(time.Duration(durationDays) * 24 * time.Hour)

	// Update user tier and expiration
	_, err = tx.Exec(`
		UPDATE users 
		SET tier = 'premium', 
		    subscription_expires = $1,
		    wtg_coins = wtg_coins + $2,
		    updated_at = CURRENT_TIMESTAMP
		WHERE discord_id = $3
	`, expiresAt, PremiumWTGAllowance, discordID)
	if err != nil {
		return fmt.Errorf("failed to activate subscription: %v", err)
	}

	// Log WTG allowance transaction
	_, err = tx.Exec(`
		INSERT INTO credit_transactions (from_user, to_user, amount, transaction_type, description, currency_type)
		VALUES ('SYSTEM', $1, $2, 'subscription', 'Premium subscription activated - monthly WTG allowance', 'WTG')
	`, discordID, PremiumWTGAllowance)
	if err != nil {
		log.Printf("Warning: Failed to log WTG allowance: %v", err)
		// Non-fatal - continue
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit subscription: %v", err)
	}

	log.Printf("✅ Premium subscription activated: %s (expires %s)", discordID, expiresAt.Format("2006-01-02"))
	return nil
}

// RenewSubscription renews an existing subscription
// Called when user pays for another month (Stripe recurring payment)
func (s *SubscriptionService) RenewSubscription(discordID string) error {
	if s.db.LocalMode() {
		return fmt.Errorf("subscriptions not available in local mode")
	}

	tx, err := s.db.DB().Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %v", err)
	}
	defer tx.Rollback()

	// Get current expiration
	var currentExpires sql.NullTime
	err = tx.QueryRow(`SELECT subscription_expires FROM users WHERE discord_id = $1`, discordID).Scan(&currentExpires)
	if err != nil {
		return fmt.Errorf("failed to get current subscription: %v", err)
	}

	// New expiration: 30 days from current expiration (or now if expired)
	var newExpires time.Time
	if currentExpires.Valid && currentExpires.Time.After(time.Now()) {
		newExpires = currentExpires.Time.Add(30 * 24 * time.Hour)
	} else {
		newExpires = time.Now().Add(30 * 24 * time.Hour)
	}

	// Update subscription and add monthly WTG allowance
	_, err = tx.Exec(`
		UPDATE users 
		SET tier = 'premium', 
		    subscription_expires = $1,
		    wtg_coins = wtg_coins + $2,
		    updated_at = CURRENT_TIMESTAMP
		WHERE discord_id = $3
	`, newExpires, PremiumWTGAllowance, discordID)
	if err != nil {
		return fmt.Errorf("failed to renew subscription: %v", err)
	}

	// Log renewal
	_, err = tx.Exec(`
		INSERT INTO credit_transactions (from_user, to_user, amount, transaction_type, description, currency_type)
		VALUES ('SYSTEM', $1, $2, 'subscription_renewal', 'Premium subscription renewed - monthly WTG allowance', 'WTG')
	`, discordID, PremiumWTGAllowance)
	if err != nil {
		log.Printf("Warning: Failed to log renewal: %v", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit renewal: %v", err)
	}

	log.Printf("🔄 Premium subscription renewed: %s (expires %s)", discordID, newExpires.Format("2006-01-02"))
	return nil
}

// CancelSubscription cancels auto-renewal but maintains benefits until expiration
func (s *SubscriptionService) CancelSubscription(discordID string) error {
	if s.db.LocalMode() {
		return fmt.Errorf("subscriptions not available in local mode")
	}

	// Set tier to 'free' but keep subscription_expires (benefits until end of period)
	_, err := s.db.DB().Exec(`
		UPDATE users 
		SET tier = 'free' 
		WHERE discord_id = $1
	`, discordID)
	if err != nil {
		return fmt.Errorf("failed to cancel subscription: %v", err)
	}

	log.Printf("❌ Subscription cancelled: %s (benefits remain until expiration)", discordID)
	return nil
}

// HasActivePremium checks if user has active premium subscription
func (s *SubscriptionService) HasActivePremium(discordID string) bool {
	if s.db.LocalMode() {
		return false
	}

	var tier string
	var expiresAt sql.NullTime

	err := s.db.DB().QueryRow(`
		SELECT tier, subscription_expires 
		FROM users 
		WHERE discord_id = $1
	`, discordID).Scan(&tier, &expiresAt)

	if err != nil {
		return false
	}

	return tier == "premium" && expiresAt.Valid && expiresAt.Time.After(time.Now())
}

// GetUserMultiplier returns the GC earning multiplier for a user
// Free: 1x, Premium: 3x (Economy Plan v4.0)
func (s *SubscriptionService) GetUserMultiplier(discordID string) int {
	if s.HasActivePremium(discordID) {
		return PremiumGCMultiplier // 3x multiplier
	}
	return 1 // Free tier
}

// ApplyMultiplierToEarnings applies premium multiplier to earnings
func (s *SubscriptionService) ApplyMultiplierToEarnings(discordID string, baseAmount int) int {
	multiplier := s.GetUserMultiplier(discordID)
	return baseAmount * multiplier
}

// GetDailyBonus returns the daily bonus amount for user's tier
func (s *SubscriptionService) GetDailyBonus(discordID string) int {
	if s.HasActivePremium(discordID) {
		return PremiumDailyBonus // 100 GC
	}
	return 50 // Free tier
}

// ExpireSubscriptions expires subscriptions that have passed their expiration date
// Should be called daily via cron job
func (s *SubscriptionService) ExpireSubscriptions() (int, error) {
	if s.db.LocalMode() {
		return 0, nil
	}

	// Find and expire subscriptions
	result, err := s.db.DB().Exec(`
		UPDATE users 
		SET tier = 'free' 
		WHERE tier = 'premium' 
		  AND subscription_expires IS NOT NULL 
		  AND subscription_expires < NOW()
	`)
	if err != nil {
		return 0, fmt.Errorf("failed to expire subscriptions: %v", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, err
	}

	if rowsAffected > 0 {
		log.Printf("⏰ Expired %d premium subscriptions", rowsAffected)
	}

	return int(rowsAffected), nil
}

// GetSubscriptionStats returns subscription statistics
func (s *SubscriptionService) GetSubscriptionStats() (map[string]int, error) {
	if s.db.LocalMode() {
		return map[string]int{}, nil
	}

	stats := make(map[string]int)

	// Active premium subscriptions
	err := s.db.DB().QueryRow(`
		SELECT COUNT(*) 
		FROM users 
		WHERE tier = 'premium' 
		  AND subscription_expires > NOW()
	`).Scan(&stats["active_premium"])
	if err != nil {
		return nil, fmt.Errorf("failed to count active premium: %v", err)
	}

	// Expired subscriptions (within last 30 days)
	err = s.db.DB().QueryRow(`
		SELECT COUNT(*) 
		FROM users 
		WHERE tier = 'free' 
		  AND subscription_expires IS NOT NULL 
		  AND subscription_expires > NOW() - INTERVAL '30 days'
		  AND subscription_expires < NOW()
	`).Scan(&stats["recently_expired"])
	if err != nil {
		return nil, fmt.Errorf("failed to count expired: %v", err)
	}

	// Total free users
	err = s.db.DB().QueryRow(`
		SELECT COUNT(*) 
		FROM users 
		WHERE tier = 'free'
	`).Scan(&stats["free_users"])
	if err != nil {
		return nil, fmt.Errorf("failed to count free users: %v", err)
	}

	// Calculate revenue (active_premium × $3.99)
	stats["monthly_revenue_cents"] = stats["active_premium"] * 399 // $3.99 in cents

	return stats, nil
}

// StartSubscriptionExpirer starts a background goroutine that expires subscriptions daily
func (s *SubscriptionService) StartSubscriptionExpirer() {
	go func() {
		ticker := time.NewTicker(24 * time.Hour)
		defer ticker.Stop()

		// Run immediately on start
		count, err := s.ExpireSubscriptions()
		if err != nil {
			log.Printf("❌ Failed to expire subscriptions: %v", err)
		} else if count > 0 {
			log.Printf("✅ Expired %d subscriptions on startup", count)
		}

		// Then run daily
		for range ticker.C {
			count, err := s.ExpireSubscriptions()
			if err != nil {
				log.Printf("❌ Failed to expire subscriptions: %v", err)
			} else if count > 0 {
				log.Printf("✅ Expired %d subscriptions", count)
			}
		}
	}()

	log.Println("⏰ Subscription expirer started (runs daily)")
}
