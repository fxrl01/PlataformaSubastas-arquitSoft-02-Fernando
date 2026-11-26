# Evidencias y Resultados de Validacion de Atributos de Calidad — LiveBid

> **Proyecto:** LiveBid — Plataforma de Subastas en Tiempo Real  
> **Enfoque:** Spec-Driven Development (`SDD`) · Monolito Modular Hexagonal (Go 1.23+ & PostgreSQL 16 & Redis 7)  
> **Especificacion asociada:** [`specs/015-quality-assurance-openapi`](../../specs/015-quality-assurance-openapi/spec.md) · [`Constitucion SDD`](../../.specify/constitution.md)

---

## 1. Resumen Ejecutivo de Validacion Empirica

Todas las pruebas y mediciones reportadas en este documento fueron ejecutadas contra el codigo fuente real y el cluster contenerizado de **6 servicios** (`livebid-nginx-gateway`, `livebid-api-1`, `livebid-api-2`, `livebid-pgbouncer`, `livebid-postgres` y `livebid-redis`):

| Suite / Herramienta | Alcance Evaluado | Resultado Medido | Estado |
| :--- | :--- | :--- | :---: |
| **Jest + `ts-jest` (`npm run test:coverage` + CI)** | Reglas de dominio, casos de uso, infraestructura (`CircuitBreaker`, `Escrow`), controlador en TypeScript y job automatizado en `ci.yml` | **100 % Statements** (`206/206`), **100 % Branches** (`98/98`), **100 % Functions** (`38/38`), **100 % Lines** (`200/200`) — `3/3` suites, `7/7` tests | **APROBADO** |
| **Go `-race` + Postgres/Redis reales (`go test -count=1 -race ./...`)** | Dominio puro, servicios, locks Redis, `singleflight`, transacciones ACID, Stripe, IA, WebSockets, Web3 y contrato OpenAPI | **`83.2 %`** en `./internal/...` (`78.0 %` global con `cmd/server`); `domain` **100 %**, `web3` **100 %**, `security` **96.3 %**, `ai` **93.5 %**; **0 condiciones de carrera** | **APROBADO** |
| **Playwright E2E (`npx playwright test`)** | Cluster real con 2 navegadores simultaneos (Admin + Postor), Nginx LB, cache estatica, Swagger UI, KYC, Stripe Hold, WebSocket y Merkle Proof | **`2 / 2` escenarios E2E aprobados (`14.9 s`)**; balanceo verificado en `>= 2` replicas Go y sincronia WS en vivo | **APROBADO** |
| **Grafana k6 Escala `2,000 VUs` (`PROFILE=scale2000`)** | Simulacion progresiva hasta **`2,000` usuarios virtuales concurrentes** (`1,650 WS + 325 Catalogo + 25 Pujadores`) | **`2,000` VUs max**, **`46,861` reqs HTTP**, **`46,010 / 46,010` checks (`100 %`)**, **`154,382` eventos WS** (`1,975.5 msg/s`), **P95 puja = `108.75 ms`** (`P50 = 15.76 ms`), **P95 catalogo = `102.18 ms`**, **`0` errores HTTP 5xx** | **APROBADO** |
| **Analisis BD & Cache (`EXPLAIN ANALYZE` + `Redis INFO`)** | Indices parciales PostgreSQL 16, `PgBouncer`, `Cache-Aside` y coalescencia `singleflight` | **Hit Ratio Redis = `97.02 %`**, `-96.8 %` en `seq_scan` sobre `subastas` (`32,171 -> 1,028`), `99.89 %` `Index Scan` en `pujas` y bloqueo `FOR UPDATE` en `0.040 ms` | **APROBADO** |
| **Auditoria Estatica SDD & PCI-DSS** | Aislamiento de `internal/core/domain` y esquema relacional `docker/init.sql` | Dominio importa **unicamente Go stdlib** (`0` librerias externas); **`0` columnas PAN/CVV** en PostgreSQL | **APROBADO** |

---

## 2. Matriz de Validacion de Atributos de Calidad (`AC01` – `AC08`)

Trazabilidad directa contra [`04-atributos-de-calidad.md`](../01-analisis-de-sistema/04-atributos-de-calidad.md):

