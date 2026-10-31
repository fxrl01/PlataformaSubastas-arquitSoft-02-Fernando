package domain

// CalcularIncrementoMinimo determina el incremento mínimo que debe superar una nueva
// puja en función del precio actual de la subasta, organizado por tramos.
//
// Tramos de incremento (configurable por negocio):
//
//	Precio actual           | Incremento mínimo
//	< 1,000                 |   50
//	1,000 – 9,999           |  100
//	10,000 – 49,999         |  500
//	50,000 – 99,999         | 1,000
//	≥ 100,000               | 2,500
func CalcularIncrementoMinimo(precioActual float64) float64 {
	switch {
	case precioActual < 1_000:
		return 50
	case precioActual < 10_000:
		return 100
	case precioActual < 50_000:
		return 500
	case precioActual < 100_000:
		return 1_000
	default:
		return 2_500
	}
}
