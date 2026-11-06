package ai

import (
	"context"
	"sync"
	"time"

	"github.com/fxrl01/livebid-backend/internal/core/domain"
)

type CircuitState string

const (
	StateClosed   CircuitState = "CLOSED"
	StateOpen     CircuitState = "OPEN"
	StateHalfOpen CircuitState = "HALF_OPEN"
)

// CircuitBreakerIA envuelve un domain.AIRiskEvaluator con protección Circuit Breaker y fallback heurístico local.
type CircuitBreakerIA struct {
	mu                  sync.Mutex
	provider            domain.AIRiskEvaluator
	state               CircuitState
	consecutiveFailures int
	failThreshold       int
	cooldown            time.Duration
	callTimeout         time.Duration
	lastFailureAt       time.Time
	now                 func() time.Time
}

func NewCircuitBreakerIA(provider domain.AIRiskEvaluator, failThreshold int, cooldown, callTimeout time.Duration) *CircuitBreakerIA {
	if failThreshold <= 0 {
		failThreshold = 2
	}
	if cooldown <= 0 {
		cooldown = 3 * time.Second
	}
	if callTimeout <= 0 {
		callTimeout = 2 * time.Second
	}
	return &CircuitBreakerIA{
		provider:      provider,
		state:         StateClosed,
		failThreshold: failThreshold,
		cooldown:      cooldown,
		callTimeout:   callTimeout,
		now:           time.Now,
	}
}

func (cb *CircuitBreakerIA) State() CircuitState {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	return cb.state
}

func (cb *CircuitBreakerIA) EvaluarRiesgo(ctx context.Context, postorID, subastaID string, _ float64) (domain.NivelRiesgo, error) {
	eval, err := cb.EvaluarPostor(ctx, postorID, subastaID)
	if err != nil {
		return domain.RiesgoMedio, nil
	}
	return eval.Nivel, nil
}

func (cb *CircuitBreakerIA) EvaluarPostor(ctx context.Context, postorID, subastaID string) (domain.EvaluacionRiesgo, error) {
	cb.mu.Lock()
	now := cb.now()
	if cb.state == StateOpen {
		if now.Sub(cb.lastFailureAt) >= cb.cooldown {
			cb.state = StateHalfOpen
		} else {
			cb.mu.Unlock()
			return fallbackEvaluation(), nil
		}
	}
	cb.mu.Unlock()

	callCtx, cancel := context.WithTimeout(ctx, cb.callTimeout)
	defer cancel()

	result, err := cb.provider.EvaluarPostor(callCtx, postorID, subastaID)

	cb.mu.Lock()
	defer cb.mu.Unlock()

	if err != nil {
		cb.consecutiveFailures++
		cb.lastFailureAt = cb.now()
		if cb.consecutiveFailures >= cb.failThreshold || cb.state == StateHalfOpen {
			cb.state = StateOpen
		}
		return fallbackEvaluation(), nil
	}

	cb.state = StateClosed
	cb.consecutiveFailures = 0
	return result, nil
}

func fallbackEvaluation() domain.EvaluacionRiesgo {
	return domain.EvaluacionRiesgo{
		Score:              75,
		Nivel:              domain.RiesgoMedio,
		PorcentajeGarantia: 0.10,
		Justificacion:      "Evaluación heurística local por contingencia (API de IA no disponible temporalmente)",
		ModoFallback:       true,
	}
}

