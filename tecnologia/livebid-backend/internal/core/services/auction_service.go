package services

import (
	"context"
	"fmt"
	"time"

	"github.com/fxrl01/livebid-backend/internal/core/domain"
)

// ProcesarPujaCmd encapsula los datos de entrada para emitir una puja.
type ProcesarPujaCmd struct {
	PujaID    string
	SubastaID string
	PostorID  string
	Monto     float64
	Timestamp time.Time
}

// PujaResponseDTO contiene el resultado de la puja procesada.
type PujaResponseDTO struct {
	SubastaID       string        `json:"subasta_id"`
	PujaID          string        `json:"puja_id"`
	PostorLiderID   string        `json:"postor_lider_id"`
	PrecioActual    float64       `json:"precio_actual"`
	TiempoExtendido bool          `json:"tiempo_extendido"`
	FechaCierre     time.Time     `json:"fecha_cierre"`
	Estado          domain.EstadoSubasta `json:"estado"`
}

// AuctionService es el servicio de aplicación que orquesta el caso de uso ProcesarPuja.
type AuctionService struct {
	repo     domain.SubastaRepository
	lock     domain.BidLockPort
	escrow   domain.PaymentEscrowPort
	notifier domain.RoomNotifierPort
	risk     domain.AIRiskPort
}

// NuevoAuctionService crea una nueva instancia de AuctionService inyectando sus dependencias.
func NuevoAuctionService(
	repo domain.SubastaRepository,
	lock domain.BidLockPort,
	escrow domain.PaymentEscrowPort,
	notifier domain.RoomNotifierPort,
	risk domain.AIRiskPort,
) *AuctionService {
	return &AuctionService{
		repo:     repo,
		lock:     lock,
		escrow:   escrow,
		notifier: notifier,
		risk:     risk,
	}
}

// ProcesarPuja orquesta la validación atómica y adjudicación de una nueva oferta.
func (s *AuctionService) ProcesarPuja(ctx context.Context, cmd ProcesarPujaCmd) (*PujaResponseDTO, error) {
	// 1. Adquirir lock distribuido atómico (Redis SETNX) para evitar race conditions
	unlock, err := s.lock.AdquirirLock(ctx, cmd.SubastaID, 1500*time.Millisecond)
	if err != nil {
		return nil, domain.ErrLockTimeout
	}
	defer func() {
		_ = unlock()
	}()

	// 2. Validar que el postor tenga garantía financiera activa en custodia
	if s.escrow != nil {
		garantiaActiva, err := s.escrow.ValidarGarantiaActiva(ctx, cmd.PostorID, cmd.SubastaID)
		if err != nil {
			return nil, fmt.Errorf("error al validar garantia: %w", err)
		}
		if !garantiaActiva {
			return nil, domain.ErrGarantiaInsuficiente
		}
	}

	// 3. Evaluación preventiva de riesgo por IA (Gemini con Circuit Breaker)
	if s.risk != nil {
		nivel, err := s.risk.EvaluarRiesgo(ctx, cmd.PostorID, cmd.SubastaID, cmd.Monto)
		if err == nil && nivel == domain.RiesgoAlto {
			return nil, domain.ErrRiesgoAltoIA
		}
	}

	// 4. Obtener la entidad del agregado Subasta
	subasta, err := s.repo.ObtenerPorID(ctx, cmd.SubastaID)
	if err != nil {
		return nil, domain.ErrSubastaNoEncontrada
	}

	// 5. Ejecutar invariantes de negocio en el dominio (incremento, anti-sniping, etc.)
	ahora := cmd.Timestamp
	if ahora.IsZero() {
		ahora = time.Now()
	}
	puja, tiempoExtendido, err := subasta.ValidarYRegistrarPuja(cmd.PujaID, cmd.PostorID, cmd.Monto, ahora)
	if err != nil {
		return nil, err
	}

	// 6. Persistir el nuevo estado transaccional
	if err := s.repo.Guardar(ctx, subasta); err != nil {
		return nil, fmt.Errorf("error al guardar subasta: %w", err)
	}

	// 7. Difundir evento en tiempo real a las salas WebSocket
	if s.notifier != nil {
		_ = s.notifier.BroadcastNuevaPuja(ctx, subasta.ID, *puja, tiempoExtendido)
	}

	return &PujaResponseDTO{
		SubastaID:       subasta.ID,
		PujaID:          puja.ID,
		PostorLiderID:   subasta.PostorLiderID,
		PrecioActual:    subasta.PrecioActual,
		TiempoExtendido: tiempoExtendido,
		FechaCierre:     subasta.FechaCierre,
		Estado:          subasta.Estado,
	}, nil
}

