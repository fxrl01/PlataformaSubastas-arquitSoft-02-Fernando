# Plan: Infraestructura Microsoft Azure + Cloudflare Edge y Escala 2,000+ Usuarios

1. Afinar `docker/nginx/nginx.conf` (`worker_rlimit_nofile 65535`, `worker_connections 16384`, `least_conn`, `keepalive 256`, cabecera `CF-Connecting-IP`).
2. Afinar `docker-compose.yml` (`x-high-scale-ulimits`, `PgBouncer MAX_CLIENT_CONN=2500` / `DEFAULT_POOL_SIZE=40`, `Redis --maxclients 10000`, `DB_MAX_CONNS=40`, `REDIS_POOL_SIZE=128`).
3. Actualizar `cmd/server/main.go` (pool Redis de alta concurrencia y timeouts HTTP seguros) y `websocket_handler.go` (ticker `PingMessage` de `30 s` + `PongHandler` compatible con Cloudflare WSS).
4. Parametrizar `tests/load/livebid-load.js` con los perfiles `ci` (`155 VUs`), `scale2000` (`2,100 VUs`) y `stress5000` (`5,000 VUs`).
5. Actualizar documentos de análisis (`01-actores.md`, `04-atributos-de-calidad.md`, `05-restricciones.md`, `07-decisiones-arquitectonicas.md` con `ADR-015`, `08-escenarios-atributos-calidad.md`), arquitectura (`arquitectura-inicial.md`, `componentes-arquitectonicos.md`, `livebid-arquitectura.architecture.json`, `livebid-arquitectura.html`) y C4 (`Nivel1`, `Nivel2`) a **Microsoft Azure + Cloudflare Edge**.
