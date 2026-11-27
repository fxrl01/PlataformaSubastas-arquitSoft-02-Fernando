# Escenarios de atributos de calidad — LiveBid

> **Proyecto:** LiveBid — Plataforma de Subastas en Tiempo Real  
> **Estándar:** Software Engineering Institute (SEI) · Escenarios de 6 partes  
> **Versión:** 3.0 · **Estado:** Alineado con Implementación Go 1.23+ + Web3 + IA Exponencial

---

## EQ-01. Rendimiento: Latencia de procesamiento y difusión multiplexada $O(1)$

| Elemento | Especificación en LiveBid |
|---|---|
| **Atributo de calidad** | Rendimiento y Escalabilidad en Tiempo Real (*Performance*) |
| **Fuente** | Postores humanos y Agentes Autónomos de IA compitiendo en una sala activa |
| **Estímulo** | Ráfaga de pujas durante los últimos 15 segundos del conteo (*franja Anti-Sniping*) |
| **Artefacto** | `AuctionService` + `RedisLockAdapter` (`SETNX`) + `AuctionRoomHub` (1 suscripción Redis Pub/Sub por sala) |
| **Entorno** | Sala en vivo con $2,000+$ espectadores conectados simultáneamente (escalable a $5,000+$) sobre Microsoft Azure + Cloudflare WSS |
| **Respuesta esperada** | El motor valida la oferta, extiende $+30\text{ s}$ el cierre, registra el nuevo líder y difunde el evento a todos los clientes compartiendo una única suscripción Redis por sala |
| **Medida** | **P95 de confirmación < 200 ms**; difusión local por `AuctionRoomHub` **< 25 ms**; **1 sola suscripción Redis Pub/Sub por subasta activa ($O(1)$)** |

---

## EQ-02. Integridad y No Repudio: Resolución atómica, Cadena SHA-256 y Árbol de Merkle Web3

| Elemento | Especificación en LiveBid |
|---|---|
| **Atributo de calidad** | Integridad Transaccional y Verificabilidad Criptográfica (*Integrity & Non-Repudiation*) |
| **Fuente** | Múltiples postores enviando ofertas simultáneas o intentando auto-superarse |
| **Estímulo** | Dos ofertas concurrentes al mismo milisegundo o intento del líder actual de pujar contra sí mismo |
| **Artefacto** | `RedisLockAdapter` (`SETNX`) + `PostgresBidRepository.RegistrarPujaAtomica` (`SELECT ... FOR UPDATE`) + `Web3AuditService` (`ComputeMerkleRoot`) |
| **Entorno** | Conteo regresivo final activo con alta contención |
| **Respuesta esperada** | El sistema serializa el acceso, impide `postor_lider_id == postor_id`, asigna correlativo único, encadena el `hash_auditoria` SHA-256 y actualiza la `MerkleRoot` verificable en $O(\log_2 N)$ |
| **Medida** | **0 condiciones de carrera**; **100% de pujas encadenadas con SHA-256 y verificables mediante `MerkleProof` y firma `EIP-712`** |

---

## EQ-03. Disponibilidad y Resiliencia IA: Degradación elegante con `CircuitBreaker` y `MultiModelRouter`

| Elemento | Especificación en LiveBid |
|---|---|
| **Atributo de calidad** | Disponibilidad y Tolerancia a Fallos (*Availability & Fault Tolerance*) |
| **Fuente** | API externa de Google Gemini AI o red externa con degradación/latencia |
| **Estímulo** | El proveedor LLM tarda más de $150\text{ ms}$ en responder o devuelve error durante una puja en vivo |
| **Artefacto** | `CircuitBreaker` (`internal/adapters/ai/circuit_breaker.go`) + `MultiModelRouter` (`multi_model_router.go`) |
| **Entorno** | Subasta en vivo con ofertas de alto valor |
| **Respuesta esperada** | El `CircuitBreaker` cancela la espera a los $150\text{ ms}$, abre el circuito tras alcanzar el umbral de fallos y evalúa el riesgo con la heurística determinista local en $< 0.1\text{ ms}$; además permite al `ADMIN` hacer *hot-swap* de modelo en caliente |
| **Medida** | **0% de pujas abortadas por caída de IA externa**; conmutación de modelo en **0 ms de inactividad (*zero downtime*)** |

---

## EQ-04. Seguridad Zero-Trust: Gobernanza RBAC, Invalidación de Caché y Custodia Escrow

| Elemento | Especificación en LiveBid |
|---|---|
| **Atributo de calidad** | Seguridad, Control de Acceso y Detección de Fraude (*Security & Zero Trust*) |
| **Fuente** | Usuario sin garantía activa, cuenta recién suspendida por `ADMIN`, o colusión entre postores |
| **Estímulo** | Intento de operar tras cambio de rol/suspensión, puja sin garantía o patrón de *Wash Trading* |
| **Artefacto** | `RequireAuth` + `RequireRoles` + `CachedUserRepository` + `EscrowPort` / `EVMEscrowAdapter` + `AsyncAnomalyDetector` |
| **Entorno** | Operación continua en producción |
| **Respuesta esperada** | `CachedUserRepository` invalida al instante cualquier cuenta modificada por el `ADMIN`; las pujas sin garantía son rechazadas con `422/403`; el detector asíncrono emite alertas de *Bot Sniping*, *Wash Trading* y *Shill Bidding* |
| **Medida** | **100% de rechazo a usuarios no activos o sin garantía**; **invalidación de caché RBAC en < 1 ms** |

---

## EQ-05. Evolución Exponencial: Delegación de Agentes IA y Desacoplamiento Hexagonal

| Elemento | Especificación en LiveBid |
|---|---|
| **Atributo de calidad** | Mantenibilidad y Adaptabilidad Exponencial (*Maintainability & Evolvability*) |
| **Fuente** | Postores que delegan Agentes Autónomos de IA o integración de nuevos proveedores LLM/Web3 |
| **Estímulo** | Un postor delega un agente con estrategia `SNIPER_ANTI_EXTENSION` y techo `max_cap`, o se incorpora un nuevo modelo en `MultiModelRouter` |
| **Artefacto** | `AgentDelegationService` + `MultiModelRouter` + Puertos en `internal/core/domain/ports.go` |
| **Entorno** | Evolución continua del sistema |
| **Respuesta esperada** | El agente puja automáticamente respetando invariablemente `max_cap` y pasando por las mismas validaciones atómicas del dominio |
| **Medida** | **0 violaciones del techo `max_cap`**; **100% de pruebas unitarias y de integración en verde (`go test ./...`)** |
