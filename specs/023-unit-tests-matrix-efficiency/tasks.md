# Plan y Tareas SDD 023

- [x] Optimizar `AuctionRoomHub` (lock `RWMutex` por sala) e `IPRateLimiter` (16 *shards* FNV-1a).
- [x] Agregar pruebas unitarias y benchmarks (`social_handler_test.go`, `room_hub_efficiency_test.go`, `rate_limiter_bench_test.go`, `domain_benchmark_test.go` y `eficiencia-casos-borde.spec.ts`).
- [x] Crear `.github/workflows/unit-tests-efficiency.yml` habilitado para todas las ramas (`branches: ['**']`).
- [x] Verificar localmente con `-race`, `-benchmem` y Jest `100%`, y sincronizar `main` y `release-publica`.
