package services

import (
	"context"
	"database/sql"
	"fmt"
	"time"
)

// ConsentService handles GDPR-compliant ad consent tracking
type ConsentService struct {
	db *sql.DB
}

// NewConsentService creates a new consent service instance
func NewConsentService(db *sql.DB) *ConsentService {
	return &ConsentService{db: db}
}

// UserConsent represents a user's ad consent record
type UserConsent struct {
	UserID             int64
	Consented          bool
	ConsentTimestamp   *time.Time
	WithdrawnTimestamp *time.Time
	IPCountry          string
	GDPRVersion        string
	ConsentMethod      string
	CreatedAt          time.Time
	UpdatedAt          time.Time
}

// InitSchema creates the consent tracking table
func (s *ConsentService) InitSchema(ctx context.Context) error {
	if s.db == nil {
		return nil
	}

	const createTable = `
	CREATE TABLE IF NOT EXISTS user_ad_consent (
		user_id BIGINT PRIMARY KEY,
		consented BOOLEAN DEFAULT FALSE,
		consent_timestamp TIMESTAMP,
		withdrawn_timestamp TIMESTAMP,
		ip_country VARCHAR(2),
		gdpr_version VARCHAR(10) DEFAULT 'v1.0',
		consent_method VARCHAR(50) DEFAULT 'unknown',
		created_at TIMESTAMP DEFAULT NOW(),
		updated_at TIMESTAMP DEFAULT NOW()
	);

	CREATE INDEX IF NOT EXISTS idx_user_ad_consent_country ON user_ad_consent(ip_country);
	CREATE INDEX IF NOT EXISTS idx_user_ad_consent_timestamp ON user_ad_consent(consent_timestamp);
	`

	if _, err := s.db.ExecContext(ctx, createTable); err != nil {
		return fmt.Errorf("failed to create user_ad_consent table: %w", err)
	}

	return nil
}

// ConsentStats represents aggregate consent statistics
type ConsentStats struct {
	TotalUsers          int
	ConsentedUsers      int
	WithdrawnUsers      int
	EUUsers             int
	EUConsentedUsers    int
	NonEUUsers          int
	ConsentRate         float64
	EUConsentRate       float64
	WithdrawalRate      float64
	RecentConsents24h   int
	RecentWithdrawals24h int
}

// EEA countries (European Economic Area) + UK + Switzerland
// GDPR applies to these countries
var gdprCountries = map[string]bool{
	// EEA (EU + Iceland, Liechtenstein, Norway)
	"AT": true, // Austria
	"BE": true, // Belgium
	"BG": true, // Bulgaria
	"HR": true, // Croatia
	"CY": true, // Cyprus
	"CZ": true, // Czech Republic
	"DK": true, // Denmark
	"EE": true, // Estonia
	"FI": true, // Finland
	"FR": true, // France
	"DE": true, // Germany
	"GR": true, // Greece
	"HU": true, // Hungary
	"IE": true, // Ireland
	"IT": true, // Italy
	"LV": true, // Latvia
	"LT": true, // Lithuania
	"LU": true, // Luxembourg
	"MT": true, // Malta
	"NL": true, // Netherlands
	"PL": true, // Poland
	"PT": true, // Portugal
	"RO": true, // Romania
	"SK": true, // Slovakia
	"SI": true, // Slovenia
	"ES": true, // Spain
	"SE": true, // Sweden
	"IS": true, // Iceland
	"LI": true, // Liechtenstein
	"NO": true, // Norway
	// Post-Brexit UK + Switzerland
	"GB": true, // United Kingdom
	"UK": true, // United Kingdom (alternate code)
	"CH": true, // Switzerland
}

// IsGDPRCountry checks if a country code requires GDPR compliance
func IsGDPRCountry(countryCode string) bool {
	if countryCode == "" {
		// Default to requiring consent if country unknown (conservative approach)
		return true
	}
	return gdprCountries[countryCode]
}

