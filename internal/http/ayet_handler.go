package http

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"agis-bot/internal/services"
)

// AyetHandler handles ayeT-Studios S2S conversion callbacks
type AyetHandler struct {
	adService *services.AdConversionService
}

// NewAyetHandler creates a new ayeT-Studios callback handler
func NewAyetHandler(adService *services.AdConversionService) *AyetHandler {
	return &AyetHandler{
		adService: adService,
	}
}

// HandleCallback processes ayeT-Studios S2S callback
// Endpoint: GET /ads/ayet/callback
// Expected params: externalIdentifier|uid, currency|amount, conversionId, signature, custom_1..custom_4
func (h *AyetHandler) HandleCallback(w http.ResponseWriter, r *http.Request) {
	// Parse query parameters
	query := r.URL.Query()

	// Extract user identifier (try both fields)
	externalID := query.Get("externalIdentifier")
	uid := query.Get("uid")
	if externalID == "" && uid == "" {
		log.Printf("⚠️ Ayet callback missing user identifier: %s", r.URL.RawQuery)
		http.Error(w, "missing user identifier", http.StatusBadRequest)
		return
	}

	// Extract conversion data
	conversionID := query.Get("conversionId")
	if conversionID == "" {
		conversionID = query.Get("conversion_id") // Try alternative
	}
	if conversionID == "" {
		log.Printf("⚠️ Ayet callback missing conversionId: %s", r.URL.RawQuery)
		http.Error(w, "missing conversionId", http.StatusBadRequest)
		return
	}

	// Extract currency and amount
	currency := query.Get("currency")
	if currency == "" {
		currency = "coins" // default
	}

	amountStr := query.Get("amount")
	if amountStr == "" {
		log.Printf("⚠️ Ayet callback missing amount: %s", r.URL.RawQuery)
		http.Error(w, "missing amount", http.StatusBadRequest)
		return
	}

	amount, err := strconv.Atoi(amountStr)
	if err != nil || amount <= 0 {
		log.Printf("⚠️ Ayet callback invalid amount: %s", amountStr)
		http.Error(w, "invalid amount", http.StatusBadRequest)
		return
	}

	// Extract signature
	signature := query.Get("signature")
	if signature == "" {
		log.Printf("⚠️ Ayet callback missing signature: %s", r.URL.RawQuery)
		http.Error(w, "missing signature", http.StatusBadRequest)
		return
	}

	// Extract custom parameters (optional)
	custom1 := query.Get("custom_1")
	custom2 := query.Get("custom_2")
	custom3 := query.Get("custom_3")
	custom4 := query.Get("custom_4")

	// Build params struct
	params := services.AyetCallbackParams{
		ExternalIdentifier: externalID,
		UID:                uid,
		Currency:           currency,
		Amount:             amount,
		ConversionID:       conversionID,
		Signature:          signature,
		Custom1:            custom1,
		Custom2:            custom2,
		Custom3:            custom3,
		Custom4:            custom4,
		IPAddress:          getClientIP(r),
		UserAgent:          r.UserAgent(),
	}

	// Process callback with timeout
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	err = h.adService.ProcessAyetCallback(ctx, params)
	if err != nil {
		switch err {
		case services.ErrInvalidSignature:
			log.Printf("🚨 Ayet callback invalid signature: %s", conversionID)
			http.Error(w, "invalid signature", http.StatusUnauthorized)
		case services.ErrDuplicateConversion:
			// Return 200 OK for duplicates (idempotent)
			log.Printf("ℹ️ Ayet callback duplicate: %s", conversionID)
			w.WriteHeader(http.StatusOK)
			fmt.Fprintf(w, "OK")
		case services.ErrConsentRequired:
			log.Printf("⚠️ Ayet callback consent required: %s for user %s", conversionID, externalID)
			http.Error(w, "consent required", http.StatusForbidden)
		case services.ErrInvalidAmount:
			log.Printf("⚠️ Ayet callback invalid amount: %s", conversionID)
			http.Error(w, "invalid amount", http.StatusBadRequest)
		default:
			log.Printf("❌ Ayet callback processing error: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
		}
		return
	}

	// Success response (ayeT-Studios expects "OK" or 200 status)
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, "OK")
	log.Printf("✅ Ayet callback processed: %s", conversionID)
}

// HandleStatus provides a status endpoint for testing
// Endpoint: GET /ads/ayet/status
func (h *AyetHandler) HandleStatus(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"service": "ayeT-Studios S2S",
		"status":  "operational",
		"version": "1.0",
	})
}

// getClientIP extracts the client's IP address from request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header (if behind proxy/load balancer)
	forwarded := r.Header.Get("X-Forwarded-For")
	if forwarded != "" {
		// Take first IP in the list
		if idx := len(forwarded); idx > 0 {
			return forwarded[:idx]
		}
	}

	// Check X-Real-IP header
	realIP := r.Header.Get("X-Real-IP")
	if realIP != "" {
		return realIP
	}

	// Fall back to RemoteAddr
	return r.RemoteAddr
}
