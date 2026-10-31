# Especificación: Eventos de cierre de subasta

> **Feature ID:** `008-auction-closure-events`
> **Estado:** Implementado para el MVP tradicional

## Alcance

- Publicar `auction.closed` por Redis Pub/Sub al cerrar una subasta.
- Difundir el evento a los clientes conectados al WebSocket de la subasta.
- Incluir el ganador y monto final cuando exista.
- Publicar un cierre desierto sin exponer datos sensibles cuando no exista ganador.

## Evento

```json
{
  "type": "auction.closed",
  "auction_id": "sub_123",
  "winner_id": "usr_456",
  "amount": 2750,
  "closed_at": "2026-10-31T23:00:00Z"
}
```
