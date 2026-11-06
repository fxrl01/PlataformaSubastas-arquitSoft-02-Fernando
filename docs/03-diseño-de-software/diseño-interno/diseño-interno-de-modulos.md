# Diseño interno de módulos — SDD

> **Proyecto:** LiveBid — Plataforma de Subastas en Tiempo Real  
> **Backend Oficial:** Go (Golang 1.23+) · **Módulo:** `github.com/fxrl01/livebid-backend`  
> **Enfoque Arquitectónico:** Arquitectura Hexagonal (Puertos y Adaptadores / Clean Architecture)  
> **Nivel C4:** 4 – Código · **Estándar:** Software Design Description (SDD) · **Versión:** 3.0

---

## 1. Propósito

Este documento de **Descripción de Diseño de Software (SDD)** especifica la estructura real de paquetes, contratos de interfaces y flujos internos del backend en **Go 1.23+** ubicado en [`tecnologia/livebid-backend`](file:///home/fernando/Softwar_Architect/PlataformaSubastas-arquitSoft-02-Fernando/tecnologia/livebid-backend).

---

## 2. Estructura Real del Proyecto en Go (`tecnologia/livebid-backend`)

```text
tecnologia/livebid-backend/
├── cmd/
│   └── server/
│       └── main.go                                 // Inyección de dependencias, servidor HTTP/WS, CORS y Expiry Worker
├── internal/
│   ├── core/                                       // NÚCLEO DE NEGOCIO PURO (0 dependencias de infraestructura)
│   │   ├── domain/
│   │   │   ├── auction.go                          // Aggregate Root Subasta (EncargadoSubasta, DetallesLote, Anti-Sniping)
│   │   │   ├── auction_state.go                    // Máquina de estados (PENDIENTE, ACTIVA, EN_CONTEO, CERRADA)
│   │   │   ├── bid.go                              // Value Object Puja (inmutable con timestamp y monto)
│   │   │   ├── increment_rule.go                   // Cálculo de tramos dinámicos de incremento mínimo
│   │   │   ├── user.go                             // Entidad User, roles RBAC (POSTOR, VENDEDOR, MARTILLERO, ADMIN) y KYC
│   │   │   ├── web3.go                             // Árbol de Merkle puro (ComputeMerkleRoot, GenerateMerkleProof, EIP-712)
│   │   │   ├── ai_agent.go                         // Mandatos de Agentes IA (AgentGrant, estrategias, BidAnomalyAlert)
│   │   │   ├── ports.go                            // Interfaces de salida (SubastaRepository, BidLockPort, AIRiskPort, etc.)
│   │   │   └── errors.go                           // Errores tipados del dominio
│   │   └── services/
│   │       ├── auction_service.go                  // Caso de uso crítico: ProcesarPuja y CerrarVencidas
│   │       ├── auction_catalog_service.go          // Gestión de subastas, EncargadoSubasta, DetallesLote y estados
│   │       ├── auth_service.go                     // Registro, Login, Refresh Token, Roles RBAC y activación KYC
│   │       ├── escrow_service.go                   // Congelación y consulta de garantías previas
│   │       ├── web3_audit_service.go               // Generación de certificados Merkle y recibos on-chain
│   │       └── agent_delegation_service.go         // Delegación y ejecución de auto-puja por Agentes IA con max_cap
│   └── adapters/                                   // CAPA DE ADAPTADORES (Presentación e Infraestructura)
│       ├── handlers/
│       │   ├── auth_handler.go                     // Endpoints /api/v1/auth/*, /admin/users/* y middlewares RequireAuth/RequireRoles
│       │   ├── auction_handler.go                  // Endpoints /api/v1/subastas/*, encargado, detalles-lote y auditoría Web3
│       │   ├── bid_handler.go                      // Endpoint POST /api/v1/subastas/{id}/pujas (Idempotency-Key + EIP-712)
│       │   ├── escrow_handler.go                   // Endpoints POST/GET /api/v1/subastas/{id}/garantias
│       │   └── agent_ai_handler.go                 // Endpoints de Agentes IA, inteligencia antifraude y Hot-Swap de modelo
│       ├── realtime/
│       │   ├── room_hub.go                         // AuctionRoomHub: 1 suscripción Redis Pub/Sub compartida por sala O(1)
│       │   ├── websocket_handler.go                // Handshake WS autenticado (Bearer o ?token=) y bombas read/write
│       │   ├── redis_bid_notifier.go               // Publicación y suscripción en canales Redis subasta:eventos:{id}
│       │   └── redis_lock_adapter.go               // Mutex distribuido atómico SETNX + liberación segura con script Lua
│       ├── persistence/
│       │   ├── postgres_auction_repository.go      // CRUD y ListarVencidas indexado en PostgreSQL 16
│       │   ├── postgres_bid_repository.go          // RegistrarPujaAtomica con SELECT FOR UPDATE y cadena SHA-256
│       │   ├── postgres_escrow_repository.go       // Persistencia de garantías retenidas
│       │   ├── postgres_user_repository.go         // Gestión de usuarios, roles, estados y refresh tokens
│       │   └── cached_user_repository.go           // Decorador en memoria con TTL e invalidación instantánea
│       ├── ai/
│       │   ├── multi_model_router.go               // Enrutador dinámico de modelos IA (Hot-Swap en tiempo de ejecución)
│       │   ├── circuit_breaker.go                  // Cortacircuitos atómico (timeout 150ms + fallback heurístico)
│       │   ├── gemini_risk_adapter.go              // Cliente HTTP estructurado hacia Google Gemini AI
│       │   └── anomaly_detector.go                 // Worker asíncrono antifraude (Bot Sniping, Wash Trading, Shill Bidding)
│       ├── web3/
│       │   ├── eip712_verifier.go                  // Verificación criptográfica de firmas EIP-712 y prevención de replay
│       │   └── evm_escrow_adapter.go               // Custodia en Smart Contract y emisión de recibos on-chain
│       └── security/
│           ├── jwt.go                              // Emisión y validación HMAC-SHA256 de Access Tokens
│           └── password.go                         // Hashing y verificación con bcrypt
├── go.mod
└── go.sum
```

---

## 3. Flujo de Ejecución del Caso de Uso Crítico: [`AuctionService.ProcesarPuja`](file:///home/fernando/Softwar_Architect/PlataformaSubastas-arquitSoft-02-Fernando/tecnologia/livebid-backend/internal/core/services/auction_service.go)

1. **Autenticación y RBAC (`RequireAuth` + `RequireRoles`):** Se verifica el JWT y se consulta [`CachedUserRepository.FindByID`](file:///home/fernando/Softwar_Architect/PlataformaSubastas-arquitSoft-02-Fernando/tecnologia/livebid-backend/internal/adapters/persistence/cached_user_repository.go) en $< 0.05\text{ ms}$ comprobando `Status == ACTIVO` y `Role == POSTOR`.
2. **Adquisición del Lock Distribuido (`SETNX`):** [`RedisLockAdapter.AdquirirLock`](file:///home/fernando/Softwar_Architect/PlataformaSubastas-arquitSoft-02-Fernando/tecnologia/livebid-backend/internal/adapters/realtime/redis_lock_adapter.go) bloquea `lock:subasta:{id}` por $1500\text{ ms}$ y retorna un `unlock()` seguro basado en script Lua.
3. **Verificación Criptográfica Web3 (Opcional):** Si la oferta incluye un `EIP712BidEnvelope`, [`EIP712Verifier.VerifyBidSignature`](file:///home/fernando/Softwar_Architect/PlataformaSubastas-arquitSoft-02-Fernando/tecnologia/livebid-backend/internal/adapters/web3/eip712_verifier.go) valida el digest tipado, expiración (`deadline`) y protección anti-replay.
4. **Verificación de Custodia (Escrow Fiduciario y On-Chain):** Se comprueba que el postor tenga garantía activa en [`PostgresEscrowRepository`](file:///home/fernando/Softwar_Architect/PlataformaSubastas-arquitSoft-02-Fernando/tecnologia/livebid-backend/internal/adapters/persistence/postgres_escrow_repository.go) y colateral válido en [`EVMEscrowAdapter`](file:///home/fernando/Softwar_Architect/PlataformaSubastas-arquitSoft-02-Fernando/tecnologia/livebid-backend/internal/adapters/web3/evm_escrow_adapter.go).
5. **Evaluación de Riesgo IA con `CircuitBreaker`:** Se consulta [`MultiModelRouter.EvaluarRiesgo`](file:///home/fernando/Softwar_Architect/PlataformaSubastas-arquitSoft-02-Fernando/tecnologia/livebid-backend/internal/adapters/ai/multi_model_router.go) con presupuesto máximo de $150\text{ ms}$; si supera el umbral de riesgo ($> 75$), la puja es rechazada con `ErrRiesgoAltoIA`.
6. **Validación en el Aggregate Root (`Subasta.ValidarYRegistrarPuja`):**
   - Verifica estado (`ACTIVA` o `EN_CONTEO`), prohibición de auto-puja (`ErrPostorIgual`) e incremento mínimo por tramo.
   - Si restan $\le 15\text{ s}$ para `FechaCierre`, activa *Anti-Sniping* sumando $+30\text{ s}$.
7. **Persistencia Transaccional Atómica (`RegistrarPujaAtomica`):** Ejecuta `SELECT ... FOR UPDATE` en PostgreSQL, verifica idempotencia (`Idempotency-Key`), calcula el `hash_auditoria` SHA-256 encadenado al hash anterior e inserta en `pujas` y `logs_auditoria`.
8. **Broadcast Multiplexado y Telemetría Asíncrona:** Publica el evento en Redis Pub/Sub (difundido en vivo por `AuctionRoomHub`) y envía la telemetría de forma no bloqueante a `AsyncAnomalyDetector`.
