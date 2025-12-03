// Package opensaas provides integration endpoints for OpenSaaS/Wasp web applications.
// These endpoints support the Wasp framework's authentication, payments, and user management.
package opensaas

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"
)

// UserService defines the interface for user operations.
type UserService interface {
	GetUserByDiscordID(ctx context.Context, discordID string) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	LinkDiscordAccount(ctx context.Context, userID int, discordID string) error
	GetUserCredits(ctx context.Context, userID int) (credits int, wtgCoins int, err error)
	UpdateUserTier(ctx context.Context, userID int, tier string, expiresAt *time.Time) error
}

// PaymentService defines the interface for payment operations.
type PaymentService interface {
	CreateCheckoutSession(ctx context.Context, userID int, packageID string, successURL, cancelURL string) (string, error)
	HandleWebhook(ctx context.Context, payload []byte, signature string) error
	GetPaymentHistory(ctx context.Context, userID int, limit int) ([]Payment, error)
}

// ServerService defines the interface for server operations.
type ServerService interface {
	GetUserServers(ctx context.Context, userID int) ([]Server, error)
	CreateServer(ctx context.Context, userID int, gameType, name string) (*Server, error)
	DeleteServer(ctx context.Context, userID int, serverID int) error
	ControlServer(ctx context.Context, userID int, serverID int, action string) error
}

// Handler provides HTTP handlers for OpenSaaS integration.
type Handler struct {
	userService    UserService
	paymentService PaymentService
	serverService  ServerService
	logger         *slog.Logger
}

// User represents a user in the system.
type User struct {
	ID           int        `json:"id"`
	Email        string     `json:"email,omitempty"`
	DiscordID    string     `json:"discord_id"`
	Username     string     `json:"username"`
	Credits      int        `json:"credits"`
	WTGCoins     int        `json:"wtg_coins"`
	Tier         string     `json:"tier"`
	TierExpires  *time.Time `json:"tier_expires,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	LastActiveAt time.Time  `json:"last_active_at"`
}

// Payment represents a payment transaction.
type Payment struct {
	ID          string    `json:"id"`
	UserID      int       `json:"user_id"`
	Amount      int64     `json:"amount"`
	Currency    string    `json:"currency"`
	Status      string    `json:"status"`
	Description string    `json:"description"`
	CreatedAt   time.Time `json:"created_at"`
}

// Server represents a game server.
type Server struct {
	ID          int       `json:"id"`
	UserID      int       `json:"user_id"`
	Name        string    `json:"name"`
	GameType    string    `json:"game_type"`
	Status      string    `json:"status"`
	Address     string    `json:"address,omitempty"`
	Port        int       `json:"port,omitempty"`
	CostPerHour int       `json:"cost_per_hour"`
	CreatedAt   time.Time `json:"created_at"`
}

// NewHandler creates a new OpenSaaS integration handler.
func NewHandler(
	userSvc UserService,
	paymentSvc PaymentService,
	serverSvc ServerService,
	logger *slog.Logger,
) *Handler {
	return &Handler{
		userService:    userSvc,
		paymentService: paymentSvc,
		serverService:  serverSvc,
		logger:         logger,
	}
}

// RegisterRoutes registers all OpenSaaS integration routes on the given mux.
func (h *Handler) RegisterRoutes(mux *http.ServeMux) {
	// User endpoints
	mux.HandleFunc("GET /api/opensaas/v1/user/me", h.authMiddleware(h.handleGetCurrentUser))
	mux.HandleFunc("POST /api/opensaas/v1/user/link-discord", h.authMiddleware(h.handleLinkDiscord))
	mux.HandleFunc("GET /api/opensaas/v1/user/credits", h.authMiddleware(h.handleGetCredits))

	// Server endpoints
	mux.HandleFunc("GET /api/opensaas/v1/servers", h.authMiddleware(h.handleListServers))
	mux.HandleFunc("POST /api/opensaas/v1/servers", h.authMiddleware(h.handleCreateServer))
	mux.HandleFunc("DELETE /api/opensaas/v1/servers/{id}", h.authMiddleware(h.handleDeleteServer))
	mux.HandleFunc("POST /api/opensaas/v1/servers/{id}/control", h.authMiddleware(h.handleControlServer))

	// Payment endpoints (Stripe/LemonSqueezy compatible)
	mux.HandleFunc("POST /api/opensaas/v1/payments/checkout", h.authMiddleware(h.handleCreateCheckout))
	mux.HandleFunc("GET /api/opensaas/v1/payments/history", h.authMiddleware(h.handlePaymentHistory))
	mux.HandleFunc("POST /api/opensaas/v1/payments/webhook/stripe", h.handleStripeWebhook)

	// Subscription endpoints
	mux.HandleFunc("GET /api/opensaas/v1/subscription", h.authMiddleware(h.handleGetSubscription))
	mux.HandleFunc("POST /api/opensaas/v1/subscription/upgrade", h.authMiddleware(h.handleUpgradeSubscription))
	mux.HandleFunc("POST /api/opensaas/v1/subscription/cancel", h.authMiddleware(h.handleCancelSubscription))

	// Game catalog
	mux.HandleFunc("GET /api/opensaas/v1/games", h.handleListGames)
	mux.HandleFunc("GET /api/opensaas/v1/shop/packages", h.handleListPackages)

	// Health/status
	mux.HandleFunc("GET /api/opensaas/v1/health", h.handleHealth)
}

// Response helpers
type apiResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *apiError   `json:"error,omitempty"`
}

type apiError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (h *Handler) respondJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(apiResponse{Success: true, Data: data})
}

func (h *Handler) respondError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(apiResponse{
		Success: false,
		Error:   &apiError{Code: code, Message: message},
	})
}

// Context keys
type contextKey string

const (
	contextKeyUserID contextKey = "user_id"
)

// Middleware for JWT authentication (compatible with Wasp auth)
func (h *Handler) authMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Extract token from Authorization header
		auth := r.Header.Get("Authorization")
		if auth == "" {
			h.respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authorization header required")
			return
		}

		// Support "Bearer <token>" format
		parts := strings.Split(auth, " ")
		if len(parts) != 2 || parts[0] != "Bearer" {
			h.respondError(w, http.StatusUnauthorized, "INVALID_TOKEN", "Invalid authorization format")
			return
		}

		token := parts[1]
		// TODO: Validate JWT token from Wasp auth
		// For now, we'll use a placeholder that accepts discord IDs
		// In production, this should validate the Wasp JWT and extract user info

		// Extract user ID from context (set by Wasp middleware or our validation)
		ctx := context.WithValue(r.Context(), contextKeyUserID, token)
		next(w, r.WithContext(ctx))
	}
}

// Handler implementations

func (h *Handler) handleGetCurrentUser(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(contextKeyUserID).(string)

	user, err := h.userService.GetUserByDiscordID(r.Context(), userID)
	if err != nil {
		h.respondError(w, http.StatusNotFound, "USER_NOT_FOUND", "User not found")
		return
	}

	h.respondJSON(w, http.StatusOK, user)
}

func (h *Handler) handleLinkDiscord(w http.ResponseWriter, r *http.Request) {
	// Parse request
	var req struct {
		DiscordID string `json:"discord_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	// TODO: Get user ID from Wasp JWT
	// For now, placeholder
	h.respondJSON(w, http.StatusOK, map[string]string{"status": "linked"})
}

