package http

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"

	"agis-bot/internal/version"

	"github.com/bwmarrin/discordgo"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Server represents the HTTP server for metrics and health checks
type Server struct {
	server *http.Server
}

// DashboardServer represents a server item returned to the WordPress dashboard
type DashboardServer struct {
	ID         interface{} `json:"id,omitempty"`
	Name       string      `json:"name"`
	Game       string      `json:"game,omitempty"`
	Address    string      `json:"address,omitempty"`
	Port       int         `json:"port,omitempty"`
	Status     string      `json:"status,omitempty"`
	Players    PlayersInfo `json:"players,omitempty"`
	Region     string      `json:"region,omitempty"`
	CreatedAt  string      `json:"created_at,omitempty"`
	ConnectURL string      `json:"connect_url,omitempty"`
	ManageURL  string      `json:"manage_url,omitempty"`
}

// PlayersInfo represents current/max players
type PlayersInfo struct {
	Current int `json:"current,omitempty"`
	Max     int `json:"max,omitempty"`
}

// Integration hooks and config (set from main)
var (
	// OnRewardWithConversion is called when a valid ad callback is received (idempotent handler).
	OnRewardWithConversion func(uid string, amount int, conversionID, source string) error
	// Backward-compatible simple reward callback (used if OnRewardWithConversion is nil)
	OnAyetReward func(uid string, amount int) error
	// token shared with ayet postback to authenticate callbacks (simple shared-secret)
	adsCallbackToken string
	// api key used to verify HMAC signatures from ayet-studios
	adsAPIKey string
	// links for ads landing (offerwall/survey/video)
	offerwallURL     string
	surveywallURL    string
	videoPlacementID string

	// Discord session and verification API config
	discordSession *discordgo.Session
	loggingService interface {
		LogAudit(userID, action, message string, details map[string]interface{})
	}
	verifyAPISecret string
	verifyGuildID   string
	verifiedRoleID  string

	// Provider for user servers exposed to WordPress dashboard
	userServersProvider func(ctx context.Context, discordID string) ([]DashboardServer, error)
)

// SetAdsCallbackToken sets the shared callback token for ad callbacks
func SetAdsCallbackToken(token string) { adsCallbackToken = token }

// SetAdsAPIKey sets the API key for signature verification
func SetAdsAPIKey(key string) { adsAPIKey = key }

// SetAdsLinks sets offerwall/survey links and video placement id for /ads page
func SetAdsLinks(offerwall, survey, videoID string) {
	offerwallURL = offerwall
	surveywallURL = survey
	videoPlacementID = videoID
}

// NewServer creates a new HTTP server
func NewServer() *Server {
	mux := http.NewServeMux()

	// Health endpoint
	mux.HandleFunc("/health", healthHandler)
	mux.HandleFunc("/healthz", healthHandler) // K8s standard alias

	// Readiness endpoint
	mux.HandleFunc("/ready", readinessHandler)
	mux.HandleFunc("/readyz", readinessHandler) // K8s standard alias

	// Info/About endpoint
	mux.HandleFunc("/info", infoHandler)
	mux.HandleFunc("/about", infoHandler) // Alias

	// Version endpoint
	mux.HandleFunc("/version", versionHandler)

	// Metrics endpoint (Prometheus metrics)
	mux.Handle("/metrics", promhttp.Handler())

	// Ad callback (ayet-studios postback)
	mux.HandleFunc("/ads/ayet/callback", ayetCallbackHandler)

	// Verification API
	mux.HandleFunc("/api/verify-user", verifyUserHandler)

	// User servers API for WordPress dashboard
	mux.HandleFunc("/api/user-servers", userServersHandler)

	// Ads landing page
	mux.HandleFunc("/ads", adsPageHandler)

	// ads.txt at domain root per ayeT requirement
	mux.HandleFunc("/ads.txt", adsTxtHandler)

	// Root endpoint
	mux.HandleFunc("/", rootHandler)

	server := &http.Server{
		Addr:         ":9090", // Prometheus standard port
		Handler:      mux,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return &Server{server: server}
}

// Start starts the HTTP server
func (s *Server) Start() error {
	log.Printf("🌐 Starting HTTP server on port 9090 (Prometheus standard)")
	return s.server.ListenAndServe()
}

// Stop gracefully stops the HTTP server
func (s *Server) Stop(ctx context.Context) error {
	log.Println("🛑 Shutting down HTTP server...")
	return s.server.Shutdown(ctx)
}

// Health check handler
func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Format(time.RFC3339),
		"service":   "agis-bot",
	})
}

