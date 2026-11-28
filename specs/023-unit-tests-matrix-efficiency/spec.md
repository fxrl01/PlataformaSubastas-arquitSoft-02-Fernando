# SDD 023 — Eficiencia de Concurrencia, Ampliación de Pruebas Unitarias y Matriz Multi-Rama en GitHub Actions

## 1. Contexto y Objetivo
1. Reducir contención de memoria y CPU bajo alta concurrencia desacoplando los locks por sala en `AuctionRoomHub` (`sync.RWMutex` por subasta) y particionando `IPRateLimiter` en 16 *shards* FNV-1a sin asignaciones dinámicas en ruta caliente (`0 allocs/op`).
2. Ampliar la suite de pruebas unitarias y microbenchmarks (`go test -bench=. -benchmem` y Jest `100%`) cubriendo Social Feed, Vitrina Sanitizada, Chat en Sala, Hub en tiempo real, cabeceras/IPs de seguridad y casos borde de Anti-Sniping.
3. Incorporar un workflow dedicado ultrarrápido (`.github/workflows/unit-tests-efficiency.yml`) que ejecute pruebas unitarias y benchmarks en paralelo sobre **cualquier rama (`branches: ['**']`)** y Pull Request en menos de 40 segundos.

## 2. Criterios de Aceptación
- `AuctionRoomHub.fanOut` difunde mensajes con `room.mu.RLock()` sin bloquear otras salas activas.
- `IPRateLimiter` opera sobre 16 *shards* independientes con `0 allocs/op` tras calentamiento.
- Nuevas suites unitarias incrementan la cobertura de `handlers`, `realtime` y `security` en Go y amplían casos borde en Jest manteniendo `100%` global.
- `.github/workflows/unit-tests-efficiency.yml` valida automáticamente cualquier rama (`feature/*`, `fix/*`, `release-publica`, `main`) y publica tabla de nanosegundos/memoria en `$GITHUB_STEP_SUMMARY`.
