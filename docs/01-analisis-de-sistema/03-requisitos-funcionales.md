# Requisitos funcionales — LiveBid

> **Proyecto:** LiveBid — Plataforma de Subastas en Tiempo Real  
> **Capa:** Análisis de Sistema · **Versión:** 3.0 (RBAC Completo, Encargado/Peritaje, Web3 & IA Exponencial)

---

## 1. Catálogo de Requisitos Funcionales

| ID | Módulo | Requisito Funcional |
|---|---|---|
| **RF01** | Catálogo y Lotes | El sistema debe permitir listar y consultar subastas incluyendo su estado, precio base, precio líder, **datos del encargado/martillero** y **ficha técnica/pericial del lote**. |
| **RF02** | Sala en Vivo (WS) | El sistema debe difundir actualizaciones de precio y estado en tiempo real a todos los clientes conectados a una subasta mediante un `AuctionRoomHub` multiplexado sobre Redis Pub/Sub ($O(1)$ suscripción por sala). |
| **RF03** | Motor de Pujas | El sistema debe permitir a un `POSTOR` activo enviar pujas en vivo con protección de idempotencia (`Idempotency-Key`) y bloqueo distribuido atómico (`SETNX` + `SELECT ... FOR UPDATE`). |
| **RF04** | Reglas de Dominio | El sistema debe validar que la subasta esté en `ACTIVA` o `EN_CONTEO`, que el postor no sea el líder actual (`ErrPostorIgual`), que el monto supere el incremento mínimo por tramo y extender automáticamente $+30\text{ s}$ si entra en los últimos $15\text{ s}$ (*Anti-Sniping*). |
| **RF05** | Custodia / Escrow | El sistema debe verificar que el postor cuente con una garantía activa (`RETENIDA`) en custodia fiduciaria o en **Smart Contract Escrow Web3** antes de aceptar cualquier puja. |
| **RF06** | Auditoría SHA-256 | El sistema debe encadenar criptográficamente cada puja y evento en PostgreSQL (`hash_auditoria` SHA-256 enlazado con `hash_anterior`) dentro de la misma transacción ACID. |
| **RF07** | Gestión de Subasta | El sistema debe permitir crear subastas (`VENDEDOR`, `MARTILLERO`, `ADMIN`), activarlas (`ACTIVA`), iniciar el conteo de martillo (`EN_CONTEO`) y cerrarlas manual o automáticamente al vencer `fin_programado`. |
| **RF08** | Encargado y Lote | El sistema debe permitir subir y actualizar los **detalles del encargado de la subasta** (`martillero_id`, nombre, cargo, licencia/colegiatura, contacto) y los **detalles del lote** (categoría, ubicación, precio de reserva, imágenes, certificado pericial y especificaciones). |
| **RF09** | Identidad y Registro | El sistema debe permitir el registro de usuarios con solicitud de rol y datos de perfil, además de garantizar el *bootstrap* automático de un usuario `ADMIN` inicial. |
| **RF10** | Administración RBAC | El sistema debe permitir al `ADMIN` listar usuarios filtrados por estado/rol, activar cuentas (`kyc_aprobado`), asignar roles (`POSTOR`, `VENDEDOR`, `MARTILLERO`, `ADMIN`) y suspender o bloquear usuarios con invalidación inmediata en caché. |
| **RF11** | Riesgo IA + Resiliencia | El sistema debe evaluar el score de confianza de cada puja mediante **Google Gemini AI** a través de un `CircuitBreaker` de $150\text{ ms}$ con fallback heurístico determinista y un **Multi-Model AI Router** reconfigurable en caliente. |
| **RF12** | Telemetría Antifraude | El sistema debe analizar de forma asíncrona (fuera del camino crítico) el flujo de pujas para detectar patrones de *Bot Sniping*, *Wash Trading* y *Shill Bidding*, exponiendo alertas en tiempo real. |
| **RF13** | Agentes IA Autónomos | El sistema debe permitir a un postor delegar un **Agente Autónomo de Puja IA** con techo presupuestario inquebrantable (`max_cap`) y estrategia configurable (`CONSERVADORA`, `EQUILIBRADA`, `SNIPER_ANTI_EXTENSION`). |
| **RF14** | Auditoría Web3 Merkle | El sistema debe verificar firmas tipadas **EIP-712**, calcular la **Raíz de Merkle (`MerkleRoot`)** de todas las pujas de una subasta, generar pruebas de inclusión (`MerkleProof`) y emitir recibos de liquidación on-chain al cierre. |

---

## 2. Matriz de Trazabilidad: Historias de Usuario $\leftrightarrow$ Requisitos Funcionales

| Historia de Usuario | Requisitos Funcionales Asociados | Endpoints / Componentes Principales |
|---|---|---|
| **HU01** Explorar subastas y peritaje | RF01, RF08 | `GET /api/v1/subastas`, `GET /api/v1/subastas/{id}` |
| **HU02, HU04** Sala en vivo y broadcast WS | RF02, RF04 | `GET /ws?subasta_id=...`, `AuctionRoomHub` |
| **HU03** Enviar puja atómica e idempotente | RF03, RF04, RF05, RF06 | `POST /api/v1/subastas/{id}/pujas`, `AuctionService.ProcesarPuja` |
| **HU05** Delegar Agente Autónomo IA | RF13 | `POST /api/v1/subastas/{id}/agentes-ia/delegar`, `POST .../auto-puja` |
| **HU06, HU17** Verificar Merkle Proof y Hash Chain | RF06, RF14 | `GET /api/v1/subastas/{id}/auditoria-web3`, `Web3AuditService` |
| **HU07** Registro de usuario y solicitud KYC | RF09 | `POST /api/v1/auth/register`, `GET /api/v1/auth/me` |
| **HU08, HU11, HU12** Subir encargado y ficha del lote | RF07, RF08 | `POST /api/v1/subastas`, `PUT /api/v1/subastas/{id}/encargado`, `PUT .../detalles-lote` |
| **HU09, HU10, HU13** Activar, conteo y cierre con liquidación | RF05, RF07, RF14 | `POST /api/v1/subastas/{id}/activar`, `/conteo`, `/cerrar` |
| **HU14, HU15** Gestión Admin de usuarios y roles | RF09, RF10 | `GET /api/v1/admin/users`, `PATCH .../role`, `PATCH .../status`, `PATCH .../activate` |
| **HU16** Inteligencia antifraude y Hot-Swap IA | RF11, RF12 | `GET /api/v1/subastas/{id}/inteligencia-ia`, `PATCH /api/v1/admin/ia/modelo` |
