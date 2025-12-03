package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestRequestID(t *testing.T) {
	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-ID")
		if reqID == "" {
			t.Error("expected request ID in header")
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}

	reqID := rec.Header().Get("X-Request-ID")
	if reqID == "" {
		t.Error("expected X-Request-ID header in response")
	}
}

func TestRequestIDExisting(t *testing.T) {
	existingID := "existing-request-id"

	handler := RequestID(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqID := r.Header.Get("X-Request-ID")
		if reqID != existingID {
			t.Errorf("expected request ID %s, got %s", existingID, reqID)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	req.Header.Set("X-Request-ID", existingID)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Errorf("expected status 200, got %d", rec.Code)
	}
}

func TestRecover(t *testing.T) {
	handler := Recover(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		panic("test panic")
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected status 500, got %d", rec.Code)
	}
}

func TestRateLimiter(t *testing.T) {
	rl := NewRateLimiter(2, 2)
	handler := rl.Handler(func(r *http.Request) string {
		return "test-key"
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("request %d: expected status 200, got %d", i+1, rec.Code)
		}
	}

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("expected status 429, got %d", rec.Code)
	}
}

func TestRateLimiterRefill(t *testing.T) {
	rl := NewRateLimiter(10, 1)
	handler := rl.Handler(func(r *http.Request) string {
		return "test-key"
	})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("first request: expected 200, got %d", rec.Code)
	}

	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusTooManyRequests {
		t.Errorf("second request: expected 429, got %d", rec.Code)
	}

	time.Sleep(150 * time.Millisecond)

	req = httptest.NewRequest(http.MethodGet, "/test", nil)
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Errorf("third request after refill: expected 200, got %d", rec.Code)
	}
}

func TestCORS(t *testing.T) {
	handler := CORS([]string{"https://example.com"})(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	t.Run("preflight request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodOptions, "/test", nil)
		req.Header.Set("Origin", "https://example.com")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Header().Get("Access-Control-Allow-Origin") != "https://example.com" {
			t.Errorf("expected Allow-Origin header")
		}
		if rec.Code != http.StatusNoContent {
			t.Errorf("expected 204, got %d", rec.Code)
		}
	})

	t.Run("actual request", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set("Origin", "https://example.com")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Header().Get("Access-Control-Allow-Origin") != "https://example.com" {
			t.Errorf("expected Allow-Origin header")
		}
		if rec.Code != http.StatusOK {
			t.Errorf("expected 200, got %d", rec.Code)
		}
	})
}

func TestChain(t *testing.T) {
	var order []string

	m1 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "m1-before")
			next.ServeHTTP(w, r)
			order = append(order, "m1-after")
		})
	}

	m2 := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			order = append(order, "m2-before")
			next.ServeHTTP(w, r)
			order = append(order, "m2-after")
		})
	}

	handler := Chain(m1, m2)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		order = append(order, "handler")
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	expected := []string{"m1-before", "m2-before", "handler", "m2-after", "m1-after"}
	if len(order) != len(expected) {
		t.Fatalf("expected %d calls, got %d: %v", len(expected), len(order), order)
	}
	for i, v := range expected {
		if order[i] != v {
			t.Errorf("position %d: expected %s, got %s", i, v, order[i])
		}
	}
}

func TestIPKeyFunc(t *testing.T) {
	tests := []struct {
		name     string
		headers  map[string]string
		remote   string
		expected string
	}{
		{
			name:     "X-Forwarded-For",
			headers:  map[string]string{"X-Forwarded-For": "1.2.3.4"},
			expected: "1.2.3.4",
		},
		{
			name:     "X-Real-IP",
			headers:  map[string]string{"X-Real-IP": "5.6.7.8"},
			expected: "5.6.7.8",
		},
		{
			name:     "RemoteAddr",
			headers:  map[string]string{},
			remote:   "9.10.11.12:1234",
			expected: "9.10.11.12:1234",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			if tt.remote != "" {
				req.RemoteAddr = tt.remote
			}

			result := IPKeyFunc(req)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestAPIVersioning(t *testing.T) {
	av := NewAPIVersioning(APIVersionV1, []APIVersion{APIVersionV1, APIVersionV2})

	handler := av.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		version := GetAPIVersion(r.Context())
		w.Write([]byte(string(version)))
	}))

	t.Run("default version", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		if rec.Body.String() != "v1" {
			t.Errorf("expected v1, got %s", rec.Body.String())
		}
		if rec.Header().Get(APIVersionHeader) != "v1" {
			t.Errorf("expected X-API-Version header v1")
		}
	})

	t.Run("version from header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set(APIVersionHeader, "v2")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		if rec.Body.String() != "v2" {
			t.Errorf("expected v2, got %s", rec.Body.String())
		}
	})

	t.Run("version from URL path /api/v1/", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v1/users", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		if rec.Body.String() != "v1" {
			t.Errorf("expected v1, got %s", rec.Body.String())
		}
	})

	t.Run("version from URL path /api/v2/", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/v2/users", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		if rec.Body.String() != "v2" {
			t.Errorf("expected v2, got %s", rec.Body.String())
		}
	})

	t.Run("unsupported version", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set(APIVersionHeader, "v3")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusBadRequest {
			t.Errorf("expected status 400, got %d", rec.Code)
		}
	})
}

func TestAPIVersioningDeprecation(t *testing.T) {
	av := NewAPIVersioning(APIVersionV2, []APIVersion{APIVersionV1, APIVersionV2})
	av.Deprecate(APIVersionV1, "v1 will be removed in 2025")

	handler := av.Handler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	t.Run("deprecated version", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set(APIVersionHeader, "v1")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		if rec.Header().Get("X-API-Deprecated") != "true" {
			t.Error("expected X-API-Deprecated header")
		}
		if rec.Header().Get("X-API-Deprecation-Message") == "" {
			t.Error("expected deprecation message header")
		}
	})

	t.Run("non-deprecated version", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		req.Header.Set(APIVersionHeader, "v2")
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Header().Get("X-API-Deprecated") != "" {
			t.Error("did not expect deprecation header for v2")
		}
	})
}

func TestSecurityHeaders(t *testing.T) {
	handler := SecurityHeaders(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	expectedHeaders := map[string]string{
		"X-Content-Type-Options": "nosniff",
		"X-Frame-Options":        "DENY",
		"X-XSS-Protection":       "1; mode=block",
		"Referrer-Policy":        "strict-origin-when-cross-origin",
	}

	for header, expected := range expectedHeaders {
		if rec.Header().Get(header) != expected {
			t.Errorf("header %s: expected %s, got %s", header, expected, rec.Header().Get(header))
		}
	}
}

func TestTimeout(t *testing.T) {
	t.Run("request completes in time", func(t *testing.T) {
		handler := Timeout(100 * time.Millisecond)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
	})

	// Note: Timeout middleware is tricky to test with httptest due to goroutine behavior
	// In production, use with actual HTTP server
}
