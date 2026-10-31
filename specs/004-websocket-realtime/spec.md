# Especificación: Difusión en tiempo real por WebSocket

> **Feature ID:** `004-websocket-realtime`
> **Estado:** En implementación para el MVP tradicional

## Alcance

- Conectar clientes autenticados a una sala de subasta.
- Difundir pujas confirmadas por PostgreSQL.
- Usar Redis Pub/Sub para distribuir eventos entre instancias del backend.
- Cerrar la conexión cuando el cliente no envíe mensajes de control válidos.

El WebSocket no procesa pujas ni modifica PostgreSQL. Las pujas continúan
entrando por el endpoint HTTP protegido.

## Endpoint

```text
GET /ws/subastas/{id}
Authorization: Bearer <JWT>
```

## Evento difundido

```json
{
  "type": "bid.accepted",
  "auction_id": "sub_...",
  "bid_id": "bid_...",
  "bidder_id": "usr_...",
  "amount": 2750,
  "extended": false,
  "close_at": "2026-11-01T12:00:00-05:00"
}
```

