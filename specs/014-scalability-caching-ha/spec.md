# Especificación: Escalabilidad Horizontal, Caché Distribuida y Alta Disponibilidad de BD

> **Feature ID:** `014-scalability-caching-ha`  
> **Estado:** En Implementación

## Alcance

- **Escalabilidad Frontend y Backend:** Despliegue con proxy inverso / balanceador de carga (**Nginx**) sirviendo la SPA estática con cabeceras `Cache-Control`/`ETag` y balanceando tráfico HTTP REST y WebSockets (`/ws/`) entre **2+ réplicas stateless** de `livebid-api` (`least_conn`).
- **Escalabilidad de Base de Datos:** Pool de conexiones `pgxpool` parametrizable (`MaxConns`, `MinConns`, `MaxConnLifetime`), separación opcional de lectura (`READ_DATABASE_URL` para consultas de catálogo e historial) e incorporación de **PgBouncer** en modo pooling transaccional en Docker Compose.
- **Caché Distribuida (Redis Cache-Aside):** Decorador `CachedAuctionRepository` sobre Redis para `Listar` y `ObtenerPorID` con TTL corto, invalidación inmediata en `Guardar` y tras pujas aceptadas, más invalidación distribuida vía Redis Pub/Sub para `CachedUserRepository` entre réplicas.

## Criterios de Aceptación

- **Dado** un entorno con múltiples réplicas de `livebid-api`, **cuando** llega tráfico REST y WebSocket a través de Nginx, **entonces** las peticiones se distribuyen entre instancias y los mensajes de sala en vivo se sincronizan mediante Redis Pub/Sub.
- **Dado** una consulta repetida al catálogo o detalle de subasta, **cuando** el registro está en caché Redis, **entonces** se responde desde Redis sin consultar PostgreSQL; y **cuando** cambia la subasta o entra una puja, la clave se invalida de inmediato.
- **Dado** que Redis falla o no está disponible, **cuando** se consulta o actualiza el catálogo, **entonces** el sistema degrada transparentemente hacia PostgreSQL sin error para el usuario.
