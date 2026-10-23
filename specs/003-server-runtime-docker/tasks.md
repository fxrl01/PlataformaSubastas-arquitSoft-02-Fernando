# Tareas de Implementación: Runtime del Servidor, Orquestación y Entorno Docker

> **Feature ID:** `003-server-runtime-docker`  
> **Fase:** Plan de Tareas de Implementación (SDD)  
> **Basado en:** `spec.md` y `plan.md`

---

## Estado General del Feature: [x] COMPLETADO Y CERTIFICADO (6/6 tareas completadas)

---

### Fase 1: Esquema DDL y Docker Compose
- [x] **Task 1.1: Crear esquema de base de datos relacional `docker/init.sql`**
  - *Archivo:* `docker/init.sql`
  - *Acción:* Tablas normalizadas `usuarios`, `subastas`, `pujas`, `garantias` y `logs_auditoria` con índices B-Tree e inserción de datos semilla.
  - *Certificado:* Esquema DDL validado.

- [x] **Task 1.2: Configurar `docker-compose.yml` y `.env.example`**
  - *Archivos:* `docker-compose.yml`, `.env.example`
  - *Acción:* Definición de servicios `livebid-db` (Postgres 16), `livebid-cache` (Redis 7) y `livebid-api` en red interna `livebid-net`.
  - *Certificado:* Sintaxis de Docker Compose v5 validada.

- [x] **Task 1.3: Crear `Dockerfile` para la aplicación**
  - *Archivo:* `Dockerfile`
  - *Acción:* Imagen basada en `node:24-alpine` con soporte nativo para TypeScript strip-types y entrypoint a `src/main.ts`.
  - *Certificado:* Dockerfile estructurado.

---

### Fase 2: Bootstrapping del Servidor (`src/main.ts`)
- [x] **Task 2.1: Implementar punto de entrada `src/main.ts` con Clean Architecture**
  - *Archivo:* `tecnologia/boilerplate-clean-arquitecture/src/main.ts`
  - *Acción:* Cableado de dependencias (DIP), routing HTTP nativo sin dependencias externas pesadas.
  - *Certificado:* Arranque en puerto configurable y Health Check.

- [x] **Task 2.2: Implementar endpoints REST de la API**
  - *Archivo:* `tecnologia/boilerplate-clean-arquitecture/src/main.ts`
  - *Acción:* Mapeo de `GET /health`, `GET /api/v1/subastas/:id`, `POST /api/v1/subastas/:id/garantia`, `POST /api/v1/subastas/:id/puja` y `POST /api/v1/subastas/:id/liquidar`.
  - *Certificado:* Validado en tests HTTP de integración.

---

### Fase 3: Suite de Pruebas de Integración
- [x] **Task 3.1: Crear suite de integración `servidor-integracion.pruebas.ts`**
  - *Archivo:* `tecnologia/boilerplate-clean-arquitecture/src/app/pruebas/servidor-integracion.pruebas.ts`
  - *Resultados de Certificación:*
    - ✅ Test 1: `GET /health` responde status 200 y JSON con estado del sistema.
    - ✅ Test 2: `GET /api/v1/subastas/SUB-2026-001` retorna datos del lote en vivo.
    - ✅ Test 3: `POST /puja` sin garantía es rechazado en HTTP con status 403 (Zero Trust).
    - ✅ Test 4: `POST /garantia` evalúa riesgo con IA y constituye depósito bancario.
    - ✅ Test 5: `POST /puja` con garantía activa es aceptada con status 200 y broadcast.
    - ✅ Test 6: `POST /liquidar` captura fondos del ganador al martillazo final.
  - *Ejecución:* `NODE_ENV=test node --experimental-strip-types src/app/pruebas/servidor-integracion.pruebas.ts` -> **100% OK**.

