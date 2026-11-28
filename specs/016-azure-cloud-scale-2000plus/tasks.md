# Tareas: Infraestructura Microsoft Azure + Cloudflare Edge y Escala 2,000+ Usuarios

> **Estado:** Completado

- [x] Configurar `Nginx` para alta concurrencia (`16,384` conexiones por worker, `least_conn`, `keepalive 256` y soporte `CF-Connecting-IP`).
- [x] Configurar `ulimits` (`65,535`), `PgBouncer` (`2,500` clientes / pool `40`), `Redis` (`10,000` clientes / pool `128`) en `docker-compose.yml` y `cmd/server/main.go`.
- [x] Añadir keepalive `Ping/Pong` cada `30 s` en `websocket_handler.go` para sostener conexiones `WSS` largas detrás de Cloudflare.
- [x] Incorporar perfiles `ci`, `scale2000` y `stress5000` en `tests/load/livebid-load.js`.
- [x] Migrar referencias arquitectónicas de AWS/S3 a **Microsoft Azure (`Standard_B2s` / `Azure Blob Storage`) + Cloudflare Edge** e incorporar `ADR-015`.
- [x] Validar `go vet`, `go test -race`, `nginx -t`, `docker compose` y `Playwright E2E`, registrando commits locales SDD.
