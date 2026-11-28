# Especificación: Infraestructura Microsoft Azure + Cloudflare Edge y Escala 2,000+ Usuarios

> **Feature ID:** `016-azure-cloud-scale-2000plus`  
> **Estado:** Completado

## Alcance

- **Estandarización Cloud sobre Microsoft Azure + Cloudflare Edge:** Alineación integral de documentos de análisis (`AC01`–`AC08`, `RC07`, `ADR-015`, `EQ-01`), arquitectura de software, visor interactivo y diagramas C4 (Niveles 1 y 2) adoptando **Microsoft Azure (`Azure for Students`, Ubuntu VM `Standard_B2s` 2 vCPU / 4 GB RAM, escalable a `B4ms` / VMSS y `Azure Blob Storage`)** protegido en el borde por **Cloudflare (`DNS`, `CDN`, `HTTPS` y `WSS`)**.
- **Tuning de Sistema Operativo, Proxy, Pool y WebSockets para `2,000+` a `5,000+` Usuarios:**
  - `Nginx`: `worker_rlimit_nofile 65535`, `worker_connections 16384`, `multi_accept on`, `least_conn`, `keepalive 256` y preservación de IP real desde `CF-Connecting-IP`.
  - `Docker Compose & PgBouncer & Redis`: `ulimits.nofile = 65535`, `MAX_CLIENT_CONN = 2500`, `DEFAULT_POOL_SIZE = 40`, `DB_MAX_CONNS = 40`, `REDIS_POOL_SIZE = 128` y `--maxclients 10000`.
  - `Go WebSocket Hub`: Keepalive `PingMessage` cada `30 s` y `PongHandler` (`75 s`) para evitar desconexión de espectadores silenciosos por el *idle timeout* de `100 s` de Cloudflare WSS.
- **Perfiles Parametrizables en `k6` (`ci`, `scale2000`, `stress5000`):** Extensión de `tests/load/livebid-load.js` para ejecutar tanto validación rápida CI (`155 VUs`) como pruebas reales de producción a `2,100 VUs` (`1,600 WS + 450 Catálogo + 50 Pujadores`) y estrés futuro a `5,000 VUs`.

## Criterios de Aceptación

- **Dado** el despliegue detrás de Cloudflare y Nginx, **cuando** `2,000+` espectadores mantienen conexiones `WSS` abiertas, **entonces** Nginx dispone de `16,384` conexiones por worker, `ulimits` de `65,535` descriptores y latidos `Ping/Pong` cada `30 s` sin cortes por inactividad.
- **Dado** el comando `k6 run -e PROFILE=scale2000` (o `stress5000`), **cuando** se lanza desde un generador externo hacia el entorno Azure, **entonces** ejecuta las rampas de `2,100` (o `5,000`) VUs y exporta el perfil activo en `k6-report.html` y `k6-summary.json`.
- **Dado** la documentación de análisis, arquitectura y C4 del repositorio, **cuando** se auditan referencias de infraestructura cloud, **entonces** toda la arquitectura refleja consistentemente **Microsoft Azure + Cloudflare Edge** sin referencias residuales a AWS/S3.