| ID | Atributo | Criterio Objetivo (SLA) | Resultado Real Obtenido | Evidencia Verificable |
| :--- | :--- | :--- | :--- | :--- |
| **AC01** | **Rendimiento** | Confirmacion de puja `P95 <= 200 ms`; lectura de catalogo `P95 < 150 ms` | Con **`2,000 VUs` concurrentes**: **Puja `P95 = 108.75 ms`** (`P50 = 15.76 ms`); **Catalogo `P95 = 102.18 ms`** (`P50 = 17.54 ms`) | [`k6-summary.json`](../../tests/load/results/k6-summary.json), [`k6-report.html`](../../tests/load/results/k6-report.html) |
| **AC02** | **Escalabilidad** | Soportar `2,000+` espectadores/pujadores concurrentes con hub `O(1)` en Redis + `PgBouncer` + `singleflight` | **`2,000` VUs simultaneos** (`1,650 WS + 325 lectores + 25 pujadores`), `2,574` conexiones WS (`0` rechazadas), `154,382` eventos WS difundidos (`1,975.5 msg/s`) en `1` canal Redis y `97.02 %` Hit Ratio | [`analisis-bd-y-estrategia-cache.md`](../03-dise%C3%B1o-de-software/analisis-bd-y-estrategia-cache.md), [`nginx.conf`](../../docker/nginx/nginx.conf) |
| **AC03** | **Disponibilidad y Resiliencia** | `0` caidas (`5xx = 0`) y degradacion elegante con `CircuitBreaker` ante fallo de Gemini IA | **`livebid_http_5xx_errors = 0`** en `46,861` peticiones a `2,000 VUs`; `CircuitBreaker` conmuta `CLOSED -> OPEN -> HALF_OPEN -> CLOSED` activando fallback local sin abortar pujas | [`circuit_breaker_test.go`](../../tecnologia/livebid-backend/internal/adapters/ai/circuit_breaker_test.go), [`infraestructura-presentacion.spec.ts`](../../tecnologia/boilerplate-clean-arquitecture/src/app/__tests__/infraestructura-presentacion.spec.ts) |
| **AC04** | **Seguridad (`Zero-Trust`)** | `JWT (HS256)` + `RBAC` en tiempo real + `Stripe PaymentIntent` (`capture_method: manual`) sin tocar `PAN/CVV` | Rechazo `401/403` verificado para usuarios no autenticados, pendientes o sin rol requerido; `0` columnas de tarjeta en BD; webhooks verificados por HMAC-SHA256 | [`auth_middleware_test.go`](../../tecnologia/livebid-backend/internal/adapters/handlers/auth_middleware_test.go), [`stripe_adapter_test.go`](../../tecnologia/livebid-backend/internal/adapters/payments/stripe_adapter_test.go), [`init.sql`](../../docker/init.sql) |
| **AC05** | **Integridad y Orden Verificable** | `0 %` condiciones de carrera; serializacion `SETNX` + `SELECT ... FOR UPDATE`; cadena `SHA-256` y `MerkleRoot` | `1,108 / 1,108` pujas concurrentes resueltas atomicamente (`201` o `409`, `0` corrupcion); `go test -race` sin data races; `cadena_integra = true` y `prueba_merkle.verificada = true` en E2E | [`integration_postgres_redis_test.go`](../../tecnologia/livebid-backend/internal/adapters/persistence/integration_postgres_redis_test.go), [`web3_test.go`](../../tecnologia/livebid-backend/internal/adapters/web3/web3_test.go) |
| **AC06** | **Mantenibilidad y Evolucion** | Dominio hexagonal aislado (`0` dependencias externas), contratos OpenAPI verificados y cobertura alta | `internal/core/domain` importa solo `[context crypto/sha256 encoding/hex errors fmt strings time]`; Jest `100 %`; paridad automatica `main.go <-> openapi.json` | [`ports.go`](../../tecnologia/livebid-backend/internal/core/domain/ports.go), [`swagger_handler_test.go`](../../tecnologia/livebid-backend/internal/adapters/handlers/swagger_handler_test.go) |
| **AC07** | **Usabilidad en Tiempo Real** | Sincronizacion instantanea por WebSocket sin recargar pagina y extension *Anti-Sniping* (`+30 s` en ultimos `15 s`) | Navegador Admin recibe en vivo la oferta emitida desde el navegador Postor (`#totalBidsCount -> 1 oferta`) y el dominio extiende `+30 s` en ventana `<= 15 s` | [`livebid.e2e.spec.js`](../../tests/e2e/livebid.e2e.spec.js), [`domain_test.go`](../../tecnologia/livebid-backend/internal/core/domain/domain_test.go) |
| **AC08** | **Portabilidad** | Despliegue reproducible multi-plataforma mediante Docker multi-stage y pipeline CI/CD | Build multi-stage Go sobre `alpine:3.20` y stack `docker compose up --build -d --wait` operativo con `6/6` contenedores `Healthy` | [`Dockerfile`](../../Dockerfile), [`ci.yml`](../../.github/workflows/ci.yml), [`cd.yml`](../../.github/workflows/cd.yml) |

