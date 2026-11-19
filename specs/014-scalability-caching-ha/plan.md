# Plan: Escalabilidad Horizontal, Caché Distribuida y Alta Disponibilidad de BD

1. Añadir `db_pool.go` con configuración explícita de `pgxpool` y soporte de `READ_DATABASE_URL` en `PostgresAuctionRepository` y `PostgresBidRepository`.
2. Implementar `CachedAuctionRepository` (Cache-Aside en Redis + fallback seguro) y `WithRedisInvalidation` en `CachedUserRepository`.
3. Permitir invalidación automática de caché de subasta tras `RegistrarPujaAtomica` desde `AuctionBidService`.
4. Añadir `docker/nginx/nginx.conf` (SPA estática con caché HTTP + upstream balanceado REST/WS) y actualizar `docker-compose.yml` con `livebid-pgbouncer`, `livebid-api-1`, `livebid-api-2` y `livebid-gateway`.
5. Añadir pruebas unitarias para `CachedAuctionRepository`, pooling e invalidación distribuida.
