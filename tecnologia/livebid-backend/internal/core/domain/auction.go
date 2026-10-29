package domain

import (
	"fmt"
	"time"
)

const (
	// UmbralAntiSniping define los últimos segundos antes del cierre en los cuales
	// una puja activa la extensión automática del cronómetro.
	UmbralAntiSniping = 15 * time.Second

	// ExtensionAntiSniping define el tiempo adicional que se suma al cierre
	// cuando se detecta una puja en la franja anti-sniping.
	ExtensionAntiSniping = 30 * time.Second
)

// Subasta es la entidad raíz de agregado (Aggregate Root) del dominio de subastas.
// Contiene toda la lógica de negocio invariante: validación de pujas, anti-sniping,
// cálculo de incrementos y gestión de estados.
type Subasta struct {
	// ID es el identificador único de la subasta (UUID v4).
	ID string

	// Titulo es el nombre descriptivo del bien subastado.
	Titulo string

	// Descripcion contiene los detalles del bien o lote en subasta.
	Descripcion string

	// PrecioBase es el precio inicial de apertura establecido por el vendedor.
	PrecioBase float64

	// PrecioActual es el precio líder vigente (última puja aceptada).
	PrecioActual float64

	// PostorLiderID identifica al postor que realizó la última puja válida.
	PostorLiderID string

	// Estado indica la fase actual del ciclo de vida de la subasta.
	Estado EstadoSubasta

	// FechaInicio marca el inicio programado de la subasta.
	FechaInicio time.Time

	// FechaCierre marca el cierre planificado (extensible por anti-sniping).
	FechaCierre time.Time

	// Pujas almacena el historial ordenado de pujas aceptadas.
	Pujas []Puja
}

// NuevaSubasta crea una instancia de Subasta con los parámetros iniciales
// y la establece en estado PENDIENTE.
func NuevaSubasta(id, titulo, descripcion string, precioBase float64, inicio, cierre time.Time) *Subasta {
	return &Subasta{
		ID:           id,
		Titulo:       titulo,
		Descripcion:  descripcion,
		PrecioBase:   precioBase,
		PrecioActual: precioBase,
		Estado:       SubastaPendiente,
		FechaInicio:  inicio,
		FechaCierre:  cierre,
		Pujas:        make([]Puja, 0),
	}
}

// Activar cambia el estado de la subasta a ACTIVA si está en PENDIENTE.
func (s *Subasta) Activar() error {
	if s.Estado != SubastaPendiente {
		return fmt.Errorf("domain: no se puede activar una subasta en estado %s", s.Estado)
	}
	s.Estado = SubastaActiva
	return nil
}

// IniciarConteo cambia el estado de la subasta a EN_CONTEO (cuenta regresiva del martillo).
func (s *Subasta) IniciarConteo() error {
	if s.Estado != SubastaActiva {
		return fmt.Errorf("domain: no se puede iniciar conteo en estado %s", s.Estado)
	}
	s.Estado = SubastaEnConteo
	return nil
}

// Cerrar cambia el estado de la subasta a CERRADA (martillo final).
func (s *Subasta) Cerrar() error {
	if s.Estado != SubastaEnConteo && s.Estado != SubastaActiva {
		return fmt.Errorf("domain: no se puede cerrar una subasta en estado %s", s.Estado)
	}
	s.Estado = SubastaCerrada
	return nil
}

// CalcularIncrementoMinimo retorna el incremento mínimo requerido para
// superar la oferta líder actual, basado en los tramos definidos.
func (s *Subasta) CalcularIncrementoMinimo() float64 {
	return CalcularIncrementoMinimo(s.PrecioActual)
}

// EsUltimosSegundos determina si el instante proporcionado cae dentro
// de la franja anti-sniping (últimos 15 segundos antes del cierre).
func (s *Subasta) EsUltimosSegundos(ahora time.Time) bool {
	tiempoRestante := s.FechaCierre.Sub(ahora)
	return tiempoRestante > 0 && tiempoRestante <= UmbralAntiSniping
}

// ExtenderTiempo suma la extensión anti-sniping al cierre programado.
func (s *Subasta) ExtenderTiempo() {
	s.FechaCierre = s.FechaCierre.Add(ExtensionAntiSniping)
}

// ValidarYRegistrarPuja ejecuta todas las reglas de negocio para aceptar o rechazar
// una nueva puja. Si es válida, la registra en el historial y retorna la puja creada
// junto con un indicador de si se activó la extensión anti-sniping.
//
// Reglas validadas:
//  1. La subasta debe estar en estado ACTIVA o EN_CONTEO.
//  2. El postor no puede ser el mismo que el líder actual.
//  3. El monto debe ser >= precioActual + incrementoMínimo.
//  4. Si cae en los últimos 15 segundos, extiende el cierre +30s.
func (s *Subasta) ValidarYRegistrarPuja(pujaID, postorID string, monto float64, ahora time.Time) (*Puja, bool, error) {
	// 1. Verificar estado
	if !s.Estado.EsAceptandoPujas() {
		return nil, false, ErrSubastaCerrada
	}

	// 2. Verificar que no sea el mismo postor líder
	if s.PostorLiderID == postorID && s.PostorLiderID != "" {
		return nil, false, ErrPostorIgual
	}

	// 3. Verificar incremento mínimo
	incremento := s.CalcularIncrementoMinimo()
	if monto < s.PrecioActual+incremento {
		return nil, false, ErrPujaInferior
	}

	// 4. Anti-Sniping: extender tiempo si la puja entra en los últimos 15 segundos
	tiempoExtendido := false
	if s.EsUltimosSegundos(ahora) {
		s.ExtenderTiempo()
		tiempoExtendido = true
	}

	// 5. Registrar la puja
	puja := NuevaPuja(pujaID, s.ID, postorID, monto, ahora)
	s.PrecioActual = monto
	s.PostorLiderID = postorID
	s.Pujas = append(s.Pujas, puja)

	return &puja, tiempoExtendido, nil
}

// TienePujas indica si la subasta ha recibido al menos una oferta.
func (s *Subasta) TienePujas() bool {
	return len(s.Pujas) > 0
}

// UltimaPuja retorna la puja más reciente o nil si no hay pujas registradas.
func (s *Subasta) UltimaPuja() *Puja {
	if len(s.Pujas) == 0 {
		return nil
	}
	puja := s.Pujas[len(s.Pujas)-1]
	return &puja
}

