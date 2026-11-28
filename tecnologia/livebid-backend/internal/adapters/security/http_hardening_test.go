package security_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/fxrl01/livebid-backend/internal/adapters/security"
)

func TestIPRateLimiterBlocksBurstAndResetsAfterWindow(t *testing.T) {
	limiter := security.NewIPRateLimiter(2, 50*time.Millisecond)
	handler := limiter.Wrap(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	for i := 0; i < 2; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
		req.Header.Set("X-Real-IP", "203.0.113.10")
		rec := httptest.NewRecorder()
		handler(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("attempt %d expected 200, got %d", i+1, rec.Code)
		}
	}

	reqBlocked := httptest.NewRequest(http.MethodPost, "/api/v1/auth/login", nil)
	reqBlocked.Header.Set("X-Real-IP", "203.0.113.10")
	recBlocked := httptest.NewRecorder()
	handler(recBlocked, reqBlocked)
	if recBlocked.Code != http.StatusTooManyRequests {
		t.Fatalf("expected 429 on 3rd burst request, got %d", recBlocked.Code)
	}

	time.Sleep(65 * time.Millisecond)
	recAfter := httptest.NewRecorder()
	handler(recAfter, reqBlocked)
	if recAfter.Code != http.StatusOK {
		t.Fatalf("expected 200 after window reset, got %d", recAfter.Code)
	}
}

func TestSecurityHeadersAndBodySizeCap(t *testing.T) {
	mw := security.WithSecurityHeadersAndBodyLimit(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "payload too large", http.StatusRequestEntityTooLarge)
			return
		}
		w.WriteHeader(http.StatusOK)
	}), "api-test", "https://subastas.ejemplo.pe", 32)

	small := httptest.NewRequest(http.MethodPost, "/api/v1/test", bytes.NewBufferString(`{"ok":true}`))
	recSmall := httptest.NewRecorder()
	mw.ServeHTTP(recSmall, small)
	if recSmall.Code != http.StatusOK {
		t.Fatalf("expected 200 on small payload, got %d", recSmall.Code)
	}
	if recSmall.Header().Get("X-Content-Type-Options") != "nosniff" || recSmall.Header().Get("X-Frame-Options") != "DENY" {
		t.Fatalf("missing OWASP security headers: %+v", recSmall.Header())
	}
	if recSmall.Header().Get("Access-Control-Allow-Origin") != "https://subastas.ejemplo.pe" {
		t.Fatalf("unexpected CORS origin: %q", recSmall.Header().Get("Access-Control-Allow-Origin"))
	}

	large := httptest.NewRequest(http.MethodPost, "/api/v1/test", bytes.NewBuffer(bytes.Repeat([]byte("A"), 128)))
	recLarge := httptest.NewRecorder()
	mw.ServeHTTP(recLarge, large)
	if recLarge.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected 413 on oversized payload, got %d", recLarge.Code)
	}
}
