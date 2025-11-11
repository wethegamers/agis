package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"agis-bot/internal/services"

	"github.com/gorilla/mux"
)

// APIServer handles REST API requests for v1.7.0
type APIServer struct {
	router      *mux.Router
	address     string
	db          *services.DatabaseService
	agones      *services.AgonesService
	enhanced    *services.EnhancedServerService
	apiKeysSvc  *services.APIKeyService
	rateLimiter *services.RateLimiter
	server      *http.Server
}

// NewAPIServer creates a new REST API server
func NewAPIServer(address string, db *services.DatabaseService, agones *services.AgonesService, enhanced *services.EnhancedServerService) *APIServer {
	api := &APIServer{
		router:      mux.NewRouter(),
		address:     address,
		db:          db,
		agones:      agones,
		enhanced:    enhanced,
		apiKeysSvc:  services.NewAPIKeyService(db.DB()),
		rateLimiter: services.NewRateLimiter(),
	}

	// Setup routes
	api.setupRoutes()

	return api
}

// setupRoutes configures all API endpoints
func (api *APIServer) setupRoutes() {
	// API v1 routes
	v1 := api.router.PathPrefix("/api/v1").Subrouter()

	// Middleware
	v1.Use(api.loggingMiddleware)
	v1.Use(api.authMiddleware)
	v1.Use(api.rateLimitMiddleware)

	// Server endpoints
	v1.HandleFunc("/servers", api.listServers).Methods("GET")
	v1.HandleFunc("/servers", api.createServer).Methods("POST")
	v1.HandleFunc("/servers/{id}", api.getServer).Methods("GET")
	v1.HandleFunc("/servers/{id}", api.deleteServer).Methods("DELETE")
	v1.HandleFunc("/servers/{id}/start", api.startServer).Methods("POST")
	v1.HandleFunc("/servers/{id}/stop", api.stopServer).Methods("POST")
	v1.HandleFunc("/servers/{id}/restart", api.restartServer).Methods("POST")

	// User endpoints
	v1.HandleFunc("/users/me", api.getCurrentUser).Methods("GET")
	v1.HandleFunc("/users/me/stats", api.getUserStats).Methods("GET")

	// Shop endpoints
	v1.HandleFunc("/shop", api.listShopPackages).Methods("GET")

	// Leaderboard endpoints
	v1.HandleFunc("/leaderboard/credits", api.getCreditsLeaderboard).Methods("GET")
	v1.HandleFunc("/leaderboard/servers", api.getServersLeaderboard).Methods("GET")

	// API Key management endpoints
	v1.HandleFunc("/auth/keys", api.createAPIKey).Methods("POST")
	v1.HandleFunc("/auth/keys", api.listAPIKeys).Methods("GET")
	v1.HandleFunc("/auth/keys/{id}", api.revokeAPIKey).Methods("DELETE")

	// Health check (no auth required)
	api.router.HandleFunc("/api/health", api.healthCheck).Methods("GET")
}

// ServeHTTP implements http.Handler interface
func (api *APIServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	api.router.ServeHTTP(w, r)
}