---

## 3. Verificacion de Escenarios SEI (`EQ-01` – `EQ-05`) y Requisitos No Funcionales (`RNF-01` – `RNF-08`)

### 3.1 Escenarios de 6 Partes del SEI ([`08-escenarios-atributos-calidad.md`](../01-analisis-de-sistema/08-escenarios-atributos-calidad.md))

1. **`EQ-01` (Latencia y difusion multiplexada `O(1)`):** Comprobado con `AuctionRoomHub` (una sola suscripcion Redis Pub/Sub compartida por multiples clientes WebSocket de la misma sala en [`websocket_handler_test.go`](../../tecnologia/livebid-backend/internal/adapters/realtime/websocket_handler_test.go)) y benchmark `k6` (`P95 = 83.87 ms` en pujas, `3,742` mensajes WS recibidos sin perdidas).
2. **`EQ-02` (Resolucion atomica, `SHA-256` y `MerkleTree`):** Comprobado contra PostgreSQL 16 y Redis 7 reales en [`integration_postgres_redis_test.go`](../../tecnologia/livebid-backend/internal/adapters/persistence/integration_postgres_redis_test.go) y en el paso 7 de [`livebid.e2e.spec.js`](../../tests/e2e/livebid.e2e.spec.js) (`GET /api/v1/subastas/{id}/auditoria-web3?indice=0` retorna `cadena_integra: true` y `prueba_merkle.verificada: true`).
3. **`EQ-03` (Tolerancia a fallos IA con `CircuitBreaker` y `MultiModelRouter`):** Comprobado en [`ai_coverage_test.go`](../../tecnologia/livebid-backend/internal/adapters/ai/ai_coverage_test.go), [`circuit_breaker_test.go`](../../tecnologia/livebid-backend/internal/adapters/ai/circuit_breaker_test.go) y [`infraestructura-presentacion.spec.ts`](../../tecnologia/boilerplate-clean-arquitecture/src/app/__tests__/infraestructura-presentacion.spec.ts): ante fallos consecutivos de Gemini, el circuito pasa a `OPEN` devolviendo evaluacion heuristica local (`modo_fallback: true`) y permite conmutacion en caliente (`PATCH /api/v1/admin/ia/modelo`) con `0 ms` de caida.
4. **`EQ-04` (Seguridad Zero-Trust, invalidacion RBAC y custodia Escrow):** Comprobado en [`cached_user_repository_test.go`](../../tecnologia/livebid-backend/internal/adapters/persistence/cached_user_repository_test.go) (invalidacion multi-instancia via Redis Pub/Sub al cambiar rol/estado), rechazo de puja sin garantia activa y deteccion asincrona de anomalias (*Ping-Pong*, velocidad sobrehumana y salto irracional de precio).
5. **`EQ-05` (Agentes Autonomos acotados por `max_cap`):** Comprobado en [`services_coverage_test.go`](../../tecnologia/livebid-backend/internal/core/services/services_coverage_test.go): `AgentDelegationService` bloquea con `ErrAgentMandateExceeded` / `403 Forbidden` cualquier intento de superar `monto_maximo_autorizado` o usar un mandato vencido.

### 3.2 Cumplimiento de Requisitos No Funcionales (`RNF-01` – `RNF-08`)

