# Nivel 4 · Diagrama de código — LiveBid

> **Proyecto:** LiveBid — Plataforma de Subastas en Tiempo Real  
> **Flujos Detallados:** `ProcesarPuja` (Hot Path + Web3 + IA) y Gestión de `Subasta` (`EncargadoSubasta` + `DetallesLote` + RBAC)  
> **Modelo:** C4 · Nivel 4 de 4 · **Lenguaje:** Go 1.23+ · **Notación:** UML (Clases y Secuencia) · **Versión:** 3.0

---

## 1. Objetivo

Documentar a nivel de **código fuente en Go** los `structs`, interfaces (`ports.go`), firmas de métodos y diagramas de secuencia que implementan la lógica crítica del sistema.

---

## 2. Diagrama de Clases / Estructuras UML (Go 1.23+ Hexagonal)

```mermaid
classDiagram
    direction TB

    namespace Dominio_Identidad_RBAC {
        class User {
            +ID string
            +Name string
            +Email string
            +Role UserRole
            +Status UserStatus
            +KYCApproved bool
            +Phone string
            +DocumentID string
            +WalletAddress string
        }
        class UserRole {
            <<enumeration>>
            POSTOR
            VENDEDOR
            MARTILLERO
            ADMIN
        }
        class UserStatus {
            <<enumeration>>
            PENDIENTE
            ACTIVO
            SUSPENDIDO
            BLOQUEADO
        }
    }

    namespace Dominio_Subastas_Y_Lotes {
        class Subasta {
            +ID string
            +VendedorID string
            +MartilleroID string
            +Titulo string
            +Descripcion string
            +PrecioBase float64
            +PrecioActual float64
            +PostorLiderID string
            +Estado EstadoSubasta
            +FechaInicio time.Time
            +FechaCierre time.Time
            +Encargado EncargadoSubasta
            +DetallesLote DetallesLote
            +Pujas []Puja
            +Activar() error
            +IniciarConteo() error
            +Cerrar() error
            +ValidarYRegistrarPuja(id, postorID string, monto float64, ahora time.Time) (*Puja, bool, error)
        }
        class EncargadoSubasta {
            +MartilleroID string
            +Nombre string
            +Cargo string
            +Licencia string
            +Contacto string
            +Experiencia string
        }
        class DetallesLote {
            +Categoria string
            +Ubicacion string
            +PrecioReserva float64
            +ImagenesURLs []string
            +CertificadoPeritajeURL string
            +InspeccionadoPor string
            +Especificaciones map~string,string~
        }
    }

    namespace Dominio_Web3_Y_Agentes_IA {
        class EIP712BidEnvelope {
            +WalletAddress string
            +SignatureHex string
            +Nonce uint64
            +ChainID int64
            +Deadline int64
        }
        class MerkleProof {
            +LeafHash string
            +LeafIndex int
            +Siblings []string
            +Directions []string
            +MerkleRoot string
        }
        class AgentGrant {
            +ID string
            +SubastaID string
            +PostorID string
            +MaxCap float64
            +Strategy AgentStrategy
            +Active bool
        }
    }

    namespace Servicios_Aplicacion {
        class AuctionService {
            -repo SubastaRepository
            -bidRepo BidPersistencePort
            -lockPort BidLockPort
            -escrowPort EscrowPort
            -notifierPort BidNotifierPort
            -riskPort AIRiskPort
            -sigVerifier BidSignatureVerifierPort
            -web3Escrow SmartContractEscrowPort
            -telemetry FraudTelemetryPort
            +ProcesarPuja(ctx context.Context, cmd ProcesarPujaCmd) (*ProcesarPujaResult, error)
            +CerrarVencidas(ctx context.Context, now time.Time) (int, error)
        }
        class AuctionCatalogService {
            -repo SubastaRepository
            -notifier BidNotifierPort
            +Crear(ctx, cmd) (*Subasta, error)
            +ActualizarEncargado(ctx, id string, enc EncargadoSubasta) (*Subasta, error)
            +ActualizarDetallesLote(ctx, id string, det DetallesLote) (*Subasta, error)
            +Activar(ctx, id string) error
            +IniciarConteo(ctx, id string) error
            +Cerrar(ctx, id string) error
        }
    }

    Subasta *-- EncargadoSubasta
    Subasta *-- DetallesLote
    AuctionService --> Subasta : valida y actualiza
    AuctionService --> EIP712BidEnvelope : verifica
    AuctionCatalogService --> Subasta : administra ciclo y metadatos
    User --> UserRole
    User --> UserStatus
```

