# Especificación: Concurrencia distribuida con Redis

> **Feature ID:** `003-redis-concurrency`
> **Estado:** En implementación para el MVP tradicional

## Propósito

Evitar que dos procesos acepten simultáneamente pujas para la misma subasta
antes de llegar a la transacción de PostgreSQL.

## Reglas

- El lock se identifica por `subastaID`.
- La adquisición usa `SET NX` con TTL.
- Cada lock tiene un valor aleatorio propio.
- Solo el propietario del lock puede liberarlo mediante un script Lua.
- Un fallo de adquisición rechaza la puja sin modificar PostgreSQL.
- El TTL evita locks permanentes si el proceso se interrumpe.

Redis coordina la contención; PostgreSQL continúa siendo la fuente de verdad y
mantiene la transacción de actualización e historial.

