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
	SubastaID       string               `json:"subasta_id"`
	PujaID          string               `json:"puja_id"`
	PostorLiderID   string               `json:"postor_lider_id"`
	PrecioActual    float64              `json:"precio_actual"`
	TiempoExtendido bool                 `json:"tiempo_extendido"`
	FechaCierre     time.Time            `json:"fecha_cierre"`
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

type AuctionCatalogService struct {
	repo       domain.SubastaRepository
	guarantees domain.GuaranteeRepository
	notifier   domain.BidNotifierPort
	now        func() time.Time
}

type CrearSubastaCmd struct {
	ID          string
	VendedorID  string
	Titulo      string
	Descripcion string
	PrecioBase  float64
	Inicio      time.Time
	Cierre      time.Time
}

func NuevoAuctionCatalogService(repo domain.SubastaRepository) *AuctionCatalogService {
	return &AuctionCatalogService{repo: repo, now: time.Now}
}

func NuevoAuctionCatalogServiceConGarantias(repo domain.SubastaRepository, guarantees domain.GuaranteeRepository) *AuctionCatalogService {
	return &AuctionCatalogService{repo: repo, guarantees: guarantees, now: time.Now}
}

func NuevoAuctionCatalogServiceConGarantiasYNotificador(repo domain.SubastaRepository, guarantees domain.GuaranteeRepository, notifier domain.BidNotifierPort) *AuctionCatalogService {
	return &AuctionCatalogService{repo: repo, guarantees: guarantees, notifier: notifier, now: time.Now}
}

func (s *AuctionCatalogService) Crear(ctx context.Context, cmd CrearSubastaCmd) (*domain.Subasta, error) {
	if cmd.ID == "" || cmd.VendedorID == "" || cmd.Titulo == "" || cmd.PrecioBase <= 0 ||
		cmd.Inicio.IsZero() || cmd.Cierre.IsZero() || !cmd.Cierre.After(cmd.Inicio) {
		return nil, domain.ErrSubastaInvalida
	}
	subasta := domain.NuevaSubasta(cmd.ID, cmd.Titulo, cmd.Descripcion, cmd.PrecioBase, cmd.Inicio, cmd.Cierre)
	subasta.VendedorID = cmd.VendedorID
	if err := s.repo.Guardar(ctx, subasta); err != nil {
		return nil, err
	}
	return subasta, nil
}

func (s *AuctionCatalogService) Listar(ctx context.Context) ([]*domain.Subasta, error) {
	return s.repo.Listar(ctx)
}

func (s *AuctionCatalogService) Obtener(ctx context.Context, id string) (*domain.Subasta, error) {
	if id == "" {
		return nil, domain.ErrSubastaNoEncontrada
	}
	return s.repo.ObtenerPorID(ctx, id)
}

func (s *AuctionCatalogService) Activar(ctx context.Context, id string) error {
	subasta, err := s.Obtener(ctx, id)
	if err != nil {
		return err
	}
	if err := subasta.Activar(); err != nil {
		return err
	}
	return s.repo.Guardar(ctx, subasta)
}

func (s *AuctionCatalogService) Cerrar(ctx context.Context, id string) error {
	subasta, err := s.Obtener(ctx, id)
	if err != nil {
		return err
	}
	if err := subasta.Cerrar(); err != nil {
		return err
	}
	if err := s.repo.Guardar(ctx, subasta); err != nil {
		return err
	}
	if s.guarantees != nil {
		if err := s.guarantees.SettleAuction(ctx, subasta.ID, subasta.PostorLiderID); err != nil {
			return fmt.Errorf("settle auction guarantees: %w", err)
		}
	}
	if s.notifier != nil {
		var winner *domain.Puja
		if subasta.PostorLiderID != "" {
			winner = &domain.Puja{
				SubastaID: subasta.ID, PostorID: subasta.PostorLiderID,
				Monto: subasta.PrecioActual, Timestamp: s.now(),
			}
		}
		if err := s.notifier.BroadcastSubastaCerrada(ctx, subasta.ID, winner); err != nil {
			return fmt.Errorf("broadcast auction closure: %w", err)
		}
	}
	return nil
}

func (s *AuctionCatalogService) CerrarVencidas(ctx context.Context, now time.Time) (int, error) {
	auctions, err := s.repo.Listar(ctx)
	if err != nil {
		return 0, fmt.Errorf("list expired auctions: %w", err)
	}
	closed := 0
	for _, auction := range auctions {
		if !auction.Estado.EsAceptandoPujas() || auction.FechaCierre.After(now) {
			continue
		}
		if err := s.Cerrar(ctx, auction.ID); err != nil {
			return closed, fmt.Errorf("close expired auction %s: %w", auction.ID, err)
		}
		closed++
	}
	return closed, nil
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
