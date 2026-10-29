# Tareas de Implementación: Motor de Pujas en Tiempo Real

> **Feature ID:** `001-realtime-bid-engine`  
> **Implementación objetivo:** `tecnologia/livebid-backend`
> **Estado:** En progreso; no certificada para producción

## Dominio y aplicación

- [x] Implementar reglas de subasta en
  `internal/core/domain/auction.go`.
- [x] Implementar el servicio de aplicación en
  `internal/core/services/auction_service.go`.
- [ ] Completar errores públicos y mapeo HTTP/WebSocket.

## Adaptadores reales

- [ ] Implementar `BidLockPort` con Redis `SET NX PX` y liberación Lua.
- [ ] Implementar repositorio PostgreSQL para subastas y pujas.
- [ ] Implementar publicación de eventos y broadcast WebSocket.
- [ ] Integrar `PaymentEscrowPort` con la feature 002.
- [ ] Añadir autenticación y obtener `PostorID` desde el contexto autenticado.

## Verificación

- [x] Cubrir reglas de dominio con pruebas unitarias Go.
- [ ] Probar concurrencia contra Redis real.
- [ ] Probar persistencia y auditoría contra PostgreSQL.
- [ ] Validar los contratos de API y las métricas de latencia.

Las pruebas del boilerplate TypeScript quedan como referencia histórica y no
marcan esta feature como completada.
