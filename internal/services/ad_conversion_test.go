package services

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"testing"
)

// TestVerifyAyetSignature tests HMAC-SHA1 signature verification
func TestVerifyAyetSignature(t *testing.T) {
	tests := []struct {
		name      string
		apiKey    string
		params    AyetCallbackParams
		wantValid bool
	}{
		{
			name:   "Valid signature",
			apiKey: "test_api_key_12345",
			params: AyetCallbackParams{
				ExternalIdentifier: "123456789",
				UID:                "",
				Currency:           "coins",
				Amount:             100,
				ConversionID:       "conv_abc123",
				Custom1:            "offerwall",
				Custom2:            "",
				Custom3:            "",
				Custom4:            "",
				Signature:          "", // Will be computed
			},
			wantValid: true,
		},
		{
			name:   "Invalid signature",
			apiKey: "test_api_key_12345",
			params: AyetCallbackParams{
				ExternalIdentifier: "123456789",
				UID:                "",
				Currency:           "coins",
				Amount:             100,
				ConversionID:       "conv_abc123",
				Custom1:            "offerwall",
				Signature:          "invalid_signature",
			},
			wantValid: false,
		},
		{
			name:   "Wrong API key",
			apiKey: "wrong_key",
			params: AyetCallbackParams{
				ExternalIdentifier: "123456789",
				Currency:           "coins",
				Amount:             100,
				ConversionID:       "conv_abc123",
				Signature:          "", // Will be computed with different key
			},
			wantValid: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Compute expected signature
			if tt.wantValid {
				payload := fmt.Sprintf("%s|%s|%s|%d|%s|%s|%s|%s|%s",
					tt.params.ExternalIdentifier,
					tt.params.UID,
					tt.params.Currency,
					tt.params.Amount,
					tt.params.ConversionID,
					tt.params.Custom1,
					tt.params.Custom2,
					tt.params.Custom3,
					tt.params.Custom4,
				)
				
				mac := hmac.New(sha1.New, []byte(tt.apiKey))
				mac.Write([]byte(payload))
				tt.params.Signature = hex.EncodeToString(mac.Sum(nil))
			}

		// Create service
		dbService := &DatabaseService{localMode: true}
		consentService := NewConsentService(nil) // nil DB for local mode
		service := NewAdConversionService(dbService, consentService, tt.apiKey, "")

			// Verify signature
			err := service.verifyAyetSignature(tt.params)
			
			if tt.wantValid && err != nil {
				t.Errorf("Expected valid signature, got error: %v", err)
			}
			if !tt.wantValid && err == nil {
				t.Errorf("Expected invalid signature, got no error")
			}
		})
	}
}

// TestDetectFraud tests fraud detection logic
func TestDetectFraud(t *testing.T) {
	ctx := context.Background()
	
	// Create local mode service (no DB)
	dbService := &DatabaseService{localMode: true}
	consentService := NewConsentService(nil)
	service := NewAdConversionService(dbService, consentService, "", "")

	// Test in local mode (should return no fraud)
	reason, isFraud := service.detectFraud(ctx, "test_user", AyetCallbackParams{
		IPAddress: "1.2.3.4",
	})

	if isFraud {
		t.Errorf("Expected no fraud in local mode, got fraud: %s", reason)
	}
}

// TestCalculateReward tests currency conversion (from RewardAlgorithm, not AdConversionService)
func TestCalculateReward(t *testing.T) {
	tests := []struct {
		currency string
		amount   int
		expected int
	}{
		{"coins", 100, 100},
		{"gold", 50, 50},
		{"points", 200, 200},
		{"gems", 10, 20}, // 2x multiplier
		{"cash", 1, 100}, // 100x multiplier
		{"unknown", 75, 75}, // Default 1:1
	}

	dbService := &DatabaseService{localMode: true}
	algo := NewRewardAlgorithm(dbService)

	for _, tt := range tests {
		t.Run(tt.currency, func(t *testing.T) {
			result := algo.convertCurrency(tt.currency, tt.amount)
			if result != tt.expected {
				t.Errorf("convertCurrency(%s, %d) = %d, want %d", tt.currency, tt.amount, result, tt.expected)
			}
		})
	}
}