---

## 3. Diagrama de Secuencia: Flujo Completo de Puja Atómica + Web3 + IA Exponencial

```mermaid
sequenceDiagram
    autonumber
    actor Postor as 👤 Postor VIP / Agente IA
    participant Handler as 🌐 BidHandler + RequireAuth
    participant Cache as ⚡ CachedUserRepository
    participant Svc as ⚙️ AuctionService
    participant Redis as 🧠 Redis (SETNX + Pub/Sub)
    participant Web3 as ⛓️ EIP712Verifier + EVMEscrow
    participant AI as 🔀 MultiModelRouter + CircuitBreaker
    participant Dom as 💎 Subasta (Aggregate Root)
    participant PG as 🗄️ PostgresBidRepo (FOR UPDATE)
    participant Hub as 📡 AuctionRoomHub (WS)
    participant Radar as 🔎 AsyncAnomalyDetector

    Postor->>Handler: POST /api/v1/subastas/{id}/pujas (Idempotency-Key + EIP-712)
    Handler->>Cache: FindByID(userID) [< 0.05ms en RAM]
    Cache-->>Handler: User{Role: POSTOR, Status: ACTIVO}
    Handler->>Svc: ProcesarPuja(ctx, cmd)

    critical Exclusión Mutua Distribuida
        Svc->>Redis: AdquirirLock("lock:subasta:" + id, 1500ms)
        Redis-->>Svc: lockOK, luaUnlockFunc
    end

    opt Si incluye sobre Web3 (EIP-712)
        Svc->>Web3: VerifyBidSignature(envelope, subastaID, postorID, monto)
        Web3-->>Svc: Firma Criptográfica Válida
    end

    Svc->>PG: TieneGarantiaActiva(subastaID, postorID)
    Svc->>Web3: VerifyLockedCollateral(subastaID, postorID, monto)
    Svc->>AI: EvaluarRiesgo(ctx, postorID, subastaID, monto) [Timeout 150ms]
    AI-->>Svc: Score <= 75 (Gemini o Fallback Heurístico)

    Svc->>Dom: ValidarYRegistrarPuja(pujaID, postorID, monto, ahora)
    Dom-->>Svc: nuevaPuja, tiempoExtendido (Anti-Sniping +30s si <=15s)

    critical Transacción ACID + Cadena Forense SHA-256
        Svc->>PG: RegistrarPujaAtomica(subasta, puja, idempotencyKey)
        Note over PG: SELECT FOR UPDATE -> Valida postor_lider_id <> postor_id<br/>Calcula hash_actual = SHA256(prev_hash + payload)
        PG-->>Svc: Commit OK
    end

    Svc->>Redis: luaUnlockFunc()

    par Difusión Multiplexada O(1) y Telemetría No Bloqueante
        Svc->>Redis: PUBLISH subasta:eventos:{id}
        Redis-->>Hub: 1 Mensaje por Sala Activa
        Hub-->>Postor: Broadcast WSS a todos los clientes (<25ms)
    and Análisis Antifraude en Segundo Plano
        Svc-)Radar: RecordBid(BidTelemetryEvent)
        Note over Radar: Evalúa Bot Sniping, Wash Trading y Shill Bidding
    end

    Svc-->>Handler: ProcesarPujaResult (200 OK)
```
