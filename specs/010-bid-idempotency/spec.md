# Especificación: Idempotencia de pujas

> **Feature ID:** `010-bid-idempotency`
> **Estado:** Implementado para el MVP tradicional

## Alcance

- Aceptar la cabecera opcional `Idempotency-Key` al crear una puja.
- Repetir la misma respuesta sin insertar una segunda puja.
- Rechazar la reutilización de una clave con otro monto.
- Persistir la clave con un índice único por subasta y postor.

La ausencia de la cabecera conserva el comportamiento existente.
