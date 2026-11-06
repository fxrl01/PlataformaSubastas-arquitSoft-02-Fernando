# Estilo Arquitectónico — LiveBid

> **Proyecto:** LiveBid — Plataforma de Subastas en Tiempo Real  
> **Capa:** Arquitectura de Software · **Versión:** 3.0

---

## 1. Estilo Seleccionado: Arquitectura Híbrida Hot/Cold Path + Orientada a Eventos (EDA) + Verificación Web3/IA

Debido a la naturaleza de alta frecuencia y competencia simultánea en los últimos segundos de una subasta en vivo, un estilo síncrono tradicional de N-capas resulta insuficiente. **LiveBid** combina cuatro estilos arquitectónicos complementarios:

1. **Separación *Hot Path* / *Cold Path*:** El camino crítico de validación, bloqueo atómico (`SETNX`) y difusión en vivo opera en memoria a través de Go Goroutines y Redis Pub/Sub ($< 25\text{ ms}$), mientras que la auditoría forense y el análisis profundo de fraude se ejecutan sin bloquear a los espectadores.
2. **Arquitectura Orientada a Eventos (EDA) con Multiplexación $O(1)$:** Cada puja aceptada o transición de estado emite un evento en el canal Redis de la subasta, el cual es consumido por una única suscripción compartida en el [`AuctionRoomHub`](file:///home/fernando/Softwar_Architect/PlataformaSubastas-arquitSoft-02-Fernando/tecnologia/livebid-backend/internal/adapters/realtime/room_hub.go) y difundido a todos los sockets locales.
3. **Arquitectura de Confianza Criptográfica (Web3 Trustless Overlay):** Integra verificación de firmas tipadas **EIP-712**, custodia en **Smart Contract Escrow** y construcción de **Árboles de Merkle (`MerkleRoot`)** sobre la cadena de hashes SHA-256 de PostgreSQL.
4. **Arquitectura Resiliente Asistida por IA Exponencial:** Incorpora un enrutador multi-modelo (`MultiModelRouter`) protegido por `CircuitBreaker` ($150\text{ ms}$), telemetría antifraude asíncrona (`AsyncAnomalyDetector`) y delegación de **Agentes Autónomos de Puja**.

---

## 2. Descripción de las Zonas Arquitectónicas

| Zona Arquitectónica | Responsabilidad Principal | Componentes Reales |
|---|---|---|
| **1. Edge, Gateway & Gobernanza RBAC** | Autenticar peticiones HTTP y handshakes WebSocket, verificar rol (`POSTOR`, `VENDEDOR`, `MARTILLERO`, `ADMIN`) y estado KYC en caché ($< 0.05\text{ ms}$). | `RequireAuth`, `RequireRoles`, `CachedUserRepository`, `AuthHandler`. |
| **2. Hot Path (Tiempo Real)** | Coordinar exclusión mutua por subasta, validar incrementos y *Anti-Sniping*, y difundir precios instantáneamente a miles de clientes. | `AuctionService`, `RedisLockAdapter` (`SETNX`), `RedisBidNotifier`, `AuctionRoomHub`. |
| **3. Capa de IA Exponencial** | Evaluar riesgo de puja en $< 150\text{ ms}$ con fallback local, detectar colusión en segundo plano y ejecutar pujas delegadas a agentes autónomos. | `MultiModelRouter`, `CircuitBreaker`, `GeminiRiskAdapter`, `AsyncAnomalyDetector`, `AgentDelegationService`. |
| **4. Capa Web3 Trustless** | Validar firmas criptográficas de billeteras EVM, verificar colateral on-chain y generar pruebas de inclusión Merkle. | `EIP712Verifier`, `EVMEscrowAdapter`, `Web3AuditService`. |
| **5. Cold Path (Datos & Auditoría Forense)** | Garantizar persistencia transaccional ACID, bloqueo de fila `SELECT ... FOR UPDATE`, ficha del encargado/lote y cadena inmutable SHA-256. | PostgreSQL 16 (`PostgresAuctionRepository`, `PostgresBidRepository`, `PostgresEscrowRepository`). |

---

## 3. Referencia Visual e Interactiva

- **Diagrama Estático Mermaid:** Ver [`arquitectura-inicial.md`](file:///home/fernando/Softwar_Architect/PlataformaSubastas-arquitSoft-02-Fernando/docs/02-arquitectura-software/arquitectura-inicial.md).
- **Explorador Dinámico Interactivo (C4 Multinivel + Simulador de Flujos):** Abrir [`livebid-arquitectura.html`](file:///home/fernando/Softwar_Architect/PlataformaSubastas-arquitSoft-02-Fernando/docs/02-arquitectura-software/livebid-arquitectura.html).
