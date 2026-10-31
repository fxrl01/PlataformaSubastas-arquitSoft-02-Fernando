// Package domain contiene las entidades, value objects e interfaces del núcleo de negocio.
// Ningún paquete externo (frameworks, drivers, adaptadores) es importado aquí.
package domain

// EstadoSubasta representa la máquina de estados del ciclo de vida de una subasta.
type EstadoSubasta string

const (
	// SubastaPendiente indica que la subasta fue creada pero aún no está abierta al público.
	SubastaPendiente EstadoSubasta = "PENDIENTE"

	// SubastaActiva indica que la subasta está en curso y acepta pujas.
	SubastaActiva EstadoSubasta = "ACTIVA"

	// SubastaEnConteo indica que el martillero inició la cuenta regresiva del martillo.
	SubastaEnConteo EstadoSubasta = "EN_CONTEO"

	// SubastaCerrada indica que el martillo cayó y la subasta fue adjudicada o desierta.
	SubastaCerrada EstadoSubasta = "CERRADA"
)

// EsAceptandoPujas retorna true si el estado actual permite recibir nuevas ofertas.
func (e EstadoSubasta) EsAceptandoPujas() bool {
	return e == SubastaActiva || e == SubastaEnConteo
}

// String retorna la representación textual del estado.
func (e EstadoSubasta) String() string {
	return string(e)
}
