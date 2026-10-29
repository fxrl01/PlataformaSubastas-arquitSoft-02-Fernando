package domain

import "time"

// Puja representa una oferta inmutable realizada por un postor en una subasta.
// Es un Value Object: una vez creada, sus campos no deben modificarse.
type Puja struct {
	// ID es el identificador único de la puja (UUID v4).
	ID string

	// SubastaID identifica la subasta a la que pertenece esta puja.
	SubastaID string

	// PostorID identifica al usuario que realizó la oferta.
	PostorID string

	// Monto es el valor monetario ofertado en la divisa de la subasta.
	Monto float64

	// Timestamp es el instante exacto en que el servidor registró la puja.
	Timestamp time.Time
}

// NuevaPuja crea una instancia inmutable de Puja con los valores proporcionados.
func NuevaPuja(id, subastaID, postorID string, monto float64, timestamp time.Time) Puja {
	return Puja{
		ID:        id,
		SubastaID: subastaID,
		PostorID:  postorID,
		Monto:     monto,
		Timestamp: timestamp,
	}
}

