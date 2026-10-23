# Tareas de Implementación: Motor de Pujas en Tiempo Real (Realtime Bid Engine)

> **Feature ID:** `001-realtime-bid-engine`  
> **Fase:** Plan de Tareas de Implementación (SDD)  
> **Basado en:** `spec.md` y `plan.md`

---

## Estado General del Feature: [x] COMPLETADO Y CERTIFICADO (7/7 tareas completadas)

---

### Fase 1: Dominio y Reglas Invariantes
- [x] **Task 1.1: Refinar la entidad `Subasta` con validación de autopuja y Anti-Sniping**
  - *Archivo:* `tecnologia/boilerplate-clean-arquitecture/src/app/dominio/modelos/subasta.modelo.ts`
  - *Acción:* Validación explícita de `POSTOR_YA_ES_LIDER` y extensión automática de $+30$ segundos si `tiempoRestante < 15000 ms`.
  - *Certificado:* Aprobado en Test 3 y Test 5.

- [x] **Task 1.2: Estandarizar catálogo de errores de dominio**
  - *Archivo:* `tecnologia/boilerplate-clean-arquitecture/src/app/dominio/modelos/errores-puja.ts`
  - *Acción:* Creada clase `ErrorDominioPuja` y enum `CodigoErrorPuja` con códigos normalizados.
  - *Certificado:* Aprobado en toda la suite.

---

### Fase 2: Capa de Aplicación y Puertos
- [x] **Task 2.1: Enriquecer el contrato `NotificadorSalaContrato`**
  - *Archivo:* `tecnologia/boilerplate-clean-arquitecture/src/app/dominio/contratos/notificador-sala.contrato.ts`
  - *Acción:* Añadido soporte para `difundirTiempoExtendido` y bandera `antiSnipingActivado`.
  - *Certificado:* Aprobado en Test 2 y Test 5.

- [x] **Task 2.2: Actualizar `ProcesarPujaCasoUso` con los nuevos códigos de error**
  - *Archivo:* `tecnologia/boilerplate-clean-arquitecture/src/app/aplicacion/procesar-puja.caso-uso.ts`
  - *Acción:* Mapeo de `codigoError`, retorno de metadatos de cronómetro y liberación segura de lock en `finally`.
  - *Certificado:* Aprobado en suite completa.

---

### Fase 3: Capa de Infraestructura
- [x] **Task 3.1: Robustecer el adaptador `MotorPujasSimulado` con Mutex Atómico y TTL**
  - *Archivo:* `tecnologia/boilerplate-clean-arquitecture/src/app/infraestructura/motor-pujas-simulado.ts`
  - *Acción:* Simulación del comando Redis `SETNX` y comprobación de contención concurrente.
  - *Certificado:* Aprobado en Test 6.

- [x] **Task 3.2: Actualizar el adaptador de notificación en consola**
  - *Archivo:* `tecnologia/boilerplate-clean-arquitecture/src/app/infraestructura/notificador-sala-consola.ts`
  - *Acción:* Registro visual en consola de alertas Anti-Sniping y nuevos precios líderes.
  - *Certificado:* Aprobado en ejecuciones de consola.

---

### Fase 4: Suite de Verificación Automatizada (Tests)
- [x] **Task 4.1: Crear suite de pruebas exhaustivas para los 5 criterios de aceptación**
  - *Archivo:* `tecnologia/boilerplate-clean-arquitecture/src/app/pruebas/subasta.pruebas.ts`
  - *Resultados de Certificación:*
    - ✅ Test 1: Rechazo Zero Trust sin depósito en garantía (Status 403 / `GARANTIA_REQUERIDA`).
    - ✅ Test 2: Puja válida con garantía activa (Status 200 / Broadcast emitido).
    - ✅ Test 3: Rechazo de autopuja del postor líder (Status 400 / `POSTOR_YA_ES_LIDER`).
    - ✅ Test 4: Rechazo por incremento insuficiente (Status 409 / `MONTO_INSUFICIENTE`).
    - ✅ Test 5: Activación de regla Anti-Sniping (Cronómetro extendido $+30$ segundos).
    - ✅ Test 6: Detección y rechazo de colisión concurrente (Status 503 / `COLISION_CONCURRENTE`).
  - *Ejecución:* `node --experimental-strip-types src/app/pruebas/subasta.pruebas.ts` -> **100% OK**.

