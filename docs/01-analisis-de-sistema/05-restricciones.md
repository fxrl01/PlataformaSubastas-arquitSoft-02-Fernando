# Restricciones — LiveBid

> **Proyecto:** LiveBid — Plataforma de Subastas en Tiempo Real  
> **Capa:** Análisis de Sistema · **Versión:** 3.0

| ID | Restricción | Descripción Técnica Actualizada |
|---|---|---|
| **RC01** | Control de versiones y Secretos | El código se gestiona con Git y GitHub; las claves privadas (`.env`, `GEMINI_API_KEY`, `JWT_SECRET`) están excluidas en `.gitignore` y nunca se suben al repositorio. |
| **RC02** | Lenguaje de Backend | Todos los servicios de dominio, pasarela en tiempo real, verificación Web3, enrutador de IA y detección asíncrona de anomalías se construyen en **Go (Golang 1.23+)** para garantizar baja huella de memoria (< 30 MB RAM) y concurrencia nativa. |
| **RC03** | Interfaz de Usuario | El cliente web de sala en vivo y panel operativo se sirve de forma integrada consumiendo la API REST (`/api/v1/*`) y WebSockets (`/ws`) con generación nativa de `Idempotency-Key`. |
| **RC04** | Comunicación en Tiempo Real | La comunicación entre cliente y backend utiliza API REST para comandos transaccionales y **WebSockets multiplexados (`AuctionRoomHub`)** sobre **Redis 7.2 Pub/Sub** para difusión en tiempo real ($O(1)$ suscripción por sala). |
| **RC05** | Persistencia Transaccional | La información contable y legal (usuarios, roles RBAC, subastas con `EncargadoSubasta` y `DetallesLote`, garantías, pujas y cadena SHA-256) se almacena en **PostgreSQL 16** con bloqueo pesimista `SELECT ... FOR UPDATE`. |
| **RC06** | Exclusión Mutua y Mensajería | La coordinación de concurrencia y difusión de eventos entre salas activas se implementa sobre **Redis 7.2** (`SETNX` con liberación Lua + canales Pub/Sub). |
| **RC07** | Contenedorización | Todo el ecosistema (`livebid-api-server`, `livebid-postgres`, `livebid-redis`) se despliega de forma reproducible mediante **Docker Compose** con compilación multi-stage. |
| **RC08** | Custodia Financiera Híbrida | La retención de garantías se abstrae mediante los puertos `EscrowPort` (fiduciario) y `SmartContractEscrowPort` (**Web3 EVM Escrow**), sin almacenar datos sensibles de tarjetas en el servidor. |
| **RC09** | Resiliencia de Inteligencia Artificial | Ninguna llamada a modelos externos de IA (**Google Gemini**) puede bloquear el *Hot Path* por más de $150\text{ ms}$; debe estar protegida obligatoriamente por un `CircuitBreaker` con fallback heurístico local. |
| **RC10** | Estilo Arquitectónico | El backend debe mantener separación estricta de **Arquitectura Hexagonal (Puertos y Adaptadores)**: `internal/core/domain`, `internal/core/services` e `internal/adapters/*`, verificada con `go test ./...` y `go vet ./...`. |
