# Plan: Gobernanza RBAC, Encargado Oficial y Ficha del Lote

1. Extender entidades `User` y `Subasta` (`EncargadoSubasta`, `DetallesLote`) en `internal/core/domain`.
2. Agregar migraciones idempotentes en `schema_migrations.go` y columnas `JSONB` en `docker/init.sql`.
3. Exponer endpoints RBAC de administración, encargado y detalles del lote en `auth_handler.go` y `auction_handler.go`.
4. Adaptar la SPA (`index.html`, `app.js`, `styles.css`) con vistas segregadas por rol y selector de imágenes.
5. Cubrir con pruebas unitarias de `AuthService`, `AuctionCatalogService` y `CachedUserRepository`.
