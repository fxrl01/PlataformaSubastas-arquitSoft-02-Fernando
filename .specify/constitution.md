# Constitución de Ingeniería — LiveBid

> **Proyecto:** LiveBid — Plataforma de Subastas en Tiempo Real  
> **Estándar:** GitHub Spec Kit · Spec-Driven Development (SDD)  
> **Versión:** 2.0.0 · **Estado:** VIGENTE E INMUTABLE

---

## 1. Declaración de Principios Innegociables

Esta Constitución establece las reglas arquitectónicas y de ingeniería fundamentales que rigen el desarrollo de **LiveBid**. Todo agente de IA, desarrollador o arquitecto debe adherirse a estos principios sin excepción. Ninguna tarea de código será considerada aprobada si viola alguno de los artículos constitucionales.

---

### Artículo I: Monolito Modular Hexagonal e Inversión de Dependencias
1. **Aislamiento del Dominio:** El núcleo de negocio (`internal/core/domain`) no debe importar ni conocer frameworks web, librerías WebSocket, drivers de base de datos (`pgx`, `redis`), ni SDKs externos (`Stripe`, `Gemini`, `Web3`).
2. **Regla de Dependencia Unidireccional:** El flujo de dependencias apunta siempre hacia adentro:  
   `Adaptadores (HTTP/WS, Postgres, Redis, Stripe, IA, Web3)` $\rightarrow$ `Servicios de Aplicación (internal/core/services)` $\rightarrow$ `Dominio Puro (internal/core/domain)`.
3. **Puertos y Adaptadores:** Toda comunicación con el exterior se define mediante interfaces (`ports.go`) en el dominio y se implementa en `internal/adapters/*` bajo un único binario compilado en **Go 1.23+**.

---

### Artículo II: Principio Zero Trust, RBAC Estricto y Custodia Previa (Escrow)
1. **Cero Confianza Implícita:** Ninguna solicitud REST o WebSocket es confiable por defecto. Cada ruta protegida exige un `JWT (HS256)` válido y verifica en tiempo real rol (`ADMIN`, `MARTILLERO`, `VENDEDOR`, `POSTOR`) y estado `ACTIVO` mediante `RequireAuth` y `RequireRoles`.
2. **Garantía Financiera Obligatoria (PCI-DSS):** Ningún postor puede pujar en una subasta activa sin una retención preventiva (`PaymentIntent` con `capture_method: manual` en **Stripe** o colateral Web3). El backend y PostgreSQL jamás almacenan datos sensibles de tarjeta (`PAN`/`CVV`).
3. **Liquidación Automática al Cierre:** Al finalizar la subasta (`Cerrar` o `AuctionScheduler`), la garantía del ganador se captura (`CAPTURADA_COBRO`) y las de los demás postores se liberan (`LIBERADA`), sincronizando eventos firmados vía Stripe Webhooks.

---

### Artículo III: Concurrencia Atómica, Trazabilidad y Fuente de Verdad Legal
1. **El Orden es el Producto:** Toda puja se resuelve de forma estrictamente secuencial y atómica combinando mutex distribuido en Redis (`SETNX` con TTL 1.5s y liberación Lua), bloqueo pesimista `SELECT ... FOR UPDATE` en PostgreSQL 16 e `Idempotency-Key`.
2. **Timestamp del Servidor y Anti-Sniping:** La única fuente legal de tiempo es el reloj del servidor Go. Toda puja válida dentro de los últimos `15 segundos` extiende el cierre en `+30 segundos`.
3. **Cadena Criptográfica SHA-256 y Merkle Tree:** Cada evento crítico y cada puja se encadenan con `SHA-256` en `logs_auditoria`, permitiendo generar certificados verificables (`MerkleRoot` y `MerkleProof`).

---

### Artículo IV: Persistencia Única ACID, Bus Efímero y Estrategia de Caché
1. **Única Base de Datos Principal (Single Source of Truth):** **PostgreSQL 16** almacena de forma transaccional (`ACID`) usuarios, subastas (`encargado_json`, `detalles_lote_json` en `JSONB`), garantías, pujas y auditoría, escalado mediante pooling (`pgxpool` + `PgBouncer`) y soporte de réplica de lectura (`READ_DATABASE_URL`).
2. **Almacén Caliente Efímero (Redis 7):** Coordina exclusivamente locks distribuidos (`SETNX`), difusión Pub/Sub $O(1)$ hacia el `AuctionRoomHub` y **caché distribuida (`Cache-Aside`)** para catálogo de subastas e invalidación multi-instancia de usuarios/subastas. Si Redis falla, el backend degrada con seguridad hacia PostgreSQL.
3. **Escalabilidad Horizontal Stateless:** Múltiples réplicas de `livebid-api` corren detrás de un balanceador de carga (**Nginx**) con soporte nativo para WebSockets y entrega desacoplada de activos estáticos con cabeceras `Cache-Control` / `ETag`.

---

### Artículo V: Presupuestos de Calidad, Resiliencia IA y SLAs

| Métrica de Rendimiento / Resiliencia | Umbral Máximo Permitido |
|---|---|
| **Latencia de confirmación de puja (P95)** | $\le 200$ ms |
| **Difusión de nuevo precio líder (Broadcast)** | $\le 300$ ms a $1,000+$ postores concurrentes |
| **Timeout máximo hacia IA externa (Gemini)** | $\le 150$ ms con `CircuitBreaker` y *fallback* heurístico local |
| **Disponibilidad de sala en vivo (Uptime)** | $\ge 99.5\%$ durante eventos programados |
| **Tolerancia a condiciones de carrera** | **0%** (Cero colisiones permitidas) |

---

### Artículo VI: Ciclo de Vida SDD (Spec-Driven Development)
1. **No Vibe-Coding:** Ningún módulo o cambio estructural se implementa sin su especificación en `specs/`.
2. **Ciclo en Tres Fases:**
   - **`spec.md` (Qué y Por qué):** Alcance, reglas de negocio y criterios de aceptación verificables.
   - **`plan.md` (Cómo técnico):** Contratos de interfaz, esquema/infraestructura y pasos de implementación.
   - **`tasks.md` (Plan de acción):** Lista atómica y trazable de tareas completadas.
3. **Verificación Automatizada:** Todo cambio debe validar `go vet ./...`, pruebas unitarias/integración y pruebas de carga/E2E antes de pasar a producción.
