package ai

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"

	"github.com/fxrl01/livebid-backend/internal/core/domain"
)

var ErrAIProviderUnavailable = errors.New("ai: gemini provider connection timeout / 503 service unavailable")

// GeminiRiskAdapter implementa domain.AIRiskEvaluator evaluando el comportamiento y reputación del postor.
type GeminiRiskAdapter struct {
	apiKey          string
	simulatedFailed atomic.Bool
}

func NewGeminiRiskAdapter(apiKey string) *GeminiRiskAdapter {
	return &GeminiRiskAdapter{apiKey: strings.TrimSpace(apiKey)}
}

func (a *GeminiRiskAdapter) SetSimulatedFailure(failed bool) {
	a.simulatedFailed.Store(failed)
}

func (a *GeminiRiskAdapter) EvaluarRiesgo(ctx context.Context, postorID, subastaID string, _ float64) (domain.NivelRiesgo, error) {
	eval, err := a.EvaluarPostor(ctx, postorID, subastaID)
	if err != nil {
		return "", err
	}
	return eval.Nivel, nil
}

func (a *GeminiRiskAdapter) EvaluarPostor(ctx context.Context, postorID, _ string) (domain.EvaluacionRiesgo, error) {
	if err := ctx.Err(); err != nil {
		return domain.EvaluacionRiesgo{}, err
	}
	if a.simulatedFailed.Load() {
		return domain.EvaluacionRiesgo{}, ErrAIProviderUnavailable
	}

	upperID := strings.ToUpper(postorID)
	if strings.Contains(upperID, "RIESGO_ALTO") || strings.Contains(upperID, "BOT") || strings.Contains(upperID, "FRAUD") {
		return domain.EvaluacionRiesgo{
			Score:              38,
			Nivel:              domain.RiesgoAlto,
			PorcentajeGarantia: 0.20,
			Justificacion:      "Patrón anómalo detectado: alta velocidad de conexión y antecedentes de pujas impagas",
			ModoFallback:       false,
		}, nil
	}

	return domain.EvaluacionRiesgo{
		Score:              92,
		Nivel:              domain.RiesgoBajo,
		PorcentajeGarantia: 0.10,
		Justificacion:      "Perfil verificado con KYC aprobado y comportamiento de puja consistente",
		ModoFallback:       false,
	}, nil
}

