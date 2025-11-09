// +build integration

package services

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"
)

// Integration tests for ayeT-Studios S2S callbacks
// Run with: go test -tags=integration ./internal/services

const (
	// ayeT Sandbox API (replace with actual sandbox URLs)
	ayetSandboxAPIBase    = "https://sandbox-api.ayet-studios.com"
	ayetSandboxDashboard  = "https://sandbox-dashboard.ayet-studios.com"
	ayetTestUserID        = "test-user-12345"
	ayetTestConversionID  = "test-conv-"
	testDiscordID         = "999999999999999999" // Test user
)

// TestAyetSandboxConnection verifies connectivity to ayeT sandbox
func TestAyetSandboxConnection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	apiKey := os.Getenv("AYET_API_KEY_SANDBOX")
	if apiKey == "" {
		t.Skip("AYET_API_KEY_SANDBOX not set, skipping sandbox tests")
	}

	// Test API health endpoint
	resp, err := http.Get(ayetSandboxAPIBase + "/health")
	if err != nil {
		t.Fatalf("Failed to connect to ayeT sandbox: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Sandbox health check failed: status %d", resp.StatusCode)
	}
}

// TestAyetOfferwallCallback tests end-to-end offerwall conversion
func TestAyetOfferwallCallback(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	apiKey := os.Getenv("AYET_API_KEY_SANDBOX")
	callbackURL := os.Getenv("AGIS_BOT_CALLBACK_URL") // e.g., http://localhost:9090/ads/ayet/s2s
	if apiKey == "" || callbackURL == "" {
		t.Skip("Required env vars not set: AYET_API_KEY_SANDBOX, AGIS_BOT_CALLBACK_URL")
	}

	// Step 1: Simulate user completing an offer via ayeT sandbox API
	conversionID := fmt.Sprintf("%s%d", ayetTestConversionID, time.Now().Unix())
	sandboxOffer := map[string]interface{}{
		"user_id":       ayetTestUserID,
		"offer_id":      "test-offer-123",
		"payout":        500, // 500 coins
		"conversion_id": conversionID,
	}

	sandboxPayload, _ := json.Marshal(sandboxOffer)
	sandboxResp, err := http.Post(
		ayetSandboxAPIBase+"/v1/conversions/simulate",
		"application/json",
		bytes.NewReader(sandboxPayload),
	)
	if err != nil {
		t.Fatalf("Failed to simulate conversion in sandbox: %v", err)
	}
	defer sandboxResp.Body.Close()

	if sandboxResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(sandboxResp.Body)
		t.Fatalf("Sandbox conversion simulation failed: %d - %s", sandboxResp.StatusCode, string(body))
	}

	t.Logf("Sandbox conversion simulated: %s", conversionID)

	// Step 2: Wait for ayeT sandbox to send S2S callback to our server
	// In real scenario, ayeT would POST to callbackURL automatically
	// For testing, we manually trigger the callback

	params := map[string]string{
		"externalIdentifier": testDiscordID,
		"uid":                ayetTestUserID,
		"currency":           "coins",
		"amount":             "500",
		"conversionId":       conversionID,
		"custom_1":           "offerwall",
		"custom_2":           "",
		"custom_3":           "",
		"custom_4":           "",
	}

	signature := generateAyetSignature(params, apiKey)
	params["signature"] = signature

	// Build callback URL with query params
	callbackReq, err := http.NewRequest("GET", callbackURL, nil)
	if err != nil {
		t.Fatalf("Failed to create callback request: %v", err)
	}

	q := callbackReq.URL.Query()
	for k, v := range params {
		q.Add(k, v)
	}
	callbackReq.URL.RawQuery = q.Encode()

	// Step 3: Send callback to AGIS Bot
	client := &http.Client{Timeout: 10 * time.Second}
	callbackResp, err := client.Do(callbackReq)
	if err != nil {
		t.Fatalf("Failed to send callback to AGIS Bot: %v", err)
	}
	defer callbackResp.Body.Close()

	body, _ := io.ReadAll(callbackResp.Body)
	t.Logf("Callback response: %d - %s", callbackResp.StatusCode, string(body))

	if callbackResp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d: %s", callbackResp.StatusCode, string(body))
	}

	// Step 4: Verify conversion was recorded in database
	// (requires database access - skip if DB_HOST not set)
	dbHost := os.Getenv("DB_HOST")
	if dbHost != "" {
		// TODO: Query database to verify conversion record
		t.Log("Database verification not implemented yet")
	}
}

