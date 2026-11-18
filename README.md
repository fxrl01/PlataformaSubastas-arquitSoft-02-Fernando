# LiveBid — Plataforma de Subastas en Tiempo Real

> **Enfoque de Diseño:** Software Design Description (SDD) · Monolito Modular Hexagonal (Clean Architecture) · Modelo C4 · Go 1.23+ & PostgreSQL 16

---

## 📌 Descripción del Sistema

**LiveBid** es una plataforma transaccional de subastas en vivo de alto rendimiento y confianza cero (**Zero-Trust**), diseñada bajo una arquitectura de **Monolito Modular Hexagonal en Go 1.23+**. Permite la concurrencia masiva de postores durante el conteo final del martillo garantizando:

- **Gobernanza estricta por roles (`RBAC`) y KYC:** Separación formal de responsabilidades entre **Administrador (`ADMIN`)**, **Martillero Oficial (`MARTILLERO`)**, **Vendedor/Consignante (`VENDEDOR`)** y **Postor (`POSTOR`)**, con invalidación reactiva en caché (`CachedUserRepository`).
- **Custodia financiera preventiva (Escrow con Stripe & Web3):** Retención de fondos en tarjeta vía **Stripe `PaymentIntents` (`capture_method: manual`)** antes de habilitar pujas, captura automática al ganador (`Capture`), liberación inmediata a no ganadores (`Cancel`) y soporte híbrido de colateral on-chain (`Base L2`).
- **Resolución atómica y baja latencia (`< 25 ms` local):** Eliminación de condiciones de carrera mediante semáforos distribuidos en memoria (**Redis `SETNX`** de `1.5s` con liberación Lua), bloqueo pesimista ACID en **PostgreSQL 16 (`SELECT ... FOR UPDATE`)** y difusión multiplexada $O(1)$ por sala vía **WebSockets (`AuctionRoomHub`)**.
- **Inteligencia Artificial resiliente y Agentes Autónomos:** Evaluación crediticia dinámica con **Google Gemini** protegida por un **`CircuitBreaker`** (`150 ms` con *fallback* determinista local), enrutador multi-modelo en caliente (`MultiModelRouter`), detección asíncrona de anomalías y delegación de **Auto-Puja** con límite presupuestario (`max_cap`).
- **Trazabilidad legal y criptográfica inmutable:** Cadena de hashes `SHA-256` por puja, árbol de Merkle verificable (`MerkleRoot` + `MerkleProof`), firmas tipadas `EIP-712` y ficha pericial/encargado oficial en `JSONB`.

---

## 📂 Estructura del Repositorio (Docs-as-Code)

El repositorio separa formalmente la especificación arquitectónica (`docs/`) de la implementación en producción (`tecnologia/`):

```text
PlataformaSubastas-arquitSoft-02-Fernando/
├── docs/
│   ├── 01-analisis-de-sistema/             # Requisitos, actores, RNF, drivers, ADRs y escenarios SEI
│   │   ├── 01-actores.md
│   │   ├── 02-historias-de-usuario.md
│   │   ├── 03-requisitos-funcionales.md
│   │   ├── 04-atributos-de-calidad.md      # Atributos de calidad (AC01-AC08) + RNF (RNF-01 a RNF-08)
│   │   ├── 05-restricciones.md
│   │   ├── 06-drivers-arquitectonicos.md
│   │   ├── 07-decisiones-arquitectonicas.md
│   │   └── 08-escenarios-atributos-calidad.md
│   ├── 02-arquitectura-software/           # Estilo arquitectónico, 6 módulos y visor interactivo
│   │   ├── arquitectura-inicial.md
│   │   ├── componentes-arquitectonicos.md
│   │   ├── estilo-arquitectonico.md
│   │   ├── enfoque-aquitectonico.md
│   │   ├── livebid-arquitectura.architecture.json
│   │   └── livebid-arquitectura.html       # Diagrama interactivo explorable
│   ├── 03-diseño-de-software/              # SDD: diseño interno de módulos, patrones GoF/Cloud y SOLID
│   │   └── diseño-interno/
│   │       ├── diseño-interno-de-modulos.md
│   │       ├── patrones-de-diseño.md
│   │       └── principios-de-diseño.md
│   └── 04-modelo-c4/                       # Niveles 1 al 4 de Simon Brown
│       ├── Nivel1-DiagramadeContextodelSistema.md
│       ├── Nivel2-DiagramadeContenedores.md
│       ├── Nivel3-Diagrama-de-Componentes.md
│       └── Nivel4-DiagramadeCodigo.md
├── tecnologia/
│   ├── livebid-backend/                    # Backend Principal en Go 1.23+ (Clean/Hexagonal Architecture)
│   │   ├── cmd/server/main.go              # Punto de entrada, DI, rutas HTTP/WS y Scheduler
│   │   └── internal/
│   │       ├── core/domain/                # Entidades puras, máquina de estados y puertos (Ports)
│   │       ├── core/services/              # Casos de uso (Auth, Catálogo, Pujas, Garantías, IA, Web3)
│   │       └── adapters/                   # Handlers HTTP/WS, Postgres, Redis, Stripe, Gemini IA, Web3
│   └── boilerplate-clean-arquitecture/     # Cliente Web SPA (public/) + Referencia histórica TypeScript
│       ├── public/                         # Interfaz SPA reactiva por rol (index.html, app.js, styles.css)
│       └── src/app/                        # Prototipo inicial de reglas de dominio en TypeScript
├── docker/
│   └── init.sql                            # Esquema DDL relacional ACID en PostgreSQL 16
├── docker-compose.yml                      # Orquestación de livebid-api, livebid-postgres y livebid-redis
├── Dockerfile                              # Build multi-stage de Go (< 30 MB RAM)
├── .env.example                            # Plantilla de variables de entorno
└── README.md
```

