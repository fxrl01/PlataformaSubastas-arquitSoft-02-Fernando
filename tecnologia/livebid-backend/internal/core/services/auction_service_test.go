package services_test

import (
	"context"
	"testing"
	"time"

	"github.com/fxrl01/livebid-backend/internal/core/domain"
	"github.com/fxrl01/livebid-backend/internal/core/services"
)

// MockRepo simula el repositorio de subastas en memoria.
type MockRepo struct {
	subastas map[string]*domain.Subasta
}

func (m *MockRepo) ObtenerPorID(_ context.Context, id string) (*domain.Subasta, error) {
	s, ok := m.subastas[id]
	if !ok {
		return nil, domain.ErrSubastaNoEncontrada
	}
	return s, nil
}

func (m *MockRepo) Guardar(_ context.Context, s *domain.Subasta) error {
	m.subastas[s.ID] = s
	return nil
}

func (m *MockRepo) Listar(_ context.Context) ([]*domain.Subasta, error) {
	result := make([]*domain.Subasta, 0, len(m.subastas))
	for _, subasta := range m.subastas {
		result = append(result, subasta)
	}
	return result, nil
}

// MockLock simula el lock distribuido de Redis.
type MockLock struct{}

func (m *MockLock) AdquirirLock(_ context.Context, _ string, _ time.Duration) (domain.UnlockFunc, error) {
	return func() error { return nil }, nil
}

// MockEscrow simula la pasarela de custodia financiera.
type MockEscrow struct {
	valido bool
}

func (m *MockEscrow) ValidarGarantiaActiva(_ context.Context, _, _ string) (bool, error) {
	return m.valido, nil
}

func (m *MockEscrow) RetenerGarantia(_ context.Context, _, _ string, _ float64) error {
	return nil
}

func (m *MockEscrow) LiberarGarantia(_ context.Context, _, _ string) error {
	return nil
}

// MockNotifier simula la difusión WebSocket.
type MockNotifier struct {
	UltimaPujaNotificada *domain.Puja
}

func (m *MockNotifier) BroadcastNuevaPuja(_ context.Context, _ string, p domain.Puja, _ bool) error {
	m.UltimaPujaNotificada = &p
	return nil
}

func (m *MockNotifier) BroadcastSubastaCerrada(_ context.Context, _ string, _ *domain.Puja) error {
	return nil
}

// MockRisk simula el analizador de riesgo con Gemini AI.
type MockRisk struct {
	nivel domain.NivelRiesgo
}

func (m *MockRisk) EvaluarRiesgo(_ context.Context, _, _ string, _ float64) (domain.NivelRiesgo, error) {
	return m.nivel, nil
}

func TestProcesarPuja_ValidaExitosamente(t *testing.T) {
	ctx := context.Background()
	ahora := time.Now()
	cierre := ahora.Add(10 * time.Minute)

	subasta := domain.NuevaSubasta("sub-1", "Rolex Daytona 1968", "Reloj vintage", 10000, ahora, cierre)
	_ = subasta.Activar()

	repo := &MockRepo{subastas: map[string]*domain.Subasta{"sub-1": subasta}}
	lock := &MockLock{}
	escrow := &MockEscrow{valido: true}
	notifier := &MockNotifier{}
	risk := &MockRisk{nivel: domain.RiesgoBajo}

	service := services.NuevoAuctionService(repo, lock, escrow, notifier, risk)

	cmd := services.ProcesarPujaCmd{
		PujaID:    "puj-101",
		SubastaID: "sub-1",
		PostorID:  "usr-77",
		Monto:     10500, // Incremento mínimo para 10000 es 500
		Timestamp: ahora,
	}

	res, err := service.ProcesarPuja(ctx, cmd)
	if err != nil {
		t.Fatalf("se esperaba puja exitosa, pero fallo con: %v", err)
	}

	if res.PrecioActual != 10500 {
		t.Errorf("precio esperado 10500, obtenido %v", res.PrecioActual)
	}
	if res.PostorLiderID != "usr-77" {
		t.Errorf("postor líder esperado usr-77, obtenido %v", res.PostorLiderID)
	}
	if notifier.UltimaPujaNotificada == nil || notifier.UltimaPujaNotificada.Monto != 10500 {
		t.Errorf("la puja no fue notificada correctamente al hub WebSocket")
	}
}