func (h *Handler) handleGetCredits(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(contextKeyUserID).(string)

	user, err := h.userService.GetUserByDiscordID(r.Context(), userID)
	if err != nil {
		h.respondError(w, http.StatusNotFound, "USER_NOT_FOUND", "User not found")
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"credits":   user.Credits,
		"wtg_coins": user.WTGCoins,
		"tier":      user.Tier,
	})
}

func (h *Handler) handleListServers(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(contextKeyUserID).(string)
	// TODO: Convert discord ID to user ID
	_ = userID

	// Placeholder response
	h.respondJSON(w, http.StatusOK, []Server{})
}

func (h *Handler) handleCreateServer(w http.ResponseWriter, r *http.Request) {
	var req struct {
		GameType string `json:"game_type"`
		Name     string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	if req.GameType == "" || req.Name == "" {
		h.respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", "game_type and name are required")
		return
	}

	// TODO: Create server via ServerService
	h.respondJSON(w, http.StatusCreated, map[string]string{"status": "creating"})
}

func (h *Handler) handleDeleteServer(w http.ResponseWriter, r *http.Request) {
	serverID := r.PathValue("id")
	if serverID == "" {
		h.respondError(w, http.StatusBadRequest, "INVALID_ID", "Server ID required")
		return
	}

	// TODO: Delete server
	h.respondJSON(w, http.StatusOK, map[string]string{"status": "deleted"})
}

func (h *Handler) handleControlServer(w http.ResponseWriter, r *http.Request) {
	serverID := r.PathValue("id")
	if serverID == "" {
		h.respondError(w, http.StatusBadRequest, "INVALID_ID", "Server ID required")
		return
	}

	var req struct {
		Action string `json:"action"` // start, stop, restart
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	validActions := map[string]bool{"start": true, "stop": true, "restart": true}
	if !validActions[req.Action] {
		h.respondError(w, http.StatusBadRequest, "INVALID_ACTION", "Action must be start, stop, or restart")
		return
	}

	// TODO: Control server
	h.respondJSON(w, http.StatusOK, map[string]string{"status": req.Action + "ing"})
}

func (h *Handler) handleCreateCheckout(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PackageID  string `json:"package_id"`
		SuccessURL string `json:"success_url"`
		CancelURL  string `json:"cancel_url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	// TODO: Create Stripe checkout session
	h.respondJSON(w, http.StatusOK, map[string]string{
		"checkout_url": fmt.Sprintf("https://checkout.stripe.com/pay/placeholder_%s", req.PackageID),
	})
}

func (h *Handler) handlePaymentHistory(w http.ResponseWriter, r *http.Request) {
	// TODO: Get payment history
	h.respondJSON(w, http.StatusOK, []Payment{})
}

func (h *Handler) handleStripeWebhook(w http.ResponseWriter, r *http.Request) {
	// Read webhook payload
	// TODO: Validate signature and process webhook
	h.respondJSON(w, http.StatusOK, map[string]string{"received": "true"})
}

func (h *Handler) handleGetSubscription(w http.ResponseWriter, r *http.Request) {
	userID := r.Context().Value(contextKeyUserID).(string)

	user, err := h.userService.GetUserByDiscordID(r.Context(), userID)
	if err != nil {
		h.respondError(w, http.StatusNotFound, "USER_NOT_FOUND", "User not found")
		return
	}

	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"tier":        user.Tier,
		"expires_at":  user.TierExpires,
		"is_active":   user.Tier != "free",
		"can_upgrade": user.Tier != "premium_plus",
	})
}

func (h *Handler) handleUpgradeSubscription(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Tier string `json:"tier"` // premium, premium_plus
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		h.respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	// TODO: Create subscription checkout
	h.respondJSON(w, http.StatusOK, map[string]string{
		"checkout_url": "https://checkout.stripe.com/pay/subscription_placeholder",
	})
}

func (h *Handler) handleCancelSubscription(w http.ResponseWriter, r *http.Request) {
	// TODO: Cancel subscription
	h.respondJSON(w, http.StatusOK, map[string]string{"status": "cancelled"})
}

func (h *Handler) handleListGames(w http.ResponseWriter, r *http.Request) {
	// Static game catalog
	games := []map[string]interface{}{
		{"id": "minecraft", "name": "Minecraft", "cost_per_hour": 30, "tier": "standard", "enabled": true},
		{"id": "terraria", "name": "Terraria", "cost_per_hour": 35, "tier": "standard", "enabled": true},
		{"id": "valheim", "name": "Valheim", "cost_per_hour": 120, "tier": "demanding", "enabled": true},
		{"id": "rust", "name": "Rust", "cost_per_hour": 220, "tier": "enterprise", "enabled": true},
		{"id": "ark", "name": "ARK: Survival Evolved", "cost_per_hour": 240, "tier": "enterprise", "enabled": true},
		{"id": "palworld", "name": "Palworld", "cost_per_hour": 180, "tier": "demanding", "enabled": true},
		{"id": "cs2", "name": "Counter-Strike 2", "cost_per_hour": 120, "tier": "demanding", "enabled": true},
		{"id": "gmod", "name": "Garry's Mod", "cost_per_hour": 95, "tier": "demanding", "enabled": true},
		{"id": "factorio", "name": "Factorio", "cost_per_hour": 100, "tier": "demanding", "enabled": true},
	}
	h.respondJSON(w, http.StatusOK, games)
}

func (h *Handler) handleListPackages(w http.ResponseWriter, r *http.Request) {
	// WTG Coin packages
	packages := []map[string]interface{}{
		{"id": "wtg_5", "name": "5 WTG Coins", "price_cents": 499, "coins": 5, "bonus": 0, "popular": false},
		{"id": "wtg_11", "name": "11 WTG Coins", "price_cents": 999, "coins": 10, "bonus": 1, "popular": true},
		{"id": "wtg_23", "name": "23 WTG Coins", "price_cents": 1999, "coins": 20, "bonus": 3, "popular": false},
		{"id": "wtg_60", "name": "60 WTG Coins", "price_cents": 4999, "coins": 50, "bonus": 10, "popular": false},
	}
	h.respondJSON(w, http.StatusOK, packages)
}

func (h *Handler) handleHealth(w http.ResponseWriter, r *http.Request) {
	h.respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "healthy",
		"version":   "v1.0.0",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}
