# Tareas: Escalabilidad Horizontal, Caché Distribuida y Alta Disponibilidad de BD

> **Estado:** Completado

- [x] Implementar configuración de `pgxpool` y soporte Read Replica (`READ_DATABASE_URL`).
- [x] Implementar `CachedAuctionRepository` (Cache-Aside en Redis) e invalidación multi-instancia de usuarios.
- [x] Agregar pruebas unitarias de caché distribuida y degradación ante fallos de Redis.
- [x] Configurar Nginx Load Balancer + entrega de SPA con `Cache-Control` + 2 réplicas `livebid-api` + `PgBouncer` en `docker-compose.yml`.
- [x] Validar `go vet ./...`, `go test ./...` y sintaxis de `docker compose config`.