| RNF | Verificacion Tecnica | Resultado |
| :--- | :--- | :---: |
| **RNF-01 (`PCI-DSS`)** | Inspeccion de [`docker/init.sql`](../../docker/init.sql): la tabla `garantias` solo almacena `autorizacion_bancaria_id` (`pi_...`), nunca `PAN`/`CVV`. | Cumplido |
| **RNF-02 (`RBAC + Bcrypt`)** | `BcryptHasher` (costo 12 en produccion), `JWTIssuer` (`HS256`) y validacion por peticion en `RequireAuth` + `RequireRoles` (`96.3 %` cobertura en `security`). | Cumplido |
| **RNF-03 (`Unica BD ACID`)** | Unica base persistente `PostgreSQL 16` con columnas `JSONB` (`encargado_json`, `detalles_lote_json`) y transacciones `SELECT ... FOR UPDATE`. | Cumplido |
| **RNF-04 (`Redis Efimero`)** | Redis limitado a `SETNX` con liberacion Lua, Pub/Sub y `Cache-Aside` con degradacion transparente a PostgreSQL ([`cached_auction_repository_test.go`](../../tecnologia/livebid-backend/internal/adapters/persistence/cached_auction_repository_test.go)). | Cumplido |
| **RNF-05 (`Latencia Sala`)** | Mediana de puja `16.92 ms` y `P95 = 83.87 ms` bajo carga concurrente real en `k6`. | Cumplido |
| **RNF-06 (`Circuit Breaker IA`)** | `0 %` de bloqueos por proveedor externo de IA; fallback determinista verificado en Go y TypeScript. | Cumplido |
| **RNF-07 (`Idempotencia`)** | Indices unicos `uq_pujas_idempotency` y `uq_garantia_retenida_subasta_postor` + pruebas de conflicto `409` al reutilizar `Idempotency-Key` con distinto monto. | Cumplido |
| **RNF-08 (`Monolito Hexagonal`)** | Binario unico en Go, `go vet ./...` sin advertencias y suite completa en segundos. | Cumplido |

---

## 4. Cumplimiento de la Constitucion SDD (`.specify/constitution.md`)

| Articulo Constitucional | Evidencia de Cumplimiento |
| :--- | :--- |
| **Articulo I — Monolito Modular Hexagonal** | `go list -f '{{.Imports}}' ./internal/core/domain` devuelve unicamente paquetes estandar de Go. |
| **Articulo II — Zero Trust, RBAC y Escrow** | Autenticacion JWT, aprobacion KYC administrativa previa, retencion Stripe Hold y liquidacion automatica al cierre. |
| **Articulo III — Concurrencia Atomica y Fuente Legal** | Reloj oficial del servidor, mutex Redis + `FOR UPDATE`, anti-sniping (`15 s -> +30 s`), cadena `SHA-256` y `MerkleTree`. |
| **Articulo IV — Persistencia Unica ACID y Cache** | PostgreSQL 16 + PgBouncer + Redis Cache-Aside + 2 replicas Go stateless detras de Nginx. |
| **Articulo V — Presupuestos de Calidad (SLAs)** | `P95 = 83.87 ms <= 200 ms`, `0 %` race conditions (`-race`), `0` errores `5xx`. |
| **Articulo VI — Ciclo de Vida SDD (`specs/000` a `015`)** | Cada modulo e incremento de calidad cuenta con su tripleta `spec.md`, `plan.md` y `tasks.md` bajo [`specs/`](../../specs/015-quality-assurance-openapi/spec.md). |

---

## 5. Comandos para Reproducir Todas las Evidencias

```bash
# 1. Cobertura Jest al 100% (TypeScript Clean Architecture)
cd tecnologia/boilerplate-clean-arquitecture && npm run test:coverage

# 2. Analisis estatico, detector de carreras (-race) e integracion real Postgres/Redis (Go)
cd tecnologia/livebid-backend && go vet ./... && go test -count=1 -race -cover ./...

# 3. Suite End-to-End en vivo (2 navegadores Chromium + WebSocket + Stripe + Swagger + Merkle)
cd tests/e2e && npx playwright test --reporter=list

# 4. Benchmark de carga concurrente y WebSockets con Grafana k6
docker run --rm --network host -u $(id -u):$(id -g) \
  -v "$PWD/tests/load:/scripts" grafana/k6 run /scripts/livebid-load.js
```