// TestInferType tests ad type inference from custom data
func TestInferType(t *testing.T) {
	tests := []struct {
		custom1  string
		expected string
	}{
		{"offerwall_123", "offerwall"},
		{"survey_abc", "surveywall"},
		{"video_xyz", "video"},
		{"unknown", "offerwall"}, // default
		{"", "offerwall"},        // empty
	}

	for _, tt := range tests {
		t.Run(tt.custom1, func(t *testing.T) {
			result := inferType(tt.custom1)
			if result != tt.expected {
				t.Errorf("inferType(%s) = %s, want %s", tt.custom1, result, tt.expected)
			}
		})
	}
}

// BenchmarkVerifySignature benchmarks signature verification performance
func BenchmarkVerifySignature(b *testing.B) {
	params := AyetCallbackParams{
		ExternalIdentifier: "123456789",
		UID:                "",
		Currency:           "coins",
		Amount:             100,
		ConversionID:       "conv_abc123",
		Custom1:            "offerwall",
	}

	apiKey := "test_api_key_12345"
	
	// Compute signature
	payload := fmt.Sprintf("%s|%s|%s|%d|%s|%s||||",
		params.ExternalIdentifier,
		params.UID,
		params.Currency,
		params.Amount,
		params.ConversionID,
		params.Custom1,
	)
	
	mac := hmac.New(sha1.New, []byte(apiKey))
	mac.Write([]byte(payload))
	params.Signature = hex.EncodeToString(mac.Sum(nil))

	dbService := &DatabaseService{localMode: true}
	consentService := NewConsentService(nil)
	service := NewAdConversionService(dbService, consentService, apiKey, "")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		service.verifyAyetSignature(params)
	}
}

// TestConversionIdempotency tests duplicate conversion detection
func TestConversionIdempotency(t *testing.T) {
	ctx := context.Background()
	
	dbService := &DatabaseService{
		localMode:        true,
		localConversions: make(map[string]bool),
	}
	consentService := NewConsentService(nil)
	service := NewAdConversionService(dbService, consentService, "", "")

	conversionID := "test_conv_123"

	// First check - should not exist
	exists, err := service.conversionExists(ctx, conversionID)
	if err != nil {
		t.Fatalf("conversionExists error: %v", err)
	}
	if exists {
		t.Error("Expected conversion to not exist initially")
	}

	// Mark as processed
	dbService.localConversions[conversionID] = true

	// Second check - should exist
	exists, err = service.conversionExists(ctx, conversionID)
	if err != nil {
		t.Fatalf("conversionExists error: %v", err)
	}
	if !exists {
		t.Error("Expected conversion to exist after marking")
	}
}

// TestRewardFloorAndCeiling tests reward limits
func TestRewardFloorAndCeiling(t *testing.T) {
	ctx := context.Background()
	dbService := &DatabaseService{localMode: true}
	algo := NewRewardAlgorithm(dbService)

	tests := []struct {
		name     string
		baseReward int
		multiplier float64
		wantMin    int
		wantMax    int
	}{
		{"Below floor", 0, 1.0, 1, 1},
		{"At floor", 1, 1.0, 1, 1},
		{"Normal range", 1000, 1.0, 1000, 1200}, // Type multiplier can increase it
		{"Above ceiling", 15000, 1.0, 10000, 10000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rewardCtx := RewardContext{
				ProviderCurrency: "coins",
				ProviderAmount:   tt.baseReward,
				ConversionType:   "offerwall",
				UserTier:         "free",
				IsPremium:        false,
			}

			calc, err := algo.CalculateReward(ctx, rewardCtx)
			if err != nil {
				t.Fatalf("CalculateReward error: %v", err)
			}

			if calc.FinalReward < tt.wantMin {
				t.Errorf("FinalReward %d is below minimum %d", calc.FinalReward, tt.wantMin)
			}
			if calc.FinalReward > tt.wantMax {
				t.Errorf("FinalReward %d is above maximum %d", calc.FinalReward, tt.wantMax)
			}
		})
	}
}

