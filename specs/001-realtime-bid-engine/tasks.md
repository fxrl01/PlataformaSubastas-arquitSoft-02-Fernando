# Tareas de Implementación: Motor de Pujas en Tiempo Real (Realtime Bid Engine)

> **Feature ID:** `001-realtime-bid-engine`  
> **Fase:** Plan de Tareas de Implementación (SDD)  
> **Basado en:** `spec.md` y `plan.md`

---

## Estado General del Feature: [ ] En Progreso (0/7 completadas)

---

### Fase 1: Dominio y Reglas Invariantes
- [ ] **Task 1.1: Refinar la entidad `Subasta` con validación de autopuja y Anti-Sniping**
  - *Archivo:* `tecnologia/boilerplate-clean-arquitecture/src/app/dominio/modelos/subasta.modelo.ts`
  - *Acción:* Agregar validación explícita para evitar que el postor líder actual se autopuje (`POSTOR_YA_ES_LIDER`).
  - *Acción:* Implementar la extensión dinámica de tiempo (+30 segundos) si `tiempoRestante < 15000 ms`.
  - *Dependencias:* Ninguna.

- [ ] **Task 1.2: Estandarizar catálogo de errores de dominio**
  - *Archivo:* `tecnologia/boilerplate-clean-arquitecture/src/app/dominio/modelos/errores-puja.ts`
  - *Acción:* Crear tipos de error enumerados (`MONTO_INSUFICIENTE`, `SUBASTA_CERRADA`, `GARANTIA_REQUERIDA`, `POSTOR_YA_ES_LIDER`, `COLISION_CONCURRENTE`).
  - *Dependencias:* Task 1.1.

---

### Fase 2: Capa de Aplicación y Puertos
- [ ] **Task 2.1: Enriquecer el contrato `NotificadorSalaContrato`**
  - *Archivo:* `tecnologia/boilerplate-clean-arquitecture/src/app/dominio/contratos/notificador-sala.contrato.ts`
  - *Acción:* Añadir soporte para emitir evento `tiempoExtendido` cuando se activa la regla Anti-Sniping.
  - *Dependencias:* Task 1.1.

- [ ] **Task 2.2: Actualizar `ProcesarPujaCasoUso` con los nuevos códigos de error**
  - *Archivo:* `tecnologia/boilerplate-clean-arquitecture/src/app/aplicacion/procesar-puja.caso-uso.ts`
  - *Acción:* Devolver `codigoError` estructurado en el objeto de respuesta para mapeo directo a eventos WebSocket.
  - *Dependencias:* Task 1.2, Task 2.1.

---

### Fase 3: Capa de Infraestructura
- [ ] **Task 3.1: Robustecer el adaptador `MotorPujasSimulado` con Mutex Atómico y TTL**
  - *Archivo:* `tecnologia/boilerplate-clean-arquitecture/src/app/infraestructura/motor-pujas-simulado.ts`
  - *Acción:* Simular comportamiento real de Redis `SETNX` con soporte para timeout de expiración automática.
  - *Dependencias:* Task 2.2.

- [ ] **Task 3.2: Actualizar el adaptador de notificación en consola**
  - *Archivo:* `tecnologia/boilerplate-clean-arquitecture/src/app/infraestructura/notificador-sala-consola.ts`
  - *Acción:* Registrar visualmente en los logs las alertas de Anti-Sniping y los eventos de unicast/broadcast.
  - *Dependencias:* Task 2.1.

---

### Fase 4: Suite de Verificación Automatizada (Tests)
- [ ] **Task 4.1: Crear suite de pruebas exhaustivas para los 5 criterios de aceptación**
  - *Archivo:* `tecnologia/boilerplate-clean-arquitecture/src/app/pruebas/subasta.pruebas.ts`
  - *Casos a certificar:*
    - Test 1: Puja válida estándar.
    - Test 2: Bloqueo Zero Trust sin garantía.
    - Test 3: Rechazo de monto insuficiente.
    - Test 4: Rechazo de autopuja del postor líder.
    - Test 5: Activación de la regla Anti-Sniping (verificación de nuevo tiempo de cierre).
    - Test 6: Simulación de ráfaga concurrente paralela (Anti-Race Condition).
  - *Verificación:* Ejecutar con `node --experimental-strip-types` y asegurar 100% de aserciones aprobadas.
  - *Dependencias:* Tasks 1.1 a 3.2.