---

## 🗺️ Trazabilidad Arquitectónica (SDD & Modelo C4)

| Nivel C4 | Documento | Pregunta que responde | Artefactos Clave |
| :--- | :--- | :--- | :--- |
| **Nivel 1** | [Contexto del Sistema](docs/04-modelo-c4/Nivel1-DiagramadeContextodelSistema.md) | ¿Quién usa el sistema y con qué servicios externos se conecta? | Postor, Vendedor, Martillero, Admin, **Stripe API**, **Google Gemini IA**, **Cloud Storage/KYC**, **Base L2 Web3** |
| **Nivel 2** | [Contenedores](docs/04-modelo-c4/Nivel2-DiagramadeContenedores.md) | ¿Qué aplicaciones y almacenes ejecutables existen? | Frontend Web SPA, **Monolito Modular Go (`livebid-backend`)**, **PostgreSQL 16** (Única BD ACID), **Redis 7** (Bus efímero) |
| **Nivel 3** | [Componentes](docs/04-modelo-c4/Nivel3-Diagrama-de-Componentes.md) | ¿Cómo se estructuran los 6 módulos internos del backend? | M1 Auth/RBAC, M2 Catálogo/Lote, M3 Garantías Stripe/Web3, M4 Motor Pujas/WS Hub, M5 Copiloto IA, M6 Auditoría Merkle |
| **Nivel 4** | [Código](docs/04-modelo-c4/Nivel4-DiagramadeCodigo.md) | ¿Cómo interactúan las interfaces y structs del caso de uso crítico? | Diagramas UML de dominio, secuencia atómica de puja y liquidación de garantías con Stripe |

---

## ⚙️ Estado de Implementación (6 Módulos Operativos)

El backend principal en Go ([`tecnologia/livebid-backend`](tecnologia/livebid-backend)) implementa de extremo a extremo los **6 módulos arquitectónicos**:

1. **M1 — Identidad, Roles (`RBAC`) y KYC:** Registro con rol solicitado, login con JWT + Refresh Tokens, `BootstrapAdmin` automático y panel de administración (`/api/v1/admin/users/*`).
2. **M2 — Catálogo, Encargado Oficial y Ficha del Lote:** Creación de subastas, asignación de Martillero (`PUT /encargado`), subida de imágenes referenciales y peritaje (`PUT /detalles-lote` en `JSONB`) y cierre automático de vencidas (`AuctionScheduler`).
3. **M3 — Custodia de Garantías (Stripe `PaymentIntents` Hold & Web3):** Pre-autorización bancaria (`POST /garantias/stripe-intent`), confirmación (`POST /garantias`), liquidación al cierre (`Capture` al ganador / `Cancel` a perdedores) y receptor firmado (`POST /api/v1/webhooks/stripe`).
4. **M4 — Motor de Pujas en Tiempo Real y Anti-Sniping:** Exclusión mutua en Redis (`SETNX`), persistencia transaccional con `Idempotency-Key`, extensión automática de `+30s` en los últimos `15s` y difusión WebSocket por sala (`/ws/subastas/{id}`).
5. **M5 — Riesgo IA, Enrutador Multi-Modelo y Auto-Puja:** Evaluación con Google Gemini bajo `CircuitBreaker`, cambio de modelo en caliente (`PATCH /api/v1/admin/ia/modelo`), detector asíncrono de colusión y delegación de agentes (`POST /agentes-ia/delegar`).
6. **M6 — Auditoría Criptográfica y Web3:** Verificación de firmas `EIP-712` y generación de certificados de integridad con `MerkleRoot` y `MerkleProof` (`GET /auditoria-web3`).

---

## 🧪 Ejecución de Pruebas y Entorno Local

### 1. Ejecutar Pruebas Unitarias y de Integración del Backend Go

```bash
cd tecnologia/livebid-backend
go test -v ./...
```

### 2. Levantar el Ecosistema Completo con Docker Compose

```bash
cp .env.example .env
docker compose up --build -d
```

### 3. Pruebas de Referencia Histórica (Prototipo TypeScript)

```bash
cd tecnologia/boilerplate-clean-arquitecture
node --experimental-strip-types src/app/pruebas/subasta.pruebas.ts
```

---

## 🧭 Diagrama Interactivo de Arquitectura

Para explorar la vista interactiva con zoom, trazabilidad de flujos y navegación por capas y componentes, abrir el archivo:
- [docs/02-arquitectura-software/livebid-arquitectura.html](docs/02-arquitectura-software/livebid-arquitectura.html)
