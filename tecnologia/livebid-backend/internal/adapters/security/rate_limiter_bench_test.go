package security_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/fxrl01/livebid-backend/internal/adapters/security"
)

func TestExtractClientIPPrecedenceAndPreflightOptions(t *testing.T) {
	r1 := httptest.NewRequest(http.MethodGet, "/health", nil)
	r1.Header.Set("CF-Connecting-IP", "198.51.100.1")
	r1.Header.Set("X-Real-IP", "198.51.100.2")
	r1.Header.Set("X-Forwarded-For", "198.51.100.3, 10.0.0.1")
	if got := security.ExtractClientIP(r1); got != "198.51.100.1" {
		t.Fatalf("expected CF-Connecting-IP precedence, got %q", got)
	}

	r2 := httptest.NewRequest(http.MethodGet, "/health", nil)
	r2.Header.Set("X-Forwarded-For", "203.0.113.55, 10.0.0.2")
	if got := security.ExtractClientIP(r2); got != "203.0.113.55" {
		t.Fatalf("expected first X-Forwarded-For IP, got %q", got)
	}

	r3 := httptest.NewRequest(http.MethodGet, "/health", nil)
	r3.RemoteAddr = "192.0.2.44:54321"
	if got := security.ExtractClientIP(r3); got != "192.0.2.44" {
		t.Fatalf("expected host split from RemoteAddr, got %q", got)
	}

	// Verificar respuesta OPTIONS 204 sin invocar el handler interno
	called := false
	mw := security.WithSecurityHeadersAndBodyLimit(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}), "api-1", "", 0)

	optReq := httptest.NewRequest(http.MethodOptions, "/api/v1/subastas", nil)
	optRec := httptest.NewRecorder()
	mw.ServeHTTP(optRec, optReq)
	if optRec.Code != http.StatusNoContent || called {
		t.Fatalf("expected 204 NoContent on OPTIONS without calling inner handler, got %d (called=%t)", optRec.Code, called)
	}
}

func TestShardedIPRateLimiterConcurrentIsolation(t *testing.T) {
	limiter := security.NewIPRateLimiter(5, time.Minute)
	if !limiter.Allow("") {
		t.Fatal("expected empty IP fallback bucket to allow initial call")
	}

	var allowedCount atomic.Int64
	const distinctIPs = 64
	done := make(chan struct{}, distinctIPs)

	for i := 0; i < distinctIPs; i++ {
		go func(idx int) {
			ip := fmt.Sprintf("198.51.%d.%d", idx/16, idx%16+1)
			for j := 0; j < 8; j++ {
				if limiter.Allow(ip) {
					allowedCount.Add(1)
				}
			}
			done <- struct{}{}
		}(i)
	}
	for i := 0; i < distinctIPs; i++ {
		<-done
	}

	expected := int64(distinctIPs * 5)
	if allowedCount.Load() != expected {
		t.Fatalf("expected exact %d allowed requests across %d concurrent IPs, got %d", expected, distinctIPs, allowedCount.Load())
	}
}

func BenchmarkIPRateLimiterParallel(b *testing.B) {
	limiter := security.NewIPRateLimiter(1_000_000_000, time.Hour)
	ips := make([]string, 64)
	for i := range ips {
		ips[i] = fmt.Sprintf("203.0.113.%d", i+1)
		limiter.Allow(ips[i]) // Precalentar shards
	}

	var seq atomic.Uint64
	b.ResetTimer()
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			idx := seq.Add(1) & 63
			limiter.Allow(ips[idx])
		}
	})
}