// HasConsent checks if a user has given valid consent to view ads
// Returns (hasConsent, requiresConsent, error)
func (s *ConsentService) HasConsent(ctx context.Context, userID int64, userCountry string) (bool, bool, error) {
	requiresConsent := IsGDPRCountry(userCountry)
	
	// Non-EU users don't need consent
	if !requiresConsent {
		return true, false, nil
	}

	var consent UserConsent
	err := s.db.QueryRowContext(ctx, `
		SELECT user_id, consented, consent_timestamp, withdrawn_timestamp, ip_country, gdpr_version
		FROM user_ad_consent
		WHERE user_id = $1
	`, userID).Scan(
		&consent.UserID,
		&consent.Consented,
		&consent.ConsentTimestamp,
		&consent.WithdrawnTimestamp,
		&consent.IPCountry,
		&consent.GDPRVersion,
	)

	if err == sql.ErrNoRows {
		// No consent record = no consent given
		return false, true, nil
	}
	if err != nil {
		return false, true, fmt.Errorf("failed to query consent: %w", err)
	}

	// Check if consent was withdrawn
	if consent.WithdrawnTimestamp != nil {
		return false, true, nil
	}

	// Valid consent exists
	return consent.Consented, true, nil
}

// RecordConsent records a user's consent decision
func (s *ConsentService) RecordConsent(ctx context.Context, userID int64, consented bool, userCountry, method string) error {
	now := time.Now()
	
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO user_ad_consent (
			user_id, consented, consent_timestamp, ip_country, gdpr_version, consent_method, created_at, updated_at
		) VALUES (
			$1, $2, $3, $4, 'v1.0', $5, $6, $6
		)
		ON CONFLICT (user_id) DO UPDATE SET
			consented = EXCLUDED.consented,
			consent_timestamp = EXCLUDED.consent_timestamp,
			withdrawn_timestamp = NULL,
			ip_country = EXCLUDED.ip_country,
			consent_method = EXCLUDED.consent_method,
			updated_at = EXCLUDED.updated_at
	`, userID, consented, now, userCountry, method, now)

	if err != nil {
		return fmt.Errorf("failed to record consent: %w", err)
	}

	return nil
}

// WithdrawConsent withdraws a user's consent (GDPR right to withdraw)
func (s *ConsentService) WithdrawConsent(ctx context.Context, userID int64) error {
	now := time.Now()
	
	result, err := s.db.ExecContext(ctx, `
		UPDATE user_ad_consent
		SET consented = FALSE,
			consent_timestamp = NULL,
			withdrawn_timestamp = $1,
			updated_at = $2
		WHERE user_id = $3 AND consented = TRUE
	`, now, now, userID)

	if err != nil {
		return fmt.Errorf("failed to withdraw consent: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("no active consent to withdraw")
	}

	return nil
}

// GetConsentStatus retrieves a user's current consent status
func (s *ConsentService) GetConsentStatus(ctx context.Context, userID int64) (*UserConsent, error) {
	var consent UserConsent
	err := s.db.QueryRowContext(ctx, `
		SELECT user_id, consented, consent_timestamp, withdrawn_timestamp, 
		       ip_country, gdpr_version, consent_method, created_at, updated_at
		FROM user_ad_consent
		WHERE user_id = $1
	`, userID).Scan(
		&consent.UserID,
		&consent.Consented,
		&consent.ConsentTimestamp,
		&consent.WithdrawnTimestamp,
		&consent.IPCountry,
		&consent.GDPRVersion,
		&consent.ConsentMethod,
		&consent.CreatedAt,
		&consent.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		return nil, nil // No consent record
	}
	if err != nil {
		return nil, fmt.Errorf("failed to get consent status: %w", err)
	}

	return &consent, nil
}

// GetConsentStats retrieves aggregate consent statistics for admin reporting
func (s *ConsentService) GetConsentStats(ctx context.Context) (*ConsentStats, error) {
	stats := &ConsentStats{}

	// Total users with consent records
	err := s.db.QueryRowContext(ctx, `
		SELECT 
			COUNT(*) as total_users,
			SUM(CASE WHEN consented = TRUE AND withdrawn_timestamp IS NULL THEN 1 ELSE 0 END) as consented_users,
			SUM(CASE WHEN withdrawn_timestamp IS NOT NULL THEN 1 ELSE 0 END) as withdrawn_users
		FROM user_ad_consent
	`).Scan(&stats.TotalUsers, &stats.ConsentedUsers, &stats.WithdrawnUsers)
	if err != nil {
		return nil, fmt.Errorf("failed to get basic stats: %w", err)
	}

	// EU-specific stats
	err = s.db.QueryRowContext(ctx, `
		SELECT 
			COUNT(*) as eu_users,
			SUM(CASE WHEN consented = TRUE AND withdrawn_timestamp IS NULL THEN 1 ELSE 0 END) as eu_consented
		FROM user_ad_consent
		WHERE ip_country IN (
			'AT','BE','BG','HR','CY','CZ','DK','EE','FI','FR','DE','GR','HU','IE','IT',
			'LV','LT','LU','MT','NL','PL','PT','RO','SK','SI','ES','SE','IS','LI','NO',
			'GB','UK','CH'
		)
	`).Scan(&stats.EUUsers, &stats.EUConsentedUsers)
	if err != nil {
		return nil, fmt.Errorf("failed to get EU stats: %w", err)
	}

	stats.NonEUUsers = stats.TotalUsers - stats.EUUsers

	// Calculate rates
	if stats.TotalUsers > 0 {
		stats.ConsentRate = float64(stats.ConsentedUsers) / float64(stats.TotalUsers) * 100
		stats.WithdrawalRate = float64(stats.WithdrawnUsers) / float64(stats.TotalUsers) * 100
	}
	if stats.EUUsers > 0 {
		stats.EUConsentRate = float64(stats.EUConsentedUsers) / float64(stats.EUUsers) * 100
	}

	// Recent activity (last 24 hours)
	err = s.db.QueryRowContext(ctx, `
		SELECT 
			SUM(CASE WHEN consent_timestamp > NOW() - INTERVAL '24 hours' THEN 1 ELSE 0 END) as recent_consents,
			SUM(CASE WHEN withdrawn_timestamp > NOW() - INTERVAL '24 hours' THEN 1 ELSE 0 END) as recent_withdrawals
		FROM user_ad_consent
	`).Scan(&stats.RecentConsents24h, &stats.RecentWithdrawals24h)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent activity: %w", err)
	}

	return stats, nil
}

// EnsureUserConsentRecord creates a consent record if it doesn't exist
// Used for tracking users who need to give consent
func (s *ConsentService) EnsureUserConsentRecord(ctx context.Context, userID int64, userCountry string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO user_ad_consent (user_id, consented, ip_country, gdpr_version, created_at, updated_at)
		VALUES ($1, FALSE, $2, 'v1.0', NOW(), NOW())
		ON CONFLICT (user_id) DO NOTHING
	`, userID, userCountry)

	if err != nil {
		return fmt.Errorf("failed to ensure consent record: %w", err)
	}

	return nil
}

// GetConsentPromptText returns GDPR-compliant consent prompt text
func GetConsentPromptText() string {
	return `**WeTheGamers Ad Consent**

To earn Game Credits by watching ads, we need your consent to:
• Display personalized advertisements from ayeT-Studios
• Process your Discord user ID for reward delivery
• Track ad viewing to prevent fraud

**Your Rights:**
• You can withdraw consent at any time using /consent-withdraw
• Withdrawing consent will disable ad earnings
• Your data is never sold to third parties
• View our privacy policy: https://wethegamers.com/privacy

**Do you consent to viewing ads and the associated data processing?**`
}

// GetPrivacyPolicyURL returns the URL to the privacy policy
func GetPrivacyPolicyURL() string {
	return "https://wethegamers.com/privacy"
}