func TestProcesarPuja_RechazaPorIncrementoInsuficiente(t *testing.T) {
	ctx := context.Background()
	ahora := time.Now()
	cierre := ahora.Add(10 * time.Minute)

	subasta := domain.NuevaSubasta("sub-2", "Patek Philippe 5711", "Edicion acero", 50000, ahora, cierre)
	_ = subasta.Activar()

	repo := &MockRepo{subastas: map[string]*domain.Subasta{"sub-2": subasta}}
	service := services.NuevoAuctionService(repo, &MockLock{}, &MockEscrow{valido: true}, &MockNotifier{}, &MockRisk{nivel: domain.RiesgoBajo})

	cmd := services.ProcesarPujaCmd{
		PujaID:    "puj-102",
		SubastaID: "sub-2",
		PostorID:  "usr-88",
		Monto:     50500, // Para 50,000 el incremento mínimo es 1,000 -> debe ofertar >= 51,000
		Timestamp: ahora,
	}

	_, err := service.ProcesarPuja(ctx, cmd)
	if err != domain.ErrPujaInferior {
		t.Errorf("se esperaba ErrPujaInferior, obtenido: %v", err)
	}
}

func TestProcesarPuja_AntiSnipingExtiendeTiempo(t *testing.T) {
	ctx := context.Background()
	ahora := time.Now()
	// Subasta cerrará en 10 segundos (dentro de los 15s de umbral anti-sniping)
	cierreInicial := ahora.Add(10 * time.Second)

	subasta := domain.NuevaSubasta("sub-3", "Ferrari 250 GTO", "Vehiculo clasico", 100000, ahora, cierreInicial)
	_ = subasta.Activar()

	repo := &MockRepo{subastas: map[string]*domain.Subasta{"sub-3": subasta}}
	service := services.NuevoAuctionService(repo, &MockLock{}, &MockEscrow{valido: true}, &MockNotifier{}, &MockRisk{nivel: domain.RiesgoBajo})

	cmd := services.ProcesarPujaCmd{
		PujaID:    "puj-103",
		SubastaID: "sub-3",
		PostorID:  "usr-99",
		Monto:     102500,
		Timestamp: ahora,
	}

	res, err := service.ProcesarPuja(ctx, cmd)
	if err != nil {
		t.Fatalf("error inesperado: %v", err)
	}

	if !res.TiempoExtendido {
		t.Errorf("se esperaba activación de anti-sniping")
	}

	// Debe haberse extendido +30 segundos
	esperadoCierre := cierreInicial.Add(30 * time.Second)
	if !res.FechaCierre.Equal(esperadoCierre) {
		t.Errorf("cierre esperado %v, obtenido %v", esperadoCierre, res.FechaCierre)
	}
}

func TestProcesarPuja_BloqueoPorRiesgoAltoIA(t *testing.T) {
	ctx := context.Background()
	ahora := time.Now()
	cierre := ahora.Add(10 * time.Minute)

	subasta := domain.NuevaSubasta("sub-4", "Diamante Hope", "Joya exclusiva", 200000, ahora, cierre)
	_ = subasta.Activar()

	repo := &MockRepo{subastas: map[string]*domain.Subasta{"sub-4": subasta}}
	// Simulamos que el agente Gemini detecta riesgo alto de fraude
	risk := &MockRisk{nivel: domain.RiesgoAlto}
	service := services.NuevoAuctionService(repo, &MockLock{}, &MockEscrow{valido: true}, &MockNotifier{}, risk)

	cmd := services.ProcesarPujaCmd{
		PujaID:    "puj-104",
		SubastaID: "sub-4",
		PostorID:  "usr-fraudulento",
		Monto:     205000,
		Timestamp: ahora,
	}

	_, err := service.ProcesarPuja(ctx, cmd)
	if err != domain.ErrRiesgoAltoIA {
		t.Errorf("se esperaba ErrRiesgoAltoIA, obtenido: %v", err)
	}
}
