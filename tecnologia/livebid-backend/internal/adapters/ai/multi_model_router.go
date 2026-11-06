package ai

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/fxrl01/livebid-backend/internal/core/domain"
)

var ErrModelNotFound = errors.New("ai: modelo no registrado en el router")

type cachedRiskEntry struct {
	eval      domain.EvaluacionRiesgo
	expiresAt time.Time
}

// MultiModelRouter permite orquestar múltiples proveedores/generaciones de IA en caliente,
// con caché TTL de evaluaciones y retroalimentación en tiempo real desde el detector de anomalías.
type MultiModelRouter struct {
	mu             sync.RWMutex
	activeModel    string
	models         map[string]domain.AIRiskEvaluator
	cache          map[string]cachedRiskEntry
	flaggedBidders map[string]string
	cacheTTL       time.Duration
	now            func() time.Time
}

func NewMultiModelRouter(defaultModel string, primary domain.AIRiskEvaluator, cacheTTL time.Duration) *MultiModelRouter {
	if defaultModel == "" {
		defaultModel = "gemini-2.5-flash"
	}
	if cacheTTL <= 0 {
		cacheTTL = time.Minute
	}
	r := &MultiModelRouter{
		activeModel:    defaultModel,
		models:         make(map[string]domain.AIRiskEvaluator),
		cache:          make(map[string]cachedRiskEntry),
		flaggedBidders: make(map[string]string),
		cacheTTL:       cacheTTL,
		now:            time.Now,
	}
	if primary != nil {
		r.models[defaultModel] = primary
	}
	return r
}

func (r *MultiModelRouter) RegisterModel(name string, evaluator domain.AIRiskEvaluator) {
	if name == "" || evaluator == nil {
		return
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.models[name] = evaluator
}

func (r *MultiModelRouter) SetActiveModel(name string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.models[name]; !ok {
		return ErrModelNotFound
	}
	r.activeModel = name
	r.cache = make(map[string]cachedRiskEntry)
	return nil
}

func (r *MultiModelRouter) ActiveModel() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.activeModel
}

func (r *MultiModelRouter) AvailableModels() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	names := make([]string, 0, len(r.models))
	for name := range r.models {
		names = append(names, name)
	}
	return names
}

// FlagSuspiciousBidder marca inmediatamente a un postor detectado por telemetría asíncrona
// e invalida su caché para exigir garantía reforzada (20%) o bloqueo preventivo.
func (r *MultiModelRouter) FlagSuspiciousBidder(postorID, reason string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.flaggedBidders[postorID] = reason
	for key := range r.cache {
		if len(key) >= len(postorID) && key[:len(postorID)] == postorID {
			delete(r.cache, key)
		}
	}
}

func (r *MultiModelRouter) EvaluarRiesgo(ctx context.Context, postorID, subastaID string, _ float64) (domain.NivelRiesgo, error) {
	eval, err := r.EvaluarPostor(ctx, postorID, subastaID)
	if err != nil {
		return domain.RiesgoMedio, err
	}
	return eval.Nivel, nil
}

func (r *MultiModelRouter) EvaluarPostor(ctx context.Context, postorID, subastaID string) (domain.EvaluacionRiesgo, error) {
	now := r.now()
	cacheKey := postorID + "|" + subastaID

	r.mu.RLock()
	if reason, flagged := r.flaggedBidders[postorID]; flagged {
		r.mu.RUnlock()
		return domain.EvaluacionRiesgo{
			Score:              25,
			Nivel:              domain.RiesgoAlto,
			PorcentajeGarantia: 0.20,
			Justificacion:      "Alerta conductual en tiempo real: " + reason,
			ModoFallback:       false,
		}, nil
	}
	if cached, ok := r.cache[cacheKey]; ok && now.Before(cached.expiresAt) {
		r.mu.RUnlock()
		return cached.eval, nil
	}
	evaluator := r.models[r.activeModel]
	r.mu.RUnlock()

	if evaluator == nil {
		return fallbackEvaluation(), nil
	}

	eval, err := evaluator.EvaluarPostor(ctx, postorID, subastaID)
	if err != nil {
		return fallbackEvaluation(), nil
	}

	r.mu.Lock()
	r.cache[cacheKey] = cachedRiskEntry{
		eval:      eval,
		expiresAt: now.Add(r.cacheTTL),
	}
	r.mu.Unlock()

	return eval, nil
}