// Readiness check handler
func readinessHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":    "ready",
		"timestamp": time.Now().Format(time.RFC3339),
		"service":   "agis-bot",
	})
}

// Info/About handler
func infoHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	buildInfo := version.GetBuildInfo()
	response := map[string]interface{}{
		"service":     "agis-bot",
		"description": "WTG Agones GameServer Management Bot",
		"build":       buildInfo,
		"endpoints": map[string]string{
			"/health":            "Health check endpoint",
			"/ready":             "Readiness check endpoint",
			"/info":              "Service information and build details",
			"/version":           "Version information only",
			"/metrics":           "Prometheus metrics",
			"/api/verify-user":   "Assign Verified role to a Discord user (POST)",
			"/api/user-servers":  "List the current user's servers (GET; header X-WTG-Secret required)",
		},
	}

	json.NewEncoder(w).Encode(response)
}

// Version endpoint
func versionHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(version.GetBuildInfo())
}

// ads.txt for ayeT-Studios (served at domain root)
func adsTxtHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(`# AYET-STUDIOS
ayetstudios.com, AYETSTUDIOS, DIRECT
ayetstudios.com, PL-20742, DIRECT
`))
}

// Ad callback handler for ayet-studios
// Expected query params (example):
//
//	uid=<discord_id>&amount=<credits>&event=video_complete&token=<shared_token>
func ayetCallbackHandler(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	// Accept either token auth or signature auth
	uid := firstNonEmpty(q.Get("externalIdentifier"), q.Get("uid"))
	amountStr := firstNonEmpty(q.Get("currency"), q.Get("amount"))
	conversionID := firstNonEmpty(q.Get("conversionId"), q.Get("tx"))
	signature := q.Get("signature")

	// Validate presence
	if uid == "" || amountStr == "" || conversionID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "bad_request"})
		return
	}
	// Validate signature if provided, otherwise fall back to shared token
	if signature != "" && adsAPIKey != "" {
		if !verifyAyetSignature(adsAPIKey, uid, amountStr, conversionID,
			q.Get("custom_1"), q.Get("custom_2"), q.Get("custom_3"), q.Get("custom_4"), signature) {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "invalid_signature"})
			return
		}
	} else {
		token := q.Get("token")
		if adsCallbackToken != "" && token != adsCallbackToken {
			w.WriteHeader(http.StatusUnauthorized)
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "unauthorized"})
			return
		}
	}
	amt, err := strconv.Atoi(amountStr)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"status": "invalid_amount"})
		return
	}
	// Idempotent reward using conversion ID
	if OnRewardWithConversion != nil {
		if err := OnRewardWithConversion(uid, amt, conversionID, "ayet"); err != nil {
			log.Printf("ayet reward error: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "error"})
			return
		}
	} else if OnAyetReward != nil {
		if err := OnAyetReward(uid, amt); err != nil {
			log.Printf("ayet reward error: %v", err)
			w.WriteHeader(http.StatusInternalServerError)
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "error"})
			return
		}
	}
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]string{"status": "ok"})
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