// Start starts the API server
func (api *APIServer) Start() error {
	api.server = &http.Server{
		Addr:         api.address,
		Handler:      api.router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	log.Printf("REST API server listening on %s", api.address)
	return api.server.ListenAndServe()
}

// Stop gracefully shuts down the API server
func (api *APIServer) Stop(ctx context.Context) error {
	// Stop rate limiter cleanup goroutine
	if api.rateLimiter != nil {
		api.rateLimiter.Stop()
	}
	
	// Shutdown HTTP server
	if api.server != nil {
		return api.server.Shutdown(ctx)
	}
	return nil
}

// Response structures
type APIResponse struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   *APIError   `json:"error,omitempty"`
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ServerResponse struct {
	ID          int       `json:"id"`
	Name        string    `json:"name"`
	GameType    string    `json:"game_type"`
	Status      string    `json:"status"`
	Address     string    `json:"address,omitempty"`
	Port        int       `json:"port,omitempty"`
	CostPerHour int       `json:"cost_per_hour"`
	IsPublic    bool      `json:"is_public"`
	Description string    `json:"description,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
}

type UserResponse struct {
	DiscordID      string    `json:"discord_id"`
	Credits        int       `json:"credits"`
	WTGCoins       int       `json:"wtg_coins"`
	Tier           string    `json:"tier"`
	ServersUsed    int       `json:"servers_used"`
	JoinDate       time.Time `json:"join_date"`
	SubscriptionExpires *time.Time `json:"subscription_expires,omitempty"`
}

type UserStatsResponse struct {
	TotalServers  int `json:"total_servers_created"`
	TotalCommands int `json:"total_commands_used"`
	TotalEarned   int `json:"total_credits_earned"`
	TotalSpent    int `json:"total_credits_spent"`
	Rank          int `json:"rank,omitempty"`
}

type CreateServerRequest struct {
	GameType    string `json:"game_type"`
	ServerName  string `json:"server_name"`
	Description string `json:"description,omitempty"`
	IsPublic    bool   `json:"is_public,omitempty"`
}

// Middleware implementations

func (api *APIServer) loggingMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		log.Printf("API: %s %s from %s", r.Method, r.URL.Path, r.RemoteAddr)
		next.ServeHTTP(w, r)
		log.Printf("API: %s %s completed in %v", r.Method, r.URL.Path, time.Since(start))
	})
}

func (api *APIServer) authMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for health check
		if strings.HasSuffix(r.URL.Path, "/health") {
			next.ServeHTTP(w, r)
			return
		}

		// Extract API key from Authorization header
		authHeader := r.Header.Get("Authorization")
		if authHeader == "" {
			api.respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authorization header required")
			return
		}

		// Support both API key and legacy Bearer token formats
		parts := strings.Split(authHeader, " ")
		if len(parts) != 2 {
			api.respondError(w, http.StatusUnauthorized, "INVALID_AUTH", "Invalid authorization format")
			return
		}

		var discordID string
		
		// Check if it's an API key or legacy token
		if parts[0] == "Bearer" && strings.HasPrefix(parts[1], "agis_") {
			// API Key authentication
			apiKey, err := api.apiKeysSvc.ValidateAPIKey(r.Context(), parts[1])
			if err != nil {
				api.respondError(w, http.StatusUnauthorized, "INVALID_API_KEY", err.Error())
				return
			}
			discordID = apiKey.DiscordID
			
			// Store API key metadata in context for rate limiting
			ctx := context.WithValue(r.Context(), "api_key", apiKey)
			r = r.WithContext(ctx)
		} else if parts[0] == "Bearer" {
			// Legacy: Discord ID as bearer token (for backwards compatibility)
			discordID = parts[1]
		} else {
			api.respondError(w, http.StatusUnauthorized, "INVALID_AUTH", "Invalid authorization type")
			return
		}

		// Verify user exists
		user, err := api.db.GetOrCreateUser(discordID)
		if err != nil {
			api.respondError(w, http.StatusUnauthorized, "INVALID_USER", "Invalid user")
			return
		}

		// Add user to context
		ctx := context.WithValue(r.Context(), "discord_id", user.DiscordID)
		ctx = context.WithValue(ctx, "user", user)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func (api *APIServer) rateLimitMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Extract rate limit key and limit from context (set by authMiddleware)
		var rateLimitKey string
		var rateLimitValue int

		// Try to get API key from context (preferred for accurate rate limiting)
		if apiKey, ok := r.Context().Value("api_key").(*services.APIKey); ok && apiKey != nil {
			rateLimitKey = fmt.Sprintf("api_key:%d", apiKey.ID)
			rateLimitValue = apiKey.RateLimit
		} else if discordID, ok := r.Context().Value("discord_id").(string); ok {
			// Fallback to discord_id for legacy bearer tokens
			rateLimitKey = fmt.Sprintf("discord:%s", discordID)
			rateLimitValue = 100 // Default rate limit for legacy tokens
		} else {
			// No auth context, shouldn't happen after authMiddleware, but handle gracefully
			api.respondError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Authentication required")
			return
		}

		// Check rate limit
		if !api.rateLimiter.Allow(rateLimitKey, rateLimitValue) {
			// Rate limit exceeded - calculate retry after
			resetAfter := api.rateLimiter.ResetAfter(rateLimitKey)
			w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", rateLimitValue))
			w.Header().Set("X-RateLimit-Remaining", "0")
			w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(resetAfter).Unix()))
			w.Header().Set("Retry-After", fmt.Sprintf("%d", int(resetAfter.Seconds())))
			
			api.respondError(w, http.StatusTooManyRequests, "RATE_LIMIT_EXCEEDED", 
				fmt.Sprintf("Rate limit exceeded. Retry after %s", resetAfter.Round(time.Second)))
			return
		}

		// Rate limit passed - add headers and continue
		remaining := api.rateLimiter.GetRemaining(rateLimitKey, rateLimitValue)
		resetAfter := api.rateLimiter.ResetAfter(rateLimitKey)
		w.Header().Set("X-RateLimit-Limit", fmt.Sprintf("%d", rateLimitValue))
		w.Header().Set("X-RateLimit-Remaining", fmt.Sprintf("%d", remaining))
		w.Header().Set("X-RateLimit-Reset", fmt.Sprintf("%d", time.Now().Add(resetAfter).Unix()))

		next.ServeHTTP(w, r)
	})
}

// Handler implementations

func (api *APIServer) healthCheck(w http.ResponseWriter, r *http.Request) {
	api.respondSuccess(w, http.StatusOK, map[string]interface{}{
		"status":  "healthy",
		"version": "v1.7.0",
		"time":    time.Now().Format(time.RFC3339),
	})
}

func (api *APIServer) listServers(w http.ResponseWriter, r *http.Request) {
	discordID := r.Context().Value("discord_id").(string)

	servers, err := api.db.GetUserServers(discordID)
	if err != nil {
		api.respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to fetch servers")
		return
	}

	// Convert to response format
	serverResponses := make([]ServerResponse, 0, len(servers))
	for _, s := range servers {
		serverResponses = append(serverResponses, ServerResponse{
			ID:          s.ID,
			Name:        s.Name,
			GameType:    s.GameType,
			Status:      s.Status,
			Address:     s.Address,
			Port:        s.Port,
			CostPerHour: s.CostPerHour,
			IsPublic:    s.IsPublic,
			Description: s.Description,
			CreatedAt:   s.CreatedAt,
		})
	}

	api.respondSuccess(w, http.StatusOK, serverResponses)
}

func (api *APIServer) getServer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serverID, err := strconv.Atoi(vars["id"])
	if err != nil {
		api.respondError(w, http.StatusBadRequest, "INVALID_ID", "Invalid server ID")
		return
	}

	discordID := r.Context().Value("discord_id").(string)

	// Get server and verify ownership
	servers, err := api.db.GetUserServers(discordID)
	if err != nil {
		api.respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to fetch server")
		return
	}

	var server *services.GameServer
	for _, s := range servers {
		if s.ID == serverID {
			server = s
			break
		}
	}

	if server == nil {
		api.respondError(w, http.StatusNotFound, "NOT_FOUND", "Server not found")
		return
	}

	api.respondSuccess(w, http.StatusOK, ServerResponse{
		ID:          server.ID,
		Name:        server.Name,
		GameType:    server.GameType,
		Status:      server.Status,
		Address:     server.Address,
		Port:        server.Port,
		CostPerHour: server.CostPerHour,
		IsPublic:    server.IsPublic,
		Description: server.Description,
		CreatedAt:   server.CreatedAt,
	})
}

func (api *APIServer) createServer(w http.ResponseWriter, r *http.Request) {
	discordID := r.Context().Value("discord_id").(string)

	var req CreateServerRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid request body")
		return
	}

	// Validate input
	if req.GameType == "" || req.ServerName == "" {
		api.respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", "game_type and server_name are required")
		return
	}

	// Check user balance
	user, err := api.db.GetOrCreateUser(discordID)
	if err != nil {
		api.respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to get user")
		return
	}

	// Get pricing
	// TODO: Use PricingService when available, for now use defaults
	costPerHour := getDefaultGameCost(req.GameType)
	if costPerHour == 0 {
		api.respondError(w, http.StatusBadRequest, "INVALID_GAME", fmt.Sprintf("Unsupported game type: %s", req.GameType))
		return
	}

	if user.Credits < costPerHour {
		api.respondError(w, http.StatusPaymentRequired, "INSUFFICIENT_CREDITS", 
			fmt.Sprintf("Need %d credits, have %d", costPerHour, user.Credits))
		return
	}

	// Create server using enhanced service
	server, err := api.enhanced.CreateGameServer(context.Background(), discordID, req.GameType, req.ServerName, costPerHour)
	if err != nil {
		api.respondError(w, http.StatusInternalServerError, "CREATE_FAILED", fmt.Sprintf("Failed to create server: %v", err))
		return
	}

	// Deduct initial credits
	if err := api.db.AddCredits(discordID, -costPerHour); err != nil {
		log.Printf("Warning: Failed to deduct credits: %v", err)
	}

	api.respondSuccess(w, http.StatusCreated, ServerResponse{
		ID:          server.ID,
		Name:        server.Name,
		GameType:    server.GameType,
		Status:      server.Status,
		CostPerHour: server.CostPerHour,
		IsPublic:    server.IsPublic,
		Description: server.Description,
		CreatedAt:   server.CreatedAt,
	})
}

func (api *APIServer) deleteServer(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	serverID, err := strconv.Atoi(vars["id"])
	if err != nil {
		api.respondError(w, http.StatusBadRequest, "INVALID_ID", "Invalid server ID")
		return
	}

	discordID := r.Context().Value("discord_id").(string)

	// Verify ownership
	servers, err := api.db.GetUserServers(discordID)
	if err != nil {
		api.respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to fetch server")
		return
	}

	var server *services.GameServer
	for _, s := range servers {
		if s.ID == serverID {
			server = s
			break
		}
	}

	if server == nil {
		api.respondError(w, http.StatusNotFound, "NOT_FOUND", "Server not found")
		return
	}

	// Delete from Agones if exists
	if server.Name != "" {
		if err := api.agones.DeleteGameServer(context.Background(), server.Name); err != nil {
			log.Printf("Warning: Failed to delete Agones server: %v", err)
		}
	}

	// Delete from database
	if err := api.db.DeleteGameServer(serverID); err != nil {
		api.respondError(w, http.StatusInternalServerError, "DELETE_FAILED", "Failed to delete server")
		return
	}

	api.respondSuccess(w, http.StatusOK, map[string]interface{}{
		"message": "Server deleted successfully",
	})
}

func (api *APIServer) startServer(w http.ResponseWriter, r *http.Request) {
	api.respondError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Start server endpoint coming soon")
}

func (api *APIServer) stopServer(w http.ResponseWriter, r *http.Request) {
	api.respondError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Stop server endpoint coming soon")
}

func (api *APIServer) restartServer(w http.ResponseWriter, r *http.Request) {
	api.respondError(w, http.StatusNotImplemented, "NOT_IMPLEMENTED", "Restart server endpoint coming soon")
}

func (api *APIServer) getCurrentUser(w http.ResponseWriter, r *http.Request) {
	discordID := r.Context().Value("discord_id").(string)

	user, err := api.db.GetOrCreateUser(discordID)
	if err != nil {
		api.respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to fetch user")
		return
	}

	// Get subscription info
	var expiresAt *time.Time
	if user.Tier == "premium" {
		var expiry sql.NullTime
		err := api.db.DB().QueryRow(`SELECT subscription_expires FROM users WHERE discord_id = $1`, discordID).Scan(&expiry)
		if err == nil && expiry.Valid {
			expiresAt = &expiry.Time
		}
	}

	// Get WTG coins
	wtg, gc := api.db.GetUserCurrencies(discordID)

	api.respondSuccess(w, http.StatusOK, UserResponse{
		DiscordID:           user.DiscordID,
		Credits:             gc,
		WTGCoins:            wtg,
		Tier:                user.Tier,
		ServersUsed:         user.ServersUsed,
		JoinDate:            user.JoinDate,
		SubscriptionExpires: expiresAt,
	})
}

func (api *APIServer) getUserStats(w http.ResponseWriter, r *http.Request) {
	_ = r.Context().Value("discord_id").(string)

	// TODO: Implement user_stats table queries
	// For now, return placeholder
	api.respondSuccess(w, http.StatusOK, UserStatsResponse{
		TotalServers:  0,
		TotalCommands: 0,
		TotalEarned:   0,
		TotalSpent:    0,
	})
}

func (api *APIServer) listShopPackages(w http.ResponseWriter, r *http.Request) {
	packages := []map[string]interface{}{
		{
			"id":          "wtg_5",
			"name":        "5 WTG Coins",
			"amount_usd":  499,
			"wtg_coins":   5,
			"bonus_coins": 0,
		},
		{
			"id":          "wtg_11",
			"name":        "11 WTG Coins",
			"amount_usd":  999,
			"wtg_coins":   10,
			"bonus_coins": 1,
		},
		{
			"id":          "wtg_23",
			"name":        "23 WTG Coins",
			"amount_usd":  1999,
			"wtg_coins":   20,
			"bonus_coins": 3,
		},
		{
			"id":          "wtg_60",
			"name":        "60 WTG Coins",
			"amount_usd":  4999,
			"wtg_coins":   50,
			"bonus_coins": 10,
		},
	}

	api.respondSuccess(w, http.StatusOK, packages)
}

func (api *APIServer) getCreditsLeaderboard(w http.ResponseWriter, r *http.Request) {
	rows, err := api.db.DB().Query(`
		SELECT discord_id, credits, tier
		FROM users
		ORDER BY credits DESC
		LIMIT 100
	`)
	if err != nil {
		api.respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to fetch leaderboard")
		return
	}
	defer rows.Close()

	leaderboard := make([]map[string]interface{}, 0)
	rank := 1
	for rows.Next() {
		var discordID, tier string
		var credits int
		if err := rows.Scan(&discordID, &credits, &tier); err != nil {
			continue
		}

		leaderboard = append(leaderboard, map[string]interface{}{
			"rank":       rank,
			"discord_id": discordID,
			"credits":    credits,
			"tier":       tier,
		})
		rank++
	}

	api.respondSuccess(w, http.StatusOK, leaderboard)
}

func (api *APIServer) getServersLeaderboard(w http.ResponseWriter, r *http.Request) {
	rows, err := api.db.DB().Query(`
		SELECT discord_id, servers_used, tier
		FROM users
		ORDER BY servers_used DESC
		LIMIT 100
	`)
	if err != nil {
		api.respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", "Failed to fetch leaderboard")
		return
	}
	defer rows.Close()

	leaderboard := make([]map[string]interface{}, 0)
	rank := 1
	for rows.Next() {
		var discordID, tier string
		var servers int
		if err := rows.Scan(&discordID, &servers, &tier); err != nil {
			continue
		}

		leaderboard = append(leaderboard, map[string]interface{}{
			"rank":       rank,
			"discord_id": discordID,
			"servers":    servers,
			"tier":       tier,
		})
		rank++
	}

	api.respondSuccess(w, http.StatusOK, leaderboard)
}

// API Key management handlers

func (api *APIServer) createAPIKey(w http.ResponseWriter, r *http.Request) {
	discordID := r.Context().Value("discord_id").(string)
	
	var req struct {
		Name      string   `json:"name"`
		Scopes    []string `json:"scopes"`
		RateLimit int      `json:"rate_limit,omitempty"`
		TTLDays   *int     `json:"ttl_days,omitempty"`
	}
	
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		api.respondError(w, http.StatusBadRequest, "INVALID_REQUEST", "Invalid JSON")
		return
	}
	
	if req.Name == "" {
		api.respondError(w, http.StatusBadRequest, "VALIDATION_ERROR", "Name is required")
		return
	}
	
	var ttl *time.Duration
	if req.TTLDays != nil {
		d := time.Duration(*req.TTLDays) * 24 * time.Hour
		ttl = &d
	}
	
	apiKey, key, err := api.apiKeysSvc.GenerateAPIKey(r.Context(), discordID, req.Name, req.Scopes, req.RateLimit, ttl)
	if err != nil {
		api.respondError(w, http.StatusInternalServerError, "KEY_GENERATION_FAILED", err.Error())
		return
	}
	
	// Return the key once (won't be shown again)
	api.respondSuccess(w, http.StatusCreated, map[string]interface{}{
		"api_key":    apiKey,
		"id":         key.ID,
		"name":       key.Name,
		"scopes":     key.Scopes,
		"rate_limit": key.RateLimit,
		"created_at": key.CreatedAt,
		"expires_at": key.ExpiresAt,
		"warning":    "Store this key securely - it won't be shown again",
	})
}

func (api *APIServer) listAPIKeys(w http.ResponseWriter, r *http.Request) {
	discordID := r.Context().Value("discord_id").(string)
	
	keys, err := api.apiKeysSvc.ListAPIKeys(r.Context(), discordID)
	if err != nil {
		api.respondError(w, http.StatusInternalServerError, "DATABASE_ERROR", err.Error())
		return
	}
	
	// Don't expose key hashes
	response := make([]map[string]interface{}, len(keys))
	for i, key := range keys {
		response[i] = map[string]interface{}{
			"id":         key.ID,
			"name":       key.Name,
			"scopes":     key.Scopes,
			"rate_limit": key.RateLimit,
			"last_used":  key.LastUsed,
			"created_at": key.CreatedAt,
			"expires_at": key.ExpiresAt,
		}
	}
	
	api.respondSuccess(w, http.StatusOK, response)
}

func (api *APIServer) revokeAPIKey(w http.ResponseWriter, r *http.Request) {
	discordID := r.Context().Value("discord_id").(string)
	vars := mux.Vars(r)
	
	keyID, err := strconv.Atoi(vars["id"])
	if err != nil {
		api.respondError(w, http.StatusBadRequest, "INVALID_ID", "Invalid key ID")
		return
	}
	
	if err := api.apiKeysSvc.RevokeAPIKey(r.Context(), keyID, discordID); err != nil {
		if strings.Contains(err.Error(), "not found") {
			api.respondError(w, http.StatusNotFound, "NOT_FOUND", "API key not found")
		} else {
			api.respondError(w, http.StatusInternalServerError, "REVOKE_FAILED", err.Error())
		}
		return
	}
	
	api.respondSuccess(w, http.StatusOK, map[string]string{
		"message": "API key revoked successfully",
	})
}

// Helper functions

func (api *APIServer) respondSuccess(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(APIResponse{
		Success: true,
		Data:    data,
	})
}

func (api *APIServer) respondError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(APIResponse{
		Success: false,
		Error: &APIError{
			Code:    code,
			Message: message,
		},
	})
}

func getDefaultGameCost(gameType string) int {
	costs := map[string]int{
		"minecraft":  30,
		"terraria":   35,
		"dst":        60,
		"cs2":        120,
		"gmod":       95,
		"valheim":    120,
		"rust":       220,
		"ark":        240,
		"palworld":   180,
		"7d2d":       130,
		"pz":         135,
		"factorio":   100,
		"satisfactory": 240,
		"starbound":  40,
	}

	return costs[strings.ToLower(gameType)]
}
