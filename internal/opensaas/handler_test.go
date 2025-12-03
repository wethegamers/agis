package opensaas

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

type mockUserService struct {
	users map[string]*User
}

func newMockUserService() *mockUserService {
	return &mockUserService{
		users: map[string]*User{
			"123456789": {
				ID:        1,
				DiscordID: "123456789",
				Username:  "testuser",
				Credits:   100,
				WTGCoins:  10,
				Tier:      "premium",
			},
		},
	}
}

func (m *mockUserService) GetUserByDiscordID(ctx context.Context, id string) (*User, error) {
	if u, ok := m.users[id]; ok {
		return u, nil
	}
	return nil, context.DeadlineExceeded
}

func (m *mockUserService) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	return nil, nil
}

func (m *mockUserService) LinkDiscordAccount(ctx context.Context, uid int, did string) error {
	return nil
}

func (m *mockUserService) GetUserCredits(ctx context.Context, uid int) (credits, wtg int, err error) {
	return 100, 10, nil
}

func (m *mockUserService) UpdateUserTier(ctx context.Context, uid int, tier string, exp *time.Time) error {
	return nil
}

type mockServerService struct{}

func (m *mockServerService) GetUserServers(ctx context.Context, uid int) ([]Server, error) {
	return []Server{}, nil
}

func (m *mockServerService) CreateServer(ctx context.Context, uid int, g, n string) (*Server, error) {
	return &Server{ID: 1, Name: n, GameType: g}, nil
}

func (m *mockServerService) DeleteServer(ctx context.Context, uid, sid int) error {
	return nil
}

func (m *mockServerService) ControlServer(ctx context.Context, uid, sid int, a string) error {
	return nil
}

func TestNewHandler(t *testing.T) {
	h := NewHandler(nil, nil, nil, slog.Default())
	if h == nil {
		t.Fatal("expected non-nil handler")
	}
}

func TestHandleHealth(t *testing.T) {
	h := NewHandler(nil, nil, nil, slog.Default())
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/opensaas/v1/health", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}

	var resp apiResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("failed to parse response: %v", err)
	}
	if !resp.Success {
		t.Error("expected success to be true")
	}
}

func TestHandleListGames(t *testing.T) {
	h := NewHandler(nil, nil, nil, slog.Default())
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/opensaas/v1/games", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestAuthMiddleware_NoHeader(t *testing.T) {
	h := NewHandler(newMockUserService(), nil, nil, slog.Default())
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/opensaas/v1/user/me", http.NoBody)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", rec.Code)
	}
}

func TestHandleGetCurrentUser(t *testing.T) {
	h := NewHandler(newMockUserService(), nil, nil, slog.Default())
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/opensaas/v1/user/me", http.NoBody)
	req.Header.Set("Authorization", "Bearer 123456789")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}

func TestHandleGetCurrentUser_NotFound(t *testing.T) {
	h := NewHandler(newMockUserService(), nil, nil, slog.Default())
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodGet, "/api/opensaas/v1/user/me", http.NoBody)
	req.Header.Set("Authorization", "Bearer unknown_user")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", rec.Code)
	}
}

func TestHandleCreateServer_ValidationError(t *testing.T) {
	h := NewHandler(newMockUserService(), nil, &mockServerService{}, slog.Default())
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/opensaas/v1/servers", strings.NewReader(`{}`))
	req.Header.Set("Authorization", "Bearer 123456789")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestHandleControlServer_InvalidAction(t *testing.T) {
	h := NewHandler(newMockUserService(), nil, &mockServerService{}, slog.Default())
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/opensaas/v1/servers/1/control", strings.NewReader(`{"action":"invalid"}`))
	req.Header.Set("Authorization", "Bearer 123456789")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", rec.Code)
	}
}

func TestHandleControlServer_ValidAction(t *testing.T) {
	h := NewHandler(newMockUserService(), nil, &mockServerService{}, slog.Default())
	mux := http.NewServeMux()
	h.RegisterRoutes(mux)

	req := httptest.NewRequest(http.MethodPost, "/api/opensaas/v1/servers/1/control", strings.NewReader(`{"action":"start"}`))
	req.Header.Set("Authorization", "Bearer 123456789")
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", rec.Code)
	}
}
