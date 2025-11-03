package http

import (
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"agis-bot/internal/version"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// Server represents the HTTP server for metrics and health checks
type Server struct {
	server *http.Server
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
			"/health":  "Health check endpoint",
			"/ready":   "Readiness check endpoint",
			"/info":    "Service information and build details",
			"/version": "Version information only",
			"/metrics": "Prometheus metrics",
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

// Minimal ads landing page (HTML)
func adsPageHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	uid := r.URL.Query().Get("user")
	if uid == "" {
		uid = r.URL.Query().Get("uid")
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
		ol = "<a href=\"" + offerwallURL + "?externalIdentifier=" + uid + "\">Open Offerwall</a>"
	}
	if surveywallURL != "" && uid != "" {
		sl = "<a href=\"" + surveywallURL + "?externalIdentifier=" + uid + "\">Open Surveywall</a>"
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
		},
	})
}
