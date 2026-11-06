# Drivers arquitectónicos — LiveBid

> **Proyecto:** LiveBid — Plataforma de Subastas en Tiempo Real  
> **Capa:** Análisis de Sistema · **Versión:** 3.0 (Cloud-Native Go + Web3 + IA Exponencial)

---

## 1. Catálogo de Drivers Arquitectónicos

| ID | Driver Arquitectónico | Origen | ¿Por qué condiciona el diseño del sistema? |
|---|---|---|---|
| **DA01** | El sistema debe soportar $1,000+$ espectadores concurrentes por sala sin multiplicar las suscripciones al broker. | AC02 — Escalabilidad | Obliga a implementar un **`AuctionRoomHub`** en Go que multiplexe **1 única suscripción Redis Pub/Sub por sala activa** hacia $N$ clientes WebSocket locales ($O(1)$ conexiones a Redis por sala). |
| **DA02** | El orden y la integridad de las pujas deben ser matemáticamente verificables y no repudiables. | AC05 — Integridad / RF06 / RF14 | Obliga a combinar bloqueo distribuido (`SETNX`), bloqueo pesimista (`SELECT ... FOR UPDATE`), **cadena de hashes SHA-256** (`hash_auditoria`), firmas tipadas **EIP-712** y **Árbol de Merkle (`MerkleRoot`)**. |
| **DA03** | La confirmación de la puja y la difusión en vivo deben ocurrir en $< 50\text{ ms}$ (P95 $< 200\text{ ms}$). | AC01 — Rendimiento / RF02 | Obliga a separar el *Hot Path* en memoria (Go Goroutines + Redis) del *Cold Path*, y a cachear la identidad/rol del usuario en memoria (`CachedUserRepository`). |
| **DA04** | La custodia de garantías debe soportar tanto pasarelas fiduciarias como Smart Contracts Web3 sin acoplar el dominio. | RC08 / RF05 | Obliga a definir puertos puros en el dominio (`EscrowPort` y `SmartContractEscrowPort`) con adaptadores intercambiables (`PostgresEscrowRepository`, `EVMEscrowAdapter`). |
| **DA05** | La transmisión de video en vivo no debe saturar el ancho de banda del motor de pujas. | RC09 | Obliga a separar el canal de video (CDN externo WebRTC/LL-HLS) del canal de datos/precios (WebSockets nativos en Go). |
| **DA06** | Cada puja debe pasar por evaluación de riesgo IA sin que una caída o lentitud del LLM detenga la subasta. | AC04 — Seguridad / AC03 — Disponibilidad | Obliga a envolver el adaptador de IA en un **`CircuitBreaker`** estricto ($150\text{ ms}$ timeout) con fallback heurístico en memoria y a ejecutar la detección de colusión (*Wash Trading* / *Shill Bidding*) en un **worker asíncrono no bloqueante**. |
| **DA07** | La plataforma debe adaptarse a la evolución exponencial de modelos de IA y a la aparición de Agentes Autónomos de Puja. | RF11 / RF13 / AC06 | Obliga a implementar un **`MultiModelRouter`** con cambio en caliente (*hot-swap*) de proveedores/modelos y un servicio de **Delegación de Agentes Autónomos (`AgentDelegationService`)** con límites presupuestarios estrictos (`max_cap`). |
| **DA08** | El sistema debe gobernar permisos estrictos por rol (`POSTOR`, `VENDEDOR`, `MARTILLERO`, `ADMIN`) y trazabilidad del encargado oficial. | RF08 / RF10 | Obliga a interceptores RBAC granulares (`RequireAuth`, `RequireRoles`), invalidación inmediata de caché al cambiar rol/estado y estructuras formales para `EncargadoSubasta` y `DetallesLote`. |

---

## 2. Resumen de Drivers y Decisiones Arquitectónicas que Responden

| Driver | Problema Crítico | Decisión Arquitectónica Aplicada |
|---|---|---|
| **DA01 — Escalabilidad Masiva** | Evitar $N$ suscripciones Redis por $N$ usuarios en una misma subasta | **ADR-007:** `AuctionRoomHub` en Go con multiplexación $O(1)$ y goroutines `readPump`/`writePump`. |
| **DA02 — Integridad & No Repudio** | Garantizar orden exacto, cero auto-puja y prueba criptográfica auditable | **ADR-003 & ADR-012:** Lock atómico Redis + `SELECT ... FOR UPDATE` + Cadena SHA-256 + EIP-712 + Merkle Tree. |
| **DA03 — Latencia Ultra-Baja** | Evitar consultas SQL repetitivas en cada request HTTP/WS autenticado | **ADR-009 & ADR-014:** Binario Go 1.23+ + `CachedUserRepository` con TTL e invalidación reactiva. |
| **DA04 — Custodia Híbrida** | Soportar garantías bancarias y colateral Web3 on-chain | **ADR-004 & ADR-012:** Puertos `EscrowPort` y `SmartContractEscrowPort` bajo Arquitectura Hexagonal. |
| **DA06 — IA Resiliente** | Proteger el *Hot Path* frente a latencias de APIs de LLMs | **ADR-005:** `CircuitBreaker` de $150\text{ ms}$ + `AsyncAnomalyDetector` por canales con buffer. |
| **DA07 — IA Exponencial** | Evolución constante de modelos LLM y pujas asistidas por IA | **ADR-013:** `MultiModelRouter` (Hot-Swap) + `AgentDelegationService` con estrategias acotadas. |
| **DA08 — Gobernanza & Fe Pública** | Separación de funciones entre Postor, Vendedor, Martillero y Admin | **ADR-002 & ADR-014:** RBAC estricto + gestión de `EncargadoSubasta` y `DetallesLote` periciales. |
