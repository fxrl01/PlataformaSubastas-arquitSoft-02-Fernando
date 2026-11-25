package ai_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/fxrl01/livebid-backend/internal/adapters/ai"
	"github.com/fxrl01/livebid-backend/internal/core/domain"
)

type failingEvaluator struct{}

func (failingEvaluator) EvaluarRiesgo(context.Context, string, string, float64) (domain.NivelRiesgo, error) {
	return domain.RiesgoMedio, errors.New("ai down")
}
func (failingEvaluator) EvaluarPostor(context.Context, string, string) (domain.EvaluacionRiesgo, error) {
	return domain.EvaluacionRiesgo{}, errors.New("ai down")
}

func TestMultiModelRouterAndGeminiEvaluarRiesgoPaths(t *testing.T) {
	gemini := ai.NewGeminiRiskAdapter("")
	if riesgo, err := gemini.EvaluarRiesgo(context.Background(), "USR-VIP-01", "sub-1", 1000); err != nil || riesgo != domain.RiesgoBajo {
		t.Fatalf("expected RiesgoBajo for VIP user, got %v err=%v", riesgo, err)
	}
	if riesgo, err := gemini.EvaluarRiesgo(context.Background(), "USR-RIESGO_ALTO", "sub-1", 1000); err != nil || riesgo != domain.RiesgoAlto {
		t.Fatalf("expected RiesgoAlto for high risk user, got %v err=%v", riesgo, err)
	}

	breaker := ai.NewCircuitBreakerIA(gemini, 2, time.Second, 150*time.Millisecond)
	if riesgo, err := breaker.EvaluarRiesgo(context.Background(), "USR-VIP-01", "sub-1", 1000); err != nil || riesgo != domain.RiesgoBajo {
		t.Fatalf("breaker EvaluarRiesgo failed: %v err=%v", riesgo, err)
	}
	defaultBreaker := ai.NewCircuitBreakerIA(failingEvaluator{}, 0, 0, 0)
	if riesgo, err := defaultBreaker.EvaluarRiesgo(context.Background(), "u1", "s1", 100); err != nil || riesgo != domain.RiesgoMedio {
		t.Fatalf("expected fallback RiesgoMedio on failing provider, got %v err=%v", riesgo, err)
	}

	// Defaults & empty model registration + flagged bidder invalidation
	defaultRouter := ai.NewMultiModelRouter("", nil, 0)
	if eval, err := defaultRouter.EvaluarPostor(context.Background(), "u1", "s1"); err != nil || !eval.ModoFallback {
		t.Fatalf("expected fallback when no evaluator registered, got %+v err=%v", eval, err)
	}
	defaultRouter.RegisterModel("", nil)
	defaultRouter.RegisterModel("failing", failingEvaluator{})
	_ = defaultRouter.SetActiveModel("failing")
	if eval, err := defaultRouter.EvaluarPostor(context.Background(), "u1", "s1"); err != nil || !eval.ModoFallback {
		t.Fatalf("expected fallback when router evaluator fails, got %+v err=%v", eval, err)
	}

	router := ai.NewMultiModelRouter("gemini-2.5-flash", breaker, time.Minute)
	router.RegisterModel("gemini-2.5-pro-forensic", breaker)
	models := router.AvailableModels()
	if len(models) < 2 {
		t.Fatalf("expected at least 2 models, got %v", models)
	}
	if err := router.SetActiveModel("nonexistent-model"); err == nil {
		t.Fatal("expected error switching to unregistered model")
	}

	if riesgo, err := router.EvaluarRiesgo(context.Background(), "USR-VIP-01", "sub-1", 500); err != nil || riesgo != domain.RiesgoBajo {
		t.Fatalf("router EvaluarRiesgo failed: %v err=%v", riesgo, err)
	}
	if eval, err := router.EvaluarPostor(context.Background(), "USR-VIP-01", "sub-1"); err != nil || eval.Score == 0 {
		t.Fatalf("router EvaluarPostor failed: %+v err=%v", eval, err)
	}
	// Cache hit
	if eval, err := router.EvaluarPostor(context.Background(), "USR-VIP-01", "sub-1"); err != nil || eval.Score == 0 {
		t.Fatalf("router cached EvaluarPostor failed: %+v err=%v", eval, err)
	}
	// Flagged bidder bypasses cache and marks RiesgoAlto
	router.FlagSuspiciousBidder("USR-VIP-01", "Ping-pong")
	if eval, err := router.EvaluarPostor(context.Background(), "USR-VIP-01", "sub-1"); err != nil || eval.Nivel != domain.RiesgoAlto {
		t.Fatalf("expected RiesgoAlto for flagged bidder, got %+v err=%v", eval, err)
	}

	detector := ai.NewAsyncFraudDetector(router)
	detector.PublishBidTelemetry(domain.BidTelemetryEvent{
		AuctionID: "sub-tel-1", BidID: "b1", BidderID: "u1", Amount: 1000, PreviousPrice: 900, Timestamp: time.Now(),
	})
	// Salto irracional (> 300%)
	detector.PublishBidTelemetry(domain.BidTelemetryEvent{
		AuctionID: "sub-tel-1", BidID: "b2", BidderID: "u2", Amount: 5000, PreviousPrice: 1000, Timestamp: time.Now().Add(time.Second),
	})
	time.Sleep(50 * time.Millisecond)
	alerts := detector.ListAlertsByAuction("sub-tel-1")
	if len(alerts) == 0 {
		t.Fatal("expected async fraud telemetry to produce alert on >300% price jump")
	}
}