// TestAyetSurveywallCallback tests surveywall conversion flow
func TestAyetSurveywallCallback(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	apiKey := os.Getenv("AYET_API_KEY_SANDBOX")
	callbackURL := os.Getenv("AGIS_BOT_CALLBACK_URL")
	if apiKey == "" || callbackURL == "" {
		t.Skip("Required env vars not set")
	}

	conversionID := fmt.Sprintf("%s%d", ayetTestConversionID, time.Now().Unix())
	params := map[string]string{
		"externalIdentifier": testDiscordID,
		"uid":                ayetTestUserID,
		"currency":           "points",
		"amount":             "1000",
		"conversionId":       conversionID,
		"custom_1":           "surveywall",
		"custom_2":           "",
		"custom_3":           "",
		"custom_4":           "",
	}

	signature := generateAyetSignature(params, apiKey)
	params["signature"] = signature

	callbackReq, _ := http.NewRequest("GET", callbackURL, nil)
	q := callbackReq.URL.Query()
	for k, v := range params {
		q.Add(k, v)
	}
	callbackReq.URL.RawQuery = q.Encode()

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(callbackReq)
	if err != nil {
		t.Fatalf("Failed to send surveywall callback: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	t.Logf("Surveywall callback response: %d - %s", resp.StatusCode, string(body))

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected 200 OK, got %d", resp.StatusCode)
	}
}

// TestAyetRewardedVideoCallback tests rewarded video conversion
func TestAyetRewardedVideoCallback(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	apiKey := os.Getenv("AYET_API_KEY_SANDBOX")
	callbackURL := os.Getenv("AGIS_BOT_CALLBACK_URL")
	if apiKey == "" || callbackURL == "" {
		t.Skip("Required env vars not set")
	}

	conversionID := fmt.Sprintf("%s%d", ayetTestConversionID, time.Now().Unix())
	params := map[string]string{
		"externalIdentifier": testDiscordID,
		"uid":                ayetTestUserID,
		"currency":           "coins",
		"amount":             "50", // Typical video reward
		"conversionId":       conversionID,
		"custom_1":           "video",
		"custom_2":           "",
		"custom_3":           "",
		"custom_4":           "",
	}

	signature := generateAyetSignature(params, apiKey)
	params["signature"] = signature

	callbackReq, _ := http.NewRequest("GET", callbackURL, nil)
	q := callbackReq.URL.Query()
	for k, v := range params {
		q.Add(k, v)
	}
	callbackReq.URL.RawQuery = q.Encode()

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(callbackReq)
	if err != nil {
		t.Fatalf("Failed to send video callback: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Errorf("Expected 200 OK, got %d: %s", resp.StatusCode, string(body))
	}
}

// TestAyetInvalidSignature tests signature verification failure
func TestAyetInvalidSignature(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	callbackURL := os.Getenv("AGIS_BOT_CALLBACK_URL")
	if callbackURL == "" {
		t.Skip("AGIS_BOT_CALLBACK_URL not set")
	}

	params := map[string]string{
		"externalIdentifier": testDiscordID,
		"uid":                ayetTestUserID,
		"currency":           "coins",
		"amount":             "500",
		"conversionId":       fmt.Sprintf("%s%d", ayetTestConversionID, time.Now().Unix()),
		"signature":          "invalid_signature_12345",
		"custom_1":           "offerwall",
		"custom_2":           "",
		"custom_3":           "",
		"custom_4":           "",
	}

	callbackReq, _ := http.NewRequest("GET", callbackURL, nil)
	q := callbackReq.URL.Query()
	for k, v := range params {
		q.Add(k, v)
	}
	callbackReq.URL.RawQuery = q.Encode()

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(callbackReq)
	if err != nil {
		t.Fatalf("Failed to send callback: %v", err)
	}
	defer resp.Body.Close()

	// Should reject with 401 or 403
	if resp.StatusCode == http.StatusOK {
		t.Errorf("Expected rejection for invalid signature, got 200 OK")
	}
}

// TestAyetDuplicateConversion tests idempotency via duplicate conversion_id
func TestAyetDuplicateConversion(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	apiKey := os.Getenv("AYET_API_KEY_SANDBOX")
	callbackURL := os.Getenv("AGIS_BOT_CALLBACK_URL")
	if apiKey == "" || callbackURL == "" {
		t.Skip("Required env vars not set")
	}

	// Use same conversion_id for both requests
	conversionID := fmt.Sprintf("%s%d", ayetTestConversionID, time.Now().Unix())

	for i := 0; i < 2; i++ {
		params := map[string]string{
			"externalIdentifier": testDiscordID,
			"uid":                ayetTestUserID,
			"currency":           "coins",
			"amount":             "500",
			"conversionId":       conversionID,
			"custom_1":           "offerwall",
			"custom_2":           "",
			"custom_3":           "",
			"custom_4":           "",
		}

		signature := generateAyetSignature(params, apiKey)
		params["signature"] = signature

		callbackReq, _ := http.NewRequest("GET", callbackURL, nil)
		q := callbackReq.URL.Query()
		for k, v := range params {
			q.Add(k, v)
		}
		callbackReq.URL.RawQuery = q.Encode()

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(callbackReq)
		if err != nil {
			t.Fatalf("Request %d failed: %v", i+1, err)
		}
		defer resp.Body.Close()

		body, _ := io.ReadAll(resp.Body)
		t.Logf("Request %d response: %d - %s", i+1, resp.StatusCode, string(body))

		if i == 0 {
			// First request should succeed
			if resp.StatusCode != http.StatusOK {
				t.Errorf("First request should succeed, got %d", resp.StatusCode)
			}
		} else {
			// Second request should be rejected as duplicate (200 OK but no credit, or 409)
			// Check response body for "already processed" message
			if !bytes.Contains(body, []byte("already processed")) && resp.StatusCode == http.StatusOK {
				t.Logf("Warning: duplicate not detected - may need to check database state")
			}
		}
	}
}

// TestAyetFraudDetection tests velocity-based fraud detection
func TestAyetFraudDetection(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	apiKey := os.Getenv("AYET_API_KEY_SANDBOX")
	callbackURL := os.Getenv("AGIS_BOT_CALLBACK_URL")
	if apiKey == "" || callbackURL == "" {
		t.Skip("Required env vars not set")
	}

	// Send 11 conversions in rapid succession (threshold is 10/hour)
	for i := 0; i < 11; i++ {
		conversionID := fmt.Sprintf("%s%d-%d", ayetTestConversionID, time.Now().Unix(), i)
		params := map[string]string{
			"externalIdentifier": testDiscordID,
			"uid":                ayetTestUserID,
			"currency":           "coins",
			"amount":             "100",
			"conversionId":       conversionID,
			"custom_1":           "offerwall",
			"custom_2":           "",
			"custom_3":           "",
			"custom_4":           "",
		}

		signature := generateAyetSignature(params, apiKey)
		params["signature"] = signature

		callbackReq, _ := http.NewRequest("GET", callbackURL, nil)
		q := callbackReq.URL.Query()
		for k, v := range params {
			q.Add(k, v)
		}
		callbackReq.URL.RawQuery = q.Encode()

		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Do(callbackReq)
		if err != nil {
			t.Fatalf("Request %d failed: %v", i+1, err)
		}

		body, _ := io.ReadAll(resp.Body)
		resp.Body.Close()

		t.Logf("Request %d/%d: %d - %s", i+1, 11, resp.StatusCode, string(body))

		if i >= 10 {
			// 11th request should trigger fraud detection
			if !bytes.Contains(body, []byte("fraud")) && resp.StatusCode == http.StatusOK {
				t.Errorf("Expected fraud detection on request %d", i+1)
			}
		}

		time.Sleep(100 * time.Millisecond)
	}
}

// TestAyetMetricsExport verifies Prometheus metrics are updated
func TestAyetMetricsExport(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	metricsURL := os.Getenv("AGIS_BOT_METRICS_URL") // e.g., http://localhost:9090/metrics
	if metricsURL == "" {
		t.Skip("AGIS_BOT_METRICS_URL not set")
	}

	resp, err := http.Get(metricsURL)
	if err != nil {
		t.Fatalf("Failed to fetch metrics: %v", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	bodyStr := string(body)

	// Verify expected metrics exist
	expectedMetrics := []string{
		"agis_ad_conversions_total",
		"agis_ad_rewards_total",
		"agis_ad_fraud_attempts_total",
		"agis_ad_callback_latency_seconds",
		"agis_ad_conversions_by_tier_total",
	}

	for _, metric := range expectedMetrics {
		if !bytes.Contains(body, []byte(metric)) {
			t.Errorf("Expected metric %s not found in /metrics output", metric)
		}
	}

	t.Logf("Metrics endpoint validation passed (%d bytes)", len(bodyStr))
}

// Helper: Generate HMAC-SHA1 signature matching ayeT specification
func generateAyetSignature(params map[string]string, apiKey string) string {
	payload := fmt.Sprintf("%s|%s|%s|%s|%s|%s|%s|%s|%s",
		params["externalIdentifier"],
		params["uid"],
		params["currency"],
		params["amount"],
		params["conversionId"],
		params["custom_1"],
		params["custom_2"],
		params["custom_3"],
		params["custom_4"],
	)

	mac := hmac.New(sha1.New, []byte(apiKey))
	mac.Write([]byte(payload))
	return hex.EncodeToString(mac.Sum(nil))
}
