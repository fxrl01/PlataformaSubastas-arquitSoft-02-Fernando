# Plan Técnico SDD 019: Chat WS, Vitrina de Vendedor, Campana Popover y Hardening HTTP

## 1. Dominio, Persistencia y Tiempo Real (Go)
- `internal/adapters/persistence/schema_migrations.go`: tabla `sala_mensajes` con índice `(subasta_id, created_at DESC)` y columna opcional `vendedor_id` en `subastas` / cruce por consignante.
- `internal/adapters/realtime/redis_bid_notifier.go`: método `BroadcastEventoSocialSala` emitiendo `{"type":"room.social", ...}` en el canal Redis de la sala.
- `internal/adapters/handlers/social_handler.go`:
  - `GET /api/v1/vendedores/{id}/vitrina` (perfil público sanitizado + lotes + posts + total likes).
  - `GET /api/v1/subastas/{id}/mensajes` y `POST /api/v1/subastas/{id}/mensajes` (`RequireAuth`).
- `internal/adapters/security/http_hardening.go`:
  - `IPRateLimiter` concurrente por IP.
  - `WithSecurityHeadersAndBodyLimit` (`MaxBytesReader`, cabeceras OWASP, CORS configurable).
- `cmd/server/main.go` & `docker-compose.yml`:
  - `RequireAuth` en `POST /api/v1/social/posts/{id}/like`.
  - Control `SWAGGER_ENABLED` y validación de `ADMIN_BOOTSTRAP_PASSWORD` en `APP_ENV=production`.
  - Bind `127.0.0.1` en puertos host de Postgres y Redis.

## 2. Frontend Moderno sin Errores (`public/`)
- Campana `#btnNotificationsBell` con `<div id="notificationsPopover" popover="auto">` posicionado mediante `viewport-aware-dropdowns` (`@supports (position-area: block-end)` + fallback fijo).
- Modal `<dialog id="sellerStorefrontDialog">` conectado a `GET /api/v1/vendedores/{id}/vitrina`.
- Bloque de Reacciones (`🔥`, `👏`, `💎`, `🎉`) y Chat en Vivo dentro de `#viewSala` sincronizado con WebSocket.
