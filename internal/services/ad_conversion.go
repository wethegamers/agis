package services

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"
)

var (
	ErrInvalidSignature     = errors.New("invalid signature")
	ErrDuplicateConversion  = errors.New("duplicate conversion")
	ErrConsentRequired      = errors.New("user consent required")
	ErrFraudDetected        = errors.New("fraud detected")
	ErrInvalidAmount        = errors.New("invalid amount")
)

// AdConversion represents a completed ad conversion
type AdConversion struct {
	ID            int64
	UserID        int64
	ConversionID  string
	Provider      string // "ayet"
	Type          string // "offerwall", "surveywall", "video"
	Amount        int    // Game Credits earned
	Currency      string // Provider's currency (e.g. "coins")
	ProviderValue int    // Provider's currency amount
	Multiplier    float64 // Premium multiplier applied
	IPAddress     string
	UserAgent     string
	CustomData    map[string]string // custom_1..custom_4
	Signature     string
	CreatedAt     time.Time
	ProcessedAt   *time.Time
	Status        string // "pending", "completed", "fraud"
	FraudReason   string
}

// AdConversionService handles ad conversion callbacks and rewards
type AdConversionService struct {
	db             *DatabaseService
	consentService *ConsentService
	apiKey         string
	callbackToken  string
	localMode      bool
}

// NewAdConversionService creates a new ad conversion service
func NewAdConversionService(db *DatabaseService, consentService *ConsentService, apiKey, callbackToken string) *AdConversionService {
	return &AdConversionService{
		db:             db,
		consentService: consentService,
		apiKey:         apiKey,
		callbackToken:  callbackToken,
		localMode:      db.LocalMode(),
	}
}

// InitSchema creates the ad_conversions table
func (a *AdConversionService) InitSchema(ctx context.Context) error {
	if a.localMode {
		log.Println("📄 Ad conversions schema skipped (local mode)")
		return nil
	}

	createTable := `
	CREATE TABLE IF NOT EXISTS ad_conversions (
		id SERIAL PRIMARY KEY,
		discord_id VARCHAR(32) NOT NULL,
		conversion_id VARCHAR(255) NOT NULL UNIQUE,
		provider VARCHAR(50) NOT NULL,
		type VARCHAR(50) NOT NULL,
		amount INTEGER NOT NULL,
		currency VARCHAR(20),
		provider_value INTEGER,
		multiplier DECIMAL(3,2) DEFAULT 1.0,
		ip_address VARCHAR(45),
		user_agent TEXT,
		custom_data JSONB,
		signature VARCHAR(255),
		created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
		processed_at TIMESTAMP,
		status VARCHAR(20) DEFAULT 'pending',
		fraud_reason TEXT,
		FOREIGN KEY (discord_id) REFERENCES users(discord_id)
	);
	
	CREATE INDEX IF NOT EXISTS idx_ad_conversions_discord_id ON ad_conversions(discord_id);
	CREATE INDEX IF NOT EXISTS idx_ad_conversions_conversion_id ON ad_conversions(conversion_id);
	CREATE INDEX IF NOT EXISTS idx_ad_conversions_created_at ON ad_conversions(created_at);
	CREATE INDEX IF NOT EXISTS idx_ad_conversions_status ON ad_conversions(status);
	`

	_, err := a.db.DB().ExecContext(ctx, createTable)
	if err != nil {
		return fmt.Errorf("failed to create ad_conversions table: %w", err)
	}

	log.Println("✅ Ad conversions schema initialized")
	return nil
}

// AyetCallbackParams represents ayeT-Studios S2S callback parameters
type AyetCallbackParams struct {
	ExternalIdentifier string // Discord user ID
	UID                string // Alternative user ID field
	Currency           string // "coins", "gold", etc.
	Amount             int    // Amount in provider's currency
	ConversionID       string // Unique conversion ID for idempotency
	Signature          string // HMAC-SHA1 signature
	Custom1            string // Custom parameters
	Custom2            string
	Custom3            string
	Custom4            string
	IPAddress          string
	UserAgent          string
}

