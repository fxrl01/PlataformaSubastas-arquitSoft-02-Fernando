package domain

import "errors"

// Errores de dominio — representan violaciones a las reglas invariantes de negocio.
// Son independientes de HTTP, WebSocket o cualquier protocolo de transporte.
var (
	// ErrPujaInferior se produce cuando el monto ofertado no supera el precio líder
	// más el incremento mínimo exigido por el tramo actual.
	ErrPujaInferior = errors.New("domain: el monto de la puja no supera el precio líder más el incremento mínimo")

	// ErrSubastaCerrada se produce cuando se intenta pujar en una subasta que ya fue
	// adjudicada, cancelada o que aún no ha sido abierta.
	ErrSubastaCerrada = errors.New("domain: la subasta no está en un estado que acepte pujas")

	// ErrGarantiaInsuficiente se produce cuando el postor no tiene un depósito en
	// custodia activo que respalde su participación en la subasta.
	ErrGarantiaInsuficiente = errors.New("domain: el postor no tiene garantía activa para esta subasta")

	// ErrLockTimeout se produce cuando no fue posible adquirir el lock distribuido
	// en Redis dentro del tiempo máximo permitido (alta contención concurrente).
	ErrLockTimeout = errors.New("domain: no se pudo adquirir el lock atómico por contención concurrente")

	// ErrRiesgoAltoIA se produce cuando el agente de inteligencia artificial
	// (Gemini) determina que el comportamiento del postor es sospechoso de fraude.
	ErrRiesgoAltoIA = errors.New("domain: el postor fue bloqueado por evaluación de riesgo de IA")

	// ErrPostorIgual se produce cuando el postor líder actual intenta pujar contra sí mismo.
	ErrPostorIgual = errors.New("domain: el postor líder actual no puede pujar contra sí mismo")

	// ErrSubastaNoEncontrada se produce cuando la subasta solicitada no existe en el repositorio.
	ErrSubastaNoEncontrada    = errors.New("domain: la subasta solicitada no fue encontrada")
	ErrSubastaInvalida        = errors.New("domain: datos de subasta inválidos")
	ErrPujaConcurrente        = errors.New("domain: la subasta cambió mientras se registraba la puja")
	ErrIdempotencyKeyConflict = errors.New("domain: la clave de idempotencia ya fue usada con otro monto")
	ErrInvalidWeb3Wallet      = errors.New("domain: dirección de billetera Web3 inválida")
	ErrInvalidWeb3Signature   = errors.New("domain: firma criptográfica EIP-712 inválida o vencida")
	ErrAgentMandateInvalid    = errors.New("domain: mandato de agente autónomo IA inválido o no encontrado")
	ErrAgentMandateExpired    = errors.New("domain: el mandato del agente autónomo IA ha expirado o fue revocado")
	ErrAgentMandateExceeded   = errors.New("domain: el monto supera el límite máximo autorizado para el agente IA")
)
