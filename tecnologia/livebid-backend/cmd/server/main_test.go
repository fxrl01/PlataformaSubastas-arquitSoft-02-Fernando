package main

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestServerHelpersCORSStaticCacheAndEnv(t *testing.T) {
	t.Setenv("INSTANCE_ID", "livebid-test-1")
	t.Setenv("TEST_INT_VALID", "42")
	t.Setenv("TEST_INT_BAD", "not-an-int")
	t.Setenv("TEST_STR_VAL", "custom")

	if got := redisAddr(""); got != "localhost:6379" {
		t.Fatalf("expected localhost:6379, got %s", got)
	}
	if got := redisAddr("redis://redis:6379"); got != "redis:6379" {
		t.Fatalf("expected redis:6379, got %s", got)
	}
	if got := redisAddr("127.0.0.1:6379"); got != "127.0.0.1:6379" {
		t.Fatalf("expected 127.0.0.1:6379, got %s", got)
	}
	if got := envInt("TEST_INT_VALID", 10); got != 42 {
		t.Fatalf("expected 42, got %d", got)
	}
	if got := envInt("TEST_INT_BAD", 10); got != 10 {
		t.Fatalf("expected fallback 10, got %d", got)
	}
	if got := envInt("TEST_INT_MISSING", 10); got != 10 {
		t.Fatalf("expected fallback 10, got %d", got)
	}
	if got := envStr("TEST_STR_VAL", "fb"); got != "custom" {
		t.Fatalf("expected custom, got %s", got)
	}
	if got := envStr("TEST_STR_MISSING", "fb"); got != "fb" {
		t.Fatalf("expected fb, got %s", got)
	}

	called := false
	corsHandler := withCORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	}))
	optRec := httptest.NewRecorder()
	corsHandler.ServeHTTP(optRec, httptest.NewRequest(http.MethodOptions, "/api/v1/subastas", nil))
	if optRec.Code != http.StatusNoContent || called {
		t.Fatalf("expected 204 preflight without calling next, got %d called=%v", optRec.Code, called)
	}
	getRec := httptest.NewRecorder()
	corsHandler.ServeHTTP(getRec, httptest.NewRequest(http.MethodGet, "/health", nil))
	if getRec.Code != http.StatusOK || getRec.Header().Get("X-LiveBid-Instance") != "livebid-test-1" {
		t.Fatalf("expected X-LiveBid-Instance header, got %v", getRec.Header())
	}

	staticH := withStaticCache(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusOK) }))
	idxRec := httptest.NewRecorder()
	staticH.ServeHTTP(idxRec, httptest.NewRequest(http.MethodGet, "/index.html", nil))
	if idxRec.Header().Get("Cache-Control") != "no-cache, must-revalidate" {
		t.Fatalf("unexpected Cache-Control for index.html: %s", idxRec.Header().Get("Cache-Control"))
	}
	cssRec := httptest.NewRecorder()
	staticH.ServeHTTP(cssRec, httptest.NewRequest(http.MethodGet, "/styles.css", nil))
	if cssRec.Header().Get("Cache-Control") != "public, max-age=3600" {
		t.Fatalf("unexpected Cache-Control for styles.css: %s", cssRec.Header().Get("Cache-Control"))
	}
	_ = resolvePublicDir()
}