// ProcessAyetCallback handles ayeT-Studios S2S conversion callback
func (a *AdConversionService) ProcessAyetCallback(ctx context.Context, params AyetCallbackParams) error {
	// 1. Extract user ID
	userID := params.ExternalIdentifier
	if userID == "" {
		userID = params.UID
	}
	if userID == "" {
		return fmt.Errorf("missing user identifier")
	}

	// 2. Verify signature (HMAC-SHA1)
	if err := a.verifyAyetSignature(params); err != nil {
		log.Printf("⚠️ Invalid signature for conversion %s: %v", params.ConversionID, err)
		return ErrInvalidSignature
	}

	// 3. Check for duplicate conversion (idempotency)
	if exists, err := a.conversionExists(ctx, params.ConversionID); err != nil {
		return fmt.Errorf("failed to check conversion existence: %w", err)
	} else if exists {
		log.Printf("ℹ️ Duplicate conversion ignored: %s", params.ConversionID)
		return ErrDuplicateConversion
	}

	// 4. Check user consent (GDPR)
	userIDInt := int64(0)
	fmt.Sscanf(userID, "%d", &userIDInt)
	
	consent, err := a.consentService.GetConsentStatus(ctx, userIDInt)
	if err != nil {
		return fmt.Errorf("failed to check consent: %w", err)
	}
	if consent == nil || !consent.Consented || consent.WithdrawnTimestamp != nil {
		log.Printf("⚠️ User %s has not consented or withdrew consent", userID)
		return ErrConsentRequired
	}

	// 5. Validate amount
	if params.Amount <= 0 {
		return ErrInvalidAmount
	}

	// 6. Fraud detection
	if fraudReason, isFraud := a.detectFraud(ctx, userID, params); isFraud {
		log.Printf("🚨 Fraud detected for user %s: %s", userID, fraudReason)
		// Store but don't credit
		return a.recordConversion(ctx, userID, params, 0, 1.0, "fraud", fraudReason)
	}

	// 7. Calculate reward (convert provider currency to Game Credits)
	baseReward := a.calculateReward(params.Currency, params.Amount)

	// 8. Apply premium multiplier if applicable
	multiplier := 1.0
	// TODO: Check subscription status and apply multiplier (1.5x-2x)
	// This will be implemented in the Ad-Watch Multipliers task

	finalReward := int(float64(baseReward) * multiplier)

	// 9. Record conversion
	if err := a.recordConversion(ctx, userID, params, finalReward, multiplier, "completed", ""); err != nil {
		return fmt.Errorf("failed to record conversion: %w", err)
	}

	// 10. Credit user
	if err := a.creditUser(ctx, userID, finalReward); err != nil {
		return fmt.Errorf("failed to credit user: %w", err)
	}

	log.Printf("✅ Ad conversion processed: user=%s, conversion=%s, reward=%d GC", userID, params.ConversionID, finalReward)
	return nil
}

// verifyAyetSignature verifies HMAC-SHA1 signature
func (a *AdConversionService) verifyAyetSignature(params AyetCallbackParams) error {
	if a.apiKey == "" {
		log.Println("⚠️ AYET_API_KEY not configured, skipping signature verification")
		return nil // Skip verification in dev
	}

	// Build payload: externalIdentifier|uid|currency|amount|conversionId|custom1|custom2|custom3|custom4
	payload := fmt.Sprintf("%s|%s|%s|%d|%s|%s|%s|%s|%s",
		params.ExternalIdentifier,
		params.UID,
		params.Currency,
		params.Amount,
		params.ConversionID,
		params.Custom1,
		params.Custom2,
		params.Custom3,
		params.Custom4,
	)

	// Compute HMAC-SHA1
	mac := hmac.New(sha1.New, []byte(a.apiKey))
	mac.Write([]byte(payload))
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	if !hmac.Equal([]byte(strings.ToLower(params.Signature)), []byte(strings.ToLower(expectedSignature))) {
		return ErrInvalidSignature
	}

	return nil
}

