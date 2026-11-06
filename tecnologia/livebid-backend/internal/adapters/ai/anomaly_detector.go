package ai

import (
	"fmt"
	"sync"
	"time"

	"github.com/fxrl01/livebid-backend/internal/core/domain"
)

// AsyncFraudDetector analiza la telemetría de pujas en segundo plano (sin añadir latencia al motor)
// para detectar colusión (Shill Bidding / Ping-Pong), ráfagas de bots y saltos anómalos.
type AsyncFraudDetector struct {
	mu       sync.RWMutex
	history  map[string][]domain.BidTelemetryEvent
	alerts   map[string][]domain.FraudAlert
	router   *MultiModelRouter
	eventsCh chan domain.BidTelemetryEvent
}

func NewAsyncFraudDetector(router *MultiModelRouter) *AsyncFraudDetector {
	d := &AsyncFraudDetector{
		history:  make(map[string][]domain.BidTelemetryEvent),
		alerts:   make(map[string][]domain.FraudAlert),
		router:   router,
		eventsCh: make(chan domain.BidTelemetryEvent, 256),
	}
	go d.worker()
	return d
}

func (d *AsyncFraudDetector) worker() {
	for ev := range d.eventsCh {
		d.AnalyzeBidSync(ev)
	}
}

func (d *AsyncFraudDetector) PublishBidTelemetry(event domain.BidTelemetryEvent) {
	select {
	case d.eventsCh <- event:
	default:
		// Nunca bloquear el camino crítico de pujas si el buffer está lleno
	}
}

// AnalyzeBidSync evalúa sincrónicamente un evento de puja y registra alertas si detecta patrones de fraude.
func (d *AsyncFraudDetector) AnalyzeBidSync(event domain.BidTelemetryEvent) []domain.FraudAlert {
	d.mu.Lock()
	defer d.mu.Unlock()

	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	seq := append(d.history[event.AuctionID], event)
	d.history[event.AuctionID] = seq

	generated := make([]domain.FraudAlert, 0)
	n := len(seq)

	// 1. Detección de velocidad sobrehumana (< 150ms entre pujas consecutivas)
	if n >= 2 {
		prev := seq[n-2]
		delta := event.Timestamp.Sub(prev.Timestamp)
		if delta >= 0 && delta < 150*time.Millisecond {
			alert := domain.FraudAlert{
				ID:          fmt.Sprintf("alrt_bot_%s_%d", event.AuctionID, n),
				AuctionID:   event.AuctionID,
				Type:        domain.AlertBotSpeedBurst,
				Severity:    domain.RiesgoAlto,
				SuspectIDs:  []string{event.BidderID},
				Description: fmt.Sprintf("Ráfaga sub-humana detectada (%d ms desde la oferta previa)", delta.Milliseconds()),
				DetectedAt:  event.Timestamp,
			}
			d.alerts[event.AuctionID] = append(d.alerts[event.AuctionID], alert)
			generated = append(generated, alert)
			if d.router != nil {
				d.router.FlagSuspiciousBidder(event.BidderID, alert.Description)
			}
		}
	}

	// 2. Detección de Colusión Ping-Pong (Shill Bidding: A -> B -> A -> B en 4 ofertas seguidas)
	if n >= 4 {
		b1, b2, b3, b4 := seq[n-4].BidderID, seq[n-3].BidderID, seq[n-2].BidderID, seq[n-1].BidderID
		if b1 != b2 && b1 == b3 && b2 == b4 {
			alert := domain.FraudAlert{
				ID:          fmt.Sprintf("alrt_collusion_%s_%d", event.AuctionID, n),
				AuctionID:   event.AuctionID,
				Type:        domain.AlertShillPingPong,
				Severity:    domain.RiesgoAlto,
				SuspectIDs:  []string{b1, b2},
				Description: fmt.Sprintf("Patrón de colusión Ping-Pong detectado entre %s y %s (4 alternancias consecutivas)", b1, b2),
				DetectedAt:  event.Timestamp,
			}
			d.alerts[event.AuctionID] = append(d.alerts[event.AuctionID], alert)
			generated = append(generated, alert)
			if d.router != nil {
				d.router.FlagSuspiciousBidder(b1, alert.Description)
				d.router.FlagSuspiciousBidder(b2, alert.Description)
			}
		}
	}

	// 3. Detección de salto irracional de precio (>= 50% de incremento en un solo paso)
	if event.PreviousPrice > 0 && event.Amount >= event.PreviousPrice*1.50 {
		alert := domain.FraudAlert{
			ID:          fmt.Sprintf("alrt_jump_%s_%d", event.AuctionID, n),
			AuctionID:   event.AuctionID,
			Type:        domain.AlertIrrationalJump,
			Severity:    domain.RiesgoMedio,
			SuspectIDs:  []string{event.BidderID},
			Description: fmt.Sprintf("Salto abrupto de precio (+%.0f%% sobre %.2f)", ((event.Amount-event.PreviousPrice)/event.PreviousPrice)*100, event.PreviousPrice),
			DetectedAt:  event.Timestamp,
		}
		d.alerts[event.AuctionID] = append(d.alerts[event.AuctionID], alert)
		generated = append(generated, alert)
	}

	return generated
}

func (d *AsyncFraudDetector) ListAlertsByAuction(auctionID string) []domain.FraudAlert {
	d.mu.RLock()
	defer d.mu.RUnlock()
	src := d.alerts[auctionID]
	out := make([]domain.FraudAlert, len(src))
	copy(out, src)
	return out
}

