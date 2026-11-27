package security

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// IPRateLimiter protege endpoints sensibles (login, registro, IA, escrituras sociales)
// frente a ataques de fuerza bruta y agotamiento de cuota por dirección IP.
type IPRateLimiter struct {
	mu          sync.Mutex
	window      time.Duration
	maxRequests int
	buckets     map[string]*ipWindow
	now         func() time.Time
}

type ipWindow struct {
	count   int
	resetAt time.Time
}

func NewIPRateLimiter(maxRequests int, window time.Duration) *IPRateLimiter {
	if maxRequests <= 0 {
		maxRequests = 60
	}
	if window <= 0 {
		window = time.Minute
	}
	return &IPRateLimiter{
		window:      window,
		maxRequests: maxRequests,
		buckets:     make(map[string]*ipWindow),
		now:         time.Now,
	}
}

func (l *IPRateLimiter) Allow(ip string) bool {
	if ip == "" {
		ip = "unknown"
	}
	now := l.now()
	l.mu.Lock()
	defer l.mu.Unlock()

	if len(l.buckets) > 4096 {
		for k, b := range l.buckets {
			if now.After(b.resetAt) {
				delete(l.buckets, k)
			}
		}
	}

	b, exists := l.buckets[ip]
	if !exists || now.After(b.resetAt) {
		l.buckets[ip] = &ipWindow{count: 1, resetAt: now.Add(l.window)}
		return true
	}
	if b.count >= l.maxRequests {
		return false
	}
	b.count++
	return true
}

func (l *IPRateLimiter) Wrap(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ip := ExtractClientIP(r)
		if !l.Allow(ip) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "60")
			w.WriteHeader(http.StatusTooManyRequests)
			_, _ = w.Write([]byte(`{"error":"rate limit exceeded, please retry shortly"}`))
			return
		}
		next(w, r)
	}
}

func ExtractClientIP(r *http.Request) string {
	if cf := strings.TrimSpace(r.Header.Get("CF-Connecting-IP")); cf != "" {
		return cf
	}
	if xri := strings.TrimSpace(r.Header.Get("X-Real-IP")); xri != "" {
		return xri
	}
	if xff := strings.TrimSpace(r.Header.Get("X-Forwarded-For")); xff != "" {
		parts := strings.Split(xff, ",")
		if first := strings.TrimSpace(parts[0]); first != "" {
			return first
		}
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil && host != "" {
		return host
	}
	return strings.TrimSpace(r.RemoteAddr)
}

// WithSecurityHeadersAndBodyLimit aplica cabeceras OWASP, límite de tamaño de body contra DoS
// y política CORS controlada en cada petición HTTP.
func WithSecurityHeadersAndBodyLimit(next http.Handler, instanceID, allowedOrigin string, maxBodyBytes int64) http.Handler {
	if maxBodyBytes <= 0 {
		maxBodyBytes = 2 << 20 // 2 MiB
	}
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-LiveBid-Instance", instanceID)
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("Permissions-Policy", "geolocation=(), microphone=(), camera=()")

		origin := strings.TrimSpace(allowedOrigin)
		if origin == "" {
			origin = "*"
		}
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, Idempotency-Key, Stripe-Signature, X-Web3-Wallet, X-Web3-Signature, X-Web3-Nonce, X-Web3-Deadline, X-AI-Agent-Mandate")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		if r.Body != nil && !strings.HasPrefix(r.URL.Path, "/ws/") {
			r.Body = http.MaxBytesReader(w, r.Body, maxBodyBytes)
		}
		next.ServeHTTP(w, r)
	})
}
