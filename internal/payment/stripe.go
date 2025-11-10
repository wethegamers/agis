package payment

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"

	"github.com/stripe/stripe-go/v76"
	"github.com/stripe/stripe-go/v76/checkout/session"
	"github.com/stripe/stripe-go/v76/webhook"
)

// StripeService handles Stripe payment processing
type StripeService struct {
	SecretKey      string
	WebhookSecret  string
	SuccessURL     string
	CancelURL      string
	TestMode       bool
}

// NewStripeService creates a new Stripe service
func NewStripeService(secretKey, webhookSecret, successURL, cancelURL string, testMode bool) *StripeService {
	stripe.Key = secretKey
	return &StripeService{
		SecretKey:     secretKey,
		WebhookSecret: webhookSecret,
		SuccessURL:    successURL,
		CancelURL:     cancelURL,
		TestMode:      testMode,
	}
}

// WTGPackage represents a WTG coin package
type WTGPackage struct {
	ID          string
	Name        string
	Amount      int64  // Amount in cents ($4.99 = 499)
	WTGCoins    int    // Number of WTG coins
	BonusCoins  int    // Bonus WTG coins
	Description string
}

// PredefinedPackages returns the standard WTG packages
func PredefinedPackages() []WTGPackage {
	return []WTGPackage{
		{
			ID:          "wtg_5",
			Name:        "5 WTG Coins",
			Amount:      499,
			WTGCoins:    5,
			BonusCoins:  0,
			Description: "Entry-level WTG package",
		},
		{
			ID:          "wtg_11",
			Name:        "11 WTG Coins",
			Amount:      999,
			WTGCoins:    10,
			BonusCoins:  1,
			Description: "Best value for casual users - includes 1 bonus coin!",
		},
		{
			ID:          "wtg_23",
			Name:        "23 WTG Coins",
			Amount:      1999,
			WTGCoins:    20,
			BonusCoins:  3,
			Description: "Popular choice - includes 3 bonus coins!",
		},
		{
			ID:          "wtg_60",
			Name:        "60 WTG Coins",
			Amount:      4999,
			WTGCoins:    50,
			BonusCoins:  10,
			Description: "Maximum value - includes 10 bonus coins!",
		},
	}
}

// CreateCheckoutSession creates a Stripe checkout session for WTG purchase
func (s *StripeService) CreateCheckoutSession(packageID, discordID, discordUsername string) (*stripe.CheckoutSession, error) {
	// Find the package
	var pkg *WTGPackage
	for _, p := range PredefinedPackages() {
		if p.ID == packageID {
			pkg = &p
			break
		}
	}

	if pkg == nil {
		return nil, fmt.Errorf("package not found: %s", packageID)
	}

	// Create checkout session
	params := &stripe.CheckoutSessionParams{
		Mode: stripe.String(string(stripe.CheckoutSessionModePayment)),
		LineItems: []*stripe.CheckoutSessionLineItemParams{
			{
				PriceData: &stripe.CheckoutSessionLineItemPriceDataParams{
					Currency: stripe.String("usd"),
					ProductData: &stripe.CheckoutSessionLineItemPriceDataProductDataParams{
						Name:        stripe.String(pkg.Name),
						Description: stripe.String(pkg.Description),
					},
					UnitAmount: stripe.Int64(pkg.Amount),
				},
				Quantity: stripe.Int64(1),
			},
		},
		SuccessURL: stripe.String(s.SuccessURL + "?session_id={CHECKOUT_SESSION_ID}"),
		CancelURL:  stripe.String(s.CancelURL),
		Metadata: map[string]string{
			"discord_id":       discordID,
			"discord_username": discordUsername,
			"package_id":       packageID,
			"wtg_coins":        strconv.Itoa(pkg.WTGCoins),
			"bonus_coins":      strconv.Itoa(pkg.BonusCoins),
			"total_coins":      strconv.Itoa(pkg.WTGCoins + pkg.BonusCoins),
		},
	}

	sess, err := session.New(params)
	if err != nil {
		return nil, fmt.Errorf("failed to create checkout session: %v", err)
	}

	log.Printf("✅ Created Stripe checkout session: %s for user %s (%s)", sess.ID, discordUsername, discordID)
	return sess, nil
}

// WebhookEvent represents a processed webhook event
type WebhookEvent struct {
	Type       string
	DiscordID  string
	WTGCoins   int
	SessionID  string
	AmountPaid int64
}

// Implement interface methods for HTTP server
func (w *WebhookEvent) GetDiscordID() string { return w.DiscordID }
func (w *WebhookEvent) GetWTGCoins() int     { return w.WTGCoins }
func (w *WebhookEvent) GetSessionID() string { return w.SessionID }
func (w *WebhookEvent) GetAmountPaid() int64 { return w.AmountPaid }

// HandleWebhook processes Stripe webhook events
func (s *StripeService) HandleWebhook(w http.ResponseWriter, r *http.Request) (interface{}, error) {
	const MaxBodyBytes = int64(65536)
	r.Body = http.MaxBytesReader(w, r.Body, MaxBodyBytes)

	payload, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, fmt.Errorf("error reading request body: %v", err)
	}

	// Verify webhook signature
	event, err := webhook.ConstructEvent(payload, r.Header.Get("Stripe-Signature"), s.WebhookSecret)
	if err != nil {
		return nil, fmt.Errorf("webhook signature verification failed: %v", err)
	}

	log.Printf("📥 Received Stripe webhook: %s", event.Type)
	log.Printf("Stripe payload size: %d bytes", len(payload))

	// Handle specific event types
	switch event.Type {
	case "checkout.session.completed":
		var sess stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &sess); err != nil {
			return nil, fmt.Errorf("error parsing webhook JSON: %v", err)
		}

		// Extract metadata
		discordID := sess.Metadata["discord_id"]
		totalCoins, _ := strconv.Atoi(sess.Metadata["total_coins"])

		webhookEvent := &WebhookEvent{
			Type:       "purchase_completed",
			DiscordID:  discordID,
			WTGCoins:   totalCoins,
			SessionID:  sess.ID,
			AmountPaid: sess.AmountTotal,
		}

		log.Printf("✅ Payment successful: User %s purchased %d WTG coins for $%.2f",
			discordID, totalCoins, float64(sess.AmountTotal)/100)

		return webhookEvent, nil

	case "checkout.session.expired":
		var sess stripe.CheckoutSession
		if err := json.Unmarshal(event.Data.Raw, &sess); err != nil {
			return nil, fmt.Errorf("error parsing webhook JSON: %v", err)
		}

		log.Printf("⏰ Checkout session expired: %s", sess.ID)
		return &WebhookEvent{
			Type:      "session_expired",
			SessionID: sess.ID,
		}, nil

	default:
		log.Printf("ℹ️ Unhandled webhook event type: %s", event.Type)
		return nil, nil
	}
}

// GetPackage returns a package by ID
func GetPackage(packageID string) *WTGPackage {
	for _, pkg := range PredefinedPackages() {
		if pkg.ID == packageID {
			return &pkg
		}
	}
	return nil
}