// HMAC-SHA1 signature verification for ayet-studios
// message = externalIdentifier + currency + conversionId + custom_1 + custom_2 + custom_3 + custom_4 (empty strings if missing)
func verifyAyetSignature(apiKey, externalIdentifier, currency, conversionID, c1, c2, c3, c4, sig string) bool {
	msg := externalIdentifier + currency + conversionID + c1 + c2 + c3 + c4
	h := hmac.New(sha1.New, []byte(apiKey))
	h.Write([]byte(msg))
	expected := hex.EncodeToString(h.Sum(nil))
	return hmac.Equal([]byte(expected), []byte(sig))
}

// SetDiscordSessionForAPI wires the Discord session for API handlers
func SetDiscordSessionForAPI(s *discordgo.Session) { discordSession = s }

// SetLoggingServiceForAPI wires the logging service for API handlers
func SetLoggingServiceForAPI(ls interface {
	LogAudit(userID, action, message string, details map[string]interface{})
}) {
	loggingService = ls
}

// SetVerifyAPI configures the verification API
func SetVerifyAPI(secret, guildID, roleID string) {
	verifyAPISecret = secret
	verifyGuildID = guildID
	verifiedRoleID = roleID
}

// SetUserServersProvider wires a provider used by /api/user-servers to fetch data
func SetUserServersProvider(f func(ctx context.Context, discordID string) ([]DashboardServer, error)) {
	userServersProvider = f
}

// verifyUserHandler handles POST /api/verify-user to assign the Verified role
// Expects JSON body: {"discord_id": "123...", "username": "optional"}
// Expects header: X-WTG-Secret: <shared_secret>
func verifyUserHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "method_not_allowed"})
		return
	}
	
	// Check if API is configured
	if discordSession == nil || verifyGuildID == "" || verifiedRoleID == "" || verifyAPISecret == "" {
		log.Println("verify-user: API not configured")
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_configured"})
		return
	}
	
	// Verify secret from header
	providedSecret := r.Header.Get("X-WTG-Secret")
	if providedSecret == "" {
		log.Println("verify-user: missing X-WTG-Secret header")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "missing_secret"})
		return
	}
	
	if subtle.ConstantTimeCompare([]byte(providedSecret), []byte(verifyAPISecret)) != 1 {
		log.Println("verify-user: invalid secret")
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}
	
	// Parse request body
	var payload struct {
		DiscordID string `json:"discord_id"`
		Username  string `json:"username"` // optional
	}
	
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		log.Printf("verify-user: invalid JSON: %v", err)
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "invalid_json"})
		return
	}
	
	// Validate required fields
	if payload.DiscordID == "" {
		log.Println("verify-user: missing discord_id")
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "missing_discord_id"})
		return
	}
	
	log.Printf("verify-user: processing request for Discord ID: %s (username: %s)", payload.DiscordID, payload.Username)
	
	// Ensure member exists in the guild
	member, err := discordSession.GuildMember(verifyGuildID, payload.DiscordID)
	if err != nil || member == nil || member.User == nil {
		log.Printf("verify-user: member not found: %v", err)
		w.WriteHeader(http.StatusNotFound)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "member_not_found"})
		return
	}
	
	// Check if member already has the verified role
	for _, rID := range member.Roles {
		if strings.EqualFold(rID, verifiedRoleID) {
			log.Printf("verify-user: user %s already has verified role", payload.DiscordID)
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"message": "already_verified",
			})
			return
		}
	}
	
	// Add the verified role
	if err := discordSession.GuildMemberRoleAdd(verifyGuildID, payload.DiscordID, verifiedRoleID); err != nil {
		log.Printf("verify-user: failed to add role: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "failed_to_add_role"})
		return
	}
	
	log.Printf("verify-user: successfully verified user %s (%s)", payload.DiscordID, payload.Username)
	
	// Log to audit channel
	if loggingService != nil {
		userTag := payload.Username
		if userTag == "" && member.User != nil {
			userTag = fmt.Sprintf("%s#%s", member.User.Username, member.User.Discriminator)
		}
		loggingService.LogAudit(
			payload.DiscordID,
			"user_verified",
			fmt.Sprintf("✅ User %s has been verified via API", userTag),
			map[string]interface{}{
				"user_id":  payload.DiscordID,
				"username": userTag,
				"source":   "wordpress_api",
				"action":   "verified_role_assigned",
			},
		)
	}
	
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"success": true,
		"message": "role_assigned",
	})
}

