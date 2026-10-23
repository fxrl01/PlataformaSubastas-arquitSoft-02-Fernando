# Tareas de Implementación: Billetera de Custodia (Escrow) y Evaluación de Riesgo con IA

> **Feature ID:** `002-escrow-wallet-ai`  
> **Fase:** Plan de Tareas de Implementación (SDD)  
> **Basado en:** `spec.md` y `plan.md`

---

## Estado General del Feature: [x] COMPLETADO Y CERTIFICADO (7/7 tareas completadas)

---

### Fase 1: Dominio y Contratos de Puertos
- [x] **Task 1.1: Crear modelo de entidad `Garantia` y estados de custodia**
  - *Archivo:* `tecnologia/boilerplate-clean-arquitecture/src/app/dominio/modelos/garantia.modelo.ts`
  - *Acción:* Implementada entidad `Garantia` con métodos `capturar()` y `liberar()` y validación de invariantes.
  - *Certificado:* Aprobado en Test 1 y Test 4.

- [x] **Task 1.2: Definir contratos de puertos para IA y Pasarela**
  - *Archivo:* `tecnologia/boilerplate-clean-arquitecture/src/app/dominio/contratos/agente-riesgo.contrato.ts`
  - *Archivo:* `tecnologia/boilerplate-clean-arquitecture/src/app/dominio/contratos/pasarela-bancaria.contrato.ts`
  - *Archivo:* `tecnologia/boilerplate-clean-arquitecture/src/app/dominio/contratos/custodia-garantias.contrato.ts`
  - *Acción:* Interfaces fuertemente tipadas sin dependencias externas.
  - *Certificado:* Aprobado en suite completa.

---

### Fase 2: Capa de Aplicación (Casos de Uso)
- [x] **Task 2.1: Implementar `RetenerGarantiaCasoUso`**
  - *Archivo:* `tecnologia/boilerplate-clean-arquitecture/src/app/aplicacion/retener-garantia.caso-uso.ts`
  - *Acción:* Orquestación de IA, ajuste dinámico (10% vs 20%), preautorización bancaria y persistencia.
  - *Certificado:* Aprobado en Tests 1, 2 y 3.

- [x] **Task 2.2: Implementar `LiquidarSubastaCasoUso`**
  - *Archivo:* `tecnologia/boilerplate-clean-arquitecture/src/app/aplicacion/liquidar-subasta.caso-uso.ts`
  - *Acción:* Captura al postor ganador y liberación en masa inmediata a los perdedores.
  - *Certificado:* Aprobado en Test 4.

---

### Fase 3: Infraestructura, Circuit Breaker e IA
- [x] **Task 3.1: Implementar `CircuitBreaker` para la API de IA**
  - *Archivo:* `tecnologia/boilerplate-clean-arquitecture/src/app/infraestructura/circuit-breaker-ia.ts`
  - *Acción:* Manejo de estados `CLOSED`, `OPEN`, `HALF_OPEN` con fallback heurístico local.
  - *Certificado:* Aprobado en Test 3 ante simulación de caída de Gemini.

- [x] **Task 3.2: Implementar adaptadores en memoria para pasarela y repositorios**
  - *Archivo:* `tecnologia/boilerplate-clean-arquitecture/src/app/infraestructura/gemini-riesgo.adapter.ts`
  - *Archivo:* `tecnologia/boilerplate-clean-arquitecture/src/app/infraestructura/pasarela-bancaria-simulada.ts`
  - *Archivo:* `tecnologia/boilerplate-clean-arquitecture/src/app/infraestructura/repositorio-garantias-memoria.ts`
  - *Certificado:* Aprobado en suite completa.

---

### Fase 4: Suite de Verificación Automatizada (Tests)
- [x] **Task 4.1: Crear suite de pruebas `garantia-ia.pruebas.ts`**
  - *Archivo:* `tecnologia/boilerplate-clean-arquitecture/src/app/pruebas/garantia-ia.pruebas.ts`
  - *Resultados de Certificación:*
    - ✅ Test 1: Retención estándar con IA favorable (Riesgo Bajo, 10% = S/. 5,000 retenidos).
    - ✅ Test 2: Alerta de riesgo anómalo (Riesgo Alto, 20% = S/. 10,000 retenidos).
    - ✅ Test 3: Resiliencia ante caída de Gemini (Circuit Breaker OPEN + Fallback Heurístico Local).
    - ✅ Test 4: Liquidación post-martillazo (Captura al ganador + liberación inmediata a perdedores).
  - *Ejecución:* `node --experimental-strip-types src/app/pruebas/garantia-ia.pruebas.ts` -> **100% OK**.

