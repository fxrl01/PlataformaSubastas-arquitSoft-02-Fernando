# Plan técnico: Catálogo y ciclo básico de subastas

## Capas

```text
HTTP handlers -> AuctionCatalogService -> SubastaRepository
                                           -> PostgreSQL adapter
```

El dominio mantiene la máquina de estados. El servicio de aplicación valida
identidad, fechas y permisos de negocio; el adaptador PostgreSQL traduce la
persistencia sin filtrar detalles de SQL al dominio.

## Endpoints

```text
POST  /api/v1/subastas
GET   /api/v1/subastas
GET   /api/v1/subastas/{id}
PATCH /api/v1/subastas/{id}/activate
PATCH /api/v1/subastas/{id}/close
```

## Decisiones

- La identidad del vendedor proviene del JWT.
- Las consultas de catálogo son públicas.
- La creación requiere rol `VENDEDOR`.
- Activar y cerrar requieren `MARTILLERO` o `ADMIN`.
- La persistencia inicial es PostgreSQL; no se introduce Redis todavía.