// conversionExists checks if a conversion ID has already been processed
func (a *AdConversionService) conversionExists(ctx context.Context, conversionID string) (bool, error) {
	if a.localMode {
		a.db.localMutex.RLock()
		defer a.db.localMutex.RUnlock()
		exists := a.db.localConversions[conversionID]
		return exists, nil
	}

	var exists bool
	err := a.db.DB().QueryRowContext(ctx,
		"SELECT EXISTS(SELECT 1 FROM ad_conversions WHERE conversion_id = $1)",
		conversionID,
	).Scan(&exists)
	return exists, err
}

// detectFraud performs basic fraud detection
func (a *AdConversionService) detectFraud(ctx context.Context, discordID string, params AyetCallbackParams) (string, bool) {
	if a.localMode {
		return "", false // Skip fraud detection in local mode
	}

	// 1. Check conversion velocity (max 10 conversions per hour)
	var recentCount int
	err := a.db.DB().QueryRowContext(ctx,
		`SELECT COUNT(*) FROM ad_conversions 
		 WHERE discord_id = $1 AND created_at > NOW() - INTERVAL '1 hour'`,
		discordID,
	).Scan(&recentCount)
	if err == nil && recentCount >= 10 {
		return "excessive_velocity", true
	}

	// 2. Check for suspicious IP changes (same user, different IPs in short time)
	if params.IPAddress != "" {
		var differentIPs int
		err := a.db.DB().QueryRowContext(ctx,
			`SELECT COUNT(DISTINCT ip_address) FROM ad_conversions 
			 WHERE discord_id = $1 AND created_at > NOW() - INTERVAL '10 minutes' AND ip_address != $2`,
			discordID, params.IPAddress,
		).Scan(&differentIPs)
		if err == nil && differentIPs >= 3 {
			return "ip_hopping", true
		}
	}

	// 3. Check for abnormally high rewards in short time
	var recentRewards int
	err = a.db.DB().QueryRowContext(ctx,
		`SELECT COALESCE(SUM(amount), 0) FROM ad_conversions 
		 WHERE discord_id = $1 AND created_at > NOW() - INTERVAL '1 hour' AND status = 'completed'`,
		discordID,
	).Scan(&recentRewards)
	if err == nil && recentRewards > 10000 {
		return "excessive_earnings", true
	}

	return "", false
}

// calculateReward converts provider currency to Game Credits
func (a *AdConversionService) calculateReward(currency string, amount int) int {
	// Base conversion rates (these should be configurable)
	// For now, simple 1:1 mapping
	switch strings.ToLower(currency) {
	case "coins", "gold", "points":
		return amount
	default:
		return amount
	}
}

// recordConversion stores the conversion in the database
func (a *AdConversionService) recordConversion(ctx context.Context, discordID string, params AyetCallbackParams, reward int, multiplier float64, status, fraudReason string) error {
	if a.localMode {
		a.db.localMutex.Lock()
		defer a.db.localMutex.Unlock()
		a.db.localConversions[params.ConversionID] = true
		return nil
	}

	now := time.Now()
	_, err := a.db.DB().ExecContext(ctx,
		`INSERT INTO ad_conversions 
		 (discord_id, conversion_id, provider, type, amount, currency, provider_value, multiplier, 
		  ip_address, user_agent, signature, created_at, processed_at, status, fraud_reason)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15)`,
		discordID, params.ConversionID, "ayet", inferType(params.Custom1),
		reward, params.Currency, params.Amount, multiplier,
		params.IPAddress, params.UserAgent, params.Signature,
		now, &now, status, fraudReason,
	)
	return err
}

// inferType attempts to determine ad type from custom data
func inferType(custom1 string) string {
	custom1Lower := strings.ToLower(custom1)
	if strings.Contains(custom1Lower, "offer") {
		return "offerwall"
	} else if strings.Contains(custom1Lower, "survey") {
		return "surveywall"
	} else if strings.Contains(custom1Lower, "video") {
		return "video"
	}
	return "offerwall" // default
}