// TestPremiumMultipliers tests tier-based multipliers
func TestPremiumMultipliers(t *testing.T) {
	dbService := &DatabaseService{localMode: true}
	algo := NewRewardAlgorithm(dbService)

	tests := []struct {
		tier       string
		isPremium  bool
		wantMult   float64
	}{
		{"free", false, 1.0},
		{"verified", false, 1.1},
		{"premium", true, 1.5},
		{"premium_plus", true, 2.0},
	}

	for _, tt := range tests {
		t.Run(tt.tier, func(t *testing.T) {
			mult := algo.getTierMultiplier(tt.tier, tt.isPremium)
			if mult != tt.wantMult {
				t.Errorf("getTierMultiplier(%s, %v) = %.1f, want %.1f", tt.tier, tt.isPremium, mult, tt.wantMult)
			}
		})
	}
}

// TestAdTypeMultipliers tests ad type modifiers
func TestAdTypeMultipliers(t *testing.T) {
	dbService := &DatabaseService{localMode: true}
	algo := NewRewardAlgorithm(dbService)

	tests := []struct {
		adType   string
		expected float64
	}{
		{"offerwall", 1.0},
		{"surveywall", 1.2},
		{"video", 0.8},
		{"unknown", 1.0},
	}

	for _, tt := range tests {
		t.Run(tt.adType, func(t *testing.T) {
			mult := algo.getTypeMultiplier(tt.adType)
			if mult != tt.expected {
				t.Errorf("getTypeMultiplier(%s) = %.1f, want %.1f", tt.adType, mult, tt.expected)
			}
		})
	}
}

// TestEngagementBonus tests engagement bonus calculation
func TestEngagementBonus(t *testing.T) {
	dbService := &DatabaseService{localMode: true}
	algo := NewRewardAlgorithm(dbService)

	tests := []struct {
		name                string
		recentConversions   int
		totalConversions    int
		lastConversionHours int
		daysSinceJoined     int
		expectedBonus       int
	}{
		{"New user, no activity", 0, 0, 0, 1, 0},
		{"Return user", 0, 5, 25, 1, 50},
		{"Active streak", 6, 10, 1, 1, 25},
		{"Loyal user", 0, 0, 0, 35, 10},
		{"All bonuses", 6, 10, 25, 35, 85}, // 50 + 25 + 10
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := RewardContext{
				RecentConversions:   tt.recentConversions,
				TotalConversions:    tt.totalConversions,
				LastConversionHours: tt.lastConversionHours,
				DaysSinceJoined:     tt.daysSinceJoined,
			}

			bonus := algo.calculateEngagementBonus(ctx)
			if bonus != tt.expectedBonus {
				t.Errorf("Expected engagement bonus %d, got %d", tt.expectedBonus, bonus)
			}
		})
	}
}

// TestNewUserBonus tests new user acquisition bonus
func TestNewUserBonus(t *testing.T) {
	dbService := &DatabaseService{localMode: true}
	algo := NewRewardAlgorithm(dbService)

	tests := []struct {
		daysSinceJoined int
		providerAmount  int
		expectedBonus   int
	}{
		{1, 100, 20},  // Within 7 days: 20% of 100
		{7, 50, 10},   // Day 7: 20% of 50
		{8, 100, 0},   // After 7 days: no bonus
		{30, 100, 0},  // Long-time user: no bonus
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			ctx := RewardContext{
				DaysSinceJoined: tt.daysSinceJoined,
				ProviderAmount:  tt.providerAmount,
			}

			bonus := algo.calculateNewUserBonus(ctx)
			if bonus != tt.expectedBonus {
				t.Errorf("Days %d, Amount %d: expected bonus %d, got %d",
					tt.daysSinceJoined, tt.providerAmount, tt.expectedBonus, bonus)
			}
		})
	}
}