// userServersHandler handles GET /api/user-servers to list a user's servers for the dashboard
func userServersHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodGet {
		w.WriteHeader(http.StatusMethodNotAllowed)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "method_not_allowed"})
		return
	}

	if verifyAPISecret == "" || userServersProvider == nil {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "not_configured"})
		return
	}

	providedSecret := r.Header.Get("X-WTG-Secret")
	if providedSecret == "" || subtle.ConstantTimeCompare([]byte(providedSecret), []byte(verifyAPISecret)) != 1 {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "unauthorized"})
		return
	}

	discordID := r.URL.Query().Get("discord_id")
	if discordID == "" {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "missing_discord_id"})
		return
	}

	servers, err := userServersProvider(r.Context(), discordID)
	if err != nil {
		log.Printf("user-servers: provider error: %v", err)
		w.WriteHeader(http.StatusInternalServerError)
		_ = json.NewEncoder(w).Encode(map[string]string{"error": "provider_error"})
		return
	}

	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{"success": true, "data": servers})
}

// Minimal ads landing page (HTML)
func adsPageHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	uid := r.URL.Query().Get("user")
	if uid == "" {
		uid = r.URL.Query().Get("uid")
	}
	if uid == "" {
		w.WriteHeader(http.StatusBadRequest)
		_, _ = w.Write([]byte("missing user id (use ?user=<discord_id>)"))
		return
	}
	tpl := `<html><head><meta name="viewport" content="width=device-width, initial-scale=1">
<title>Earn Credits</title></head><body>
<h2>Earn Game Credits</h2>
<p>User: %s</p>
<ul>
<li>Offerwall: %s</li>
<li>Surveywall: %s</li>
<li>Rewarded Video: %s</li>
</ul>
<small>Credits are awarded automatically after completion. If not, they will post within a few minutes.</small>
</body></html>`
	ol := "(not configured)"
	sl := "(not configured)"
	vl := "(not configured)"
	if offerwallURL != "" && uid != "" {
		ow := offerwallURL
		if strings.Contains(ow, "{YOUR_USER_IDENTIFIER}") {
			ow = strings.ReplaceAll(ow, "{YOUR_USER_IDENTIFIER}", uid)
		}
		sep := "?"
		if strings.Contains(ow, "?") {
			sep = "&"
		}
		ol = "<a target=\"_blank\" rel=\"noopener\" href=\"" + ow + sep + "externalIdentifier=" + uid + "\">Open Offerwall</a>"
	}
	if surveywallURL != "" && uid != "" {
		sw := surveywallURL
		if strings.Contains(sw, "{YOUR_USER_IDENTIFIER}") {
			sw = strings.ReplaceAll(sw, "{YOUR_USER_IDENTIFIER}", uid)
		}
		sep := "?"
		if strings.Contains(sw, "?") {
			sep = "&"
		}
		sl = "<a target=\"_blank\" rel=\"noopener\" href=\"" + sw + sep + "externalIdentifier=" + uid + "\">Open Surveywall</a>"
	}
	if videoPlacementID != "" && uid != "" {
		vl = "<a href=\"#\" onclick=\"alert('Integrate video SDK on your web app using placement ` + videoPlacementID + `');return false;\">Play Rewarded Video</a>"
	}
	_, _ = w.Write([]byte(fmt.Sprintf(tpl, uid, ol, sl, vl)))
}

// Root handler
func rootHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"service": "agis-bot",
		"status":  "running",
		"endpoints": []string{
			"/health", "/healthz",
			"/ready", "/readyz",
			"/info", "/about",
			"/version",
			"/metrics",
			"/api/verify-user",
			"/api/user-servers",
		},
	})
}