// creditUser adds Game Credits to the user's account
func (a *AdConversionService) creditUser(ctx context.Context, discordID string, amount int) error {
	if a.localMode {
		a.db.localMutex.Lock()
		defer a.db.localMutex.Unlock()
		
		user, exists := a.db.localUsers[discordID]
		if !exists {
			user = &User{
				DiscordID: discordID,
				Credits:   0,
				Tier:      "free",
				JoinDate:  time.Now(),
			}
			a.db.localUsers[discordID] = user
		}
		user.Credits += amount
		return nil
	}

	_, err := a.db.DB().ExecContext(ctx,
		`INSERT INTO users (discord_id, credits) VALUES ($1, $2)
		 ON CONFLICT (discord_id) DO UPDATE SET credits = users.credits + $2`,
		discordID, amount,
	)
	return err
}

// GetUserConversions retrieves a user's conversion history
func (a *AdConversionService) GetUserConversions(ctx context.Context, userID int64, limit int) ([]AdConversion, error) {
	if a.localMode {
		return []AdConversion{}, nil
	}

	rows, err := a.db.DB().QueryContext(ctx,
		`SELECT id, user_id, conversion_id, provider, type, amount, currency, provider_value, 
		        multiplier, created_at, status, fraud_reason
		 FROM ad_conversions WHERE user_id = $1 ORDER BY created_at DESC LIMIT $2`,
		userID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var conversions []AdConversion
	for rows.Next() {
		var c AdConversion
		err := rows.Scan(&c.ID, &c.UserID, &c.ConversionID, &c.Provider, &c.Type,
			&c.Amount, &c.Currency, &c.ProviderValue, &c.Multiplier,
			&c.CreatedAt, &c.Status, &c.FraudReason)
		if err != nil {
			return nil, err
		}
		conversions = append(conversions, c)
	}

	return conversions, nil
}

// GetConversionStats retrieves aggregate statistics
func (a *AdConversionService) GetConversionStats(ctx context.Context) (*ConversionStats, error) {
	if a.localMode {
		return &ConversionStats{}, nil
	}

	stats := &ConversionStats{}

	// Total conversions and rewards
	err := a.db.DB().QueryRowContext(ctx,
		`SELECT 
			COUNT(*), 
			COALESCE(SUM(amount), 0),
			COUNT(DISTINCT user_id)
		 FROM ad_conversions WHERE status = 'completed'`,
	).Scan(&stats.TotalConversions, &stats.TotalRewards, &stats.UniqueUsers)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Fraud attempts
	err = a.db.DB().QueryRowContext(ctx,
		`SELECT COUNT(*) FROM ad_conversions WHERE status = 'fraud'`,
	).Scan(&stats.FraudAttempts)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// Recent activity (24h)
	err = a.db.DB().QueryRowContext(ctx,
		`SELECT COUNT(*), COALESCE(SUM(amount), 0)
		 FROM ad_conversions 
		 WHERE status = 'completed' AND created_at > NOW() - INTERVAL '24 hours'`,
	).Scan(&stats.Conversions24h, &stats.Rewards24h)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	// By type
	rows, err := a.db.DB().QueryContext(ctx,
		`SELECT type, COUNT(*), COALESCE(SUM(amount), 0)
		 FROM ad_conversions WHERE status = 'completed'
		 GROUP BY type`,
	)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}
	if rows != nil {
		defer rows.Close()
		stats.ByType = make(map[string]TypeStats)
		for rows.Next() {
			var adType string
			var count, rewards int
			if err := rows.Scan(&adType, &count, &rewards); err != nil {
				continue
			}
			stats.ByType[adType] = TypeStats{
				Count:   count,
				Rewards: rewards,
			}
		}
	}

	return stats, nil
}

// ConversionStats holds aggregate conversion statistics
type ConversionStats struct {
	TotalConversions int
	TotalRewards     int
	UniqueUsers      int
	FraudAttempts    int
	Conversions24h   int
	Rewards24h       int
	ByType           map[string]TypeStats
}

// TypeStats holds statistics for a specific ad type
type TypeStats struct {
	Count   int
	Rewards int
}
