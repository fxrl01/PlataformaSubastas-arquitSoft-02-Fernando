# Tareas de Implementación: Billetera de Custodia y Riesgo con IA

> **Feature ID:** `002-escrow-wallet-ai`  
> **Basado en:** `spec.md` y `plan.md`
> **Estado:** En progreso; no certificada para producción

## Fase 1: Dominio y puertos

- [ ] Definir `Garantia` y sus transiciones en
  `tecnologia/livebid-backend/internal/core/domain`.
- [ ] Completar `PaymentEscrowPort` con retención, captura, liberación e
  idempotencia.
- [ ] Definir errores tipados para fondos insuficientes, garantía duplicada y
  subasta no disponible.
- [ ] Separar el caso de uso de garantías del `AuctionService` sin acoplar el
  dominio a proveedores externos.

## Fase 2: Riesgo y resiliencia

- [ ] Implementar `AIRiskPort` con adaptador HTTP real para Gemini.
- [ ] Implementar circuit breaker `CLOSED`, `OPEN` y `HALF_OPEN`.
- [ ] Aplicar timeout de 2 segundos y apertura después de 3 fallos.
- [ ] Implementar fallback heurístico determinista y observabilidad del evento
  `CIRCUIT_BREAKER_FALLBACK_ACTIVATED`.

## Fase 3: Infraestructura

- [ ] Implementar repositorios PostgreSQL para garantías y auditoría.
- [ ] Implementar adaptador de pasarela bancaria en entorno sandbox.
- [ ] Garantizar idempotencia y consistencia de la liquidación.
- [ ] Implementar configuración segura mediante variables de entorno.

## Fase 4: API y pruebas

- [ ] Implementar handlers Go para retención y liquidación.
- [ ] Añadir pruebas unitarias de dominio y casos de uso.
- [ ] Añadir pruebas de integración con PostgreSQL y el sandbox bancario.
- [ ] Añadir pruebas de timeout, recuperación del circuito e idempotencia.
- [ ] Actualizar la certificación solo después de pasar las pruebas de
  integración.
