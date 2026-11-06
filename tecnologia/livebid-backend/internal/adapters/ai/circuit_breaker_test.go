package ai_test

import (
	"context"
	"testing"
	"time"

	"github.com/fxrl01/livebid-backend/internal/adapters/ai"
	"github.com/fxrl01/livebid-backend/internal/core/domain"
)

func TestGeminiRiskAdapterClassifiesLowAndHighRiskBidders(t *testing.T) {
	adapter := ai.NewGeminiRiskAdapter("test-key")

	low, err := adapter.EvaluarPostor(context.Background(), "USR-VIP-01", "SUB-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if low.Score != 92 || low.Nivel != domain.RiesgoBajo || low.PorcentajeGarantia != 0.10 {
		t.Fatalf("unexpected low risk evaluation: %+v", low)
	}

	high, err := adapter.EvaluarPostor(context.Background(), "USR-RIESGO_ALTO", "SUB-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if high.Score != 38 || high.Nivel != domain.RiesgoAlto || high.PorcentajeGarantia != 0.20 {
		t.Fatalf("unexpected high risk evaluation: %+v", high)
	}
}

func TestCircuitBreakerOpensAfterThresholdAndRecoversAfterCooldown(t *testing.T) {
	provider := ai.NewGeminiRiskAdapter("test-key")
	breaker := ai.NewCircuitBreakerIA(provider, 2, 20*time.Millisecond, time.Second)

	provider.SetSimulatedFailure(true)

	first, err := breaker.EvaluarPostor(context.Background(), "USR-VIP-01", "SUB-1")
	if err != nil || !first.ModoFallback || breaker.State() != ai.StateClosed {
		t.Fatalf("expected fallback in CLOSED after 1st failure, got state=%s eval=%+v", breaker.State(), first)
	}

	second, err := breaker.EvaluarPostor(context.Background(), "USR-VIP-01", "SUB-1")
	if err != nil || !second.ModoFallback || breaker.State() != ai.StateOpen {
		t.Fatalf("expected circuit OPEN after 2nd failure, got state=%s eval=%+v", breaker.State(), second)
	}

	// While OPEN, even if provider recovers, calls before cooldown use local fallback
	provider.SetSimulatedFailure(false)
	whileOpen, _ := breaker.EvaluarPostor(context.Background(), "USR-VIP-01", "SUB-1")
	if !whileOpen.ModoFallback {
		t.Fatalf("expected fallback while circuit is OPEN before cooldown")
	}

	// After cooldown, circuit transitions HALF_OPEN -> CLOSED
	time.Sleep(25 * time.Millisecond)
	recovered, err := breaker.EvaluarPostor(context.Background(), "USR-VIP-01", "SUB-1")
	if err != nil || recovered.ModoFallback || breaker.State() != ai.StateClosed || recovered.Score != 92 {
		t.Fatalf("expected circuit recovery to CLOSED with score 92, got state=%s eval=%+v", breaker.State(), recovered)
	}
}

func TestMultiModelRouterAndAsyncFraudDetectorFlagsCollusionAndBotBursts(t *testing.T) {
	provider := ai.NewGeminiRiskAdapter("test-key")
	router := ai.NewMultiModelRouter("gemini-2.5-flash", provider, time.Minute)
	router.RegisterModel("gemini-2.5-pro-forensic", provider)

	if err := router.SetActiveModel("gemini-2.5-pro-forensic"); err != nil || router.ActiveModel() != "gemini-2.5-pro-forensic" {
		t.Fatalf("expected active model switch to gemini-2.5-pro-forensic, err=%v", err)
	}

	detector := ai.NewAsyncFraudDetector(router)
	baseTime := time.Now()

	// Simulate 4 alternating bids (A -> B -> A -> B) with normal 5s spacing to trigger Ping-Pong collusion
	bidders := []string{"usr-colluder-A", "usr-colluder-B", "usr-colluder-A", "usr-colluder-B"}
	for i, bidder := range bidders {
		detector.AnalyzeBidSync(domain.BidTelemetryEvent{
			AuctionID:     "sub-collusion",
			BidID:         "bid",
			BidderID:      bidder,
			Amount:        float64(1000 + (i+1)*100),
			PreviousPrice: float64(1000 + i*100),
			Timestamp:     baseTime.Add(time.Duration(i*5) * time.Second),
		})
	}

	alerts := detector.ListAlertsByAuction("sub-collusion")
	if len(alerts) == 0 || alerts[0].Type != domain.AlertShillPingPong {
		t.Fatalf("expected COLUSION_PING_PONG alert, got %+v", alerts)
	}

	// Colluder A must now be dynamically flagged as HIGH RISK (20% guarantee, Score 25) in the AI Router
	eval, err := router.EvaluarPostor(context.Background(), "usr-colluder-A", "sub-collusion")
	if err != nil || eval.Nivel != domain.RiesgoAlto || eval.PorcentajeGarantia != 0.20 || eval.Score != 25 {
		t.Fatalf("expected flagged colluder to be downgraded to HIGH RISK (Score 25, 20%%), got %+v", eval)
	}
}

