# Atributos de Calidad y Requisitos No Funcionales (RNF) — LiveBid

> **Proyecto:** LiveBid — Plataforma de Subastas en Tiempo Real  
> **Capa:** Análisis de Sistema · **Versión:** 3.1  
> **Propósito:** Definir los atributos de calidad del dominio (`AC01`–`AC08`) y los requisitos no funcionales verificables (`RNF-01`–`RNF-08`) que rigen los 6 módulos del Monolito Modular Hexagonal.

---

## 1. Atributos de Calidad del Dominio (`AC01` – `AC08`)

Escenario analizado: durante un evento en vivo, $2,000$ o más espectadores pueden estar conectados simultáneamente (escalable a $5,000+$), varios pujando en la misma ventana de segundos sobre el mismo lote, mientras el estado de la sala y el precio líder se difunden en tiempo real sin condiciones de carrera.

| ID | Atributo de calidad | Escenario de calidad y criterio de éxito |
| :--- | :--- | :--- |
| **AC01** | **Rendimiento** | Una puja debe validarse y confirmarse en menos de $200\text{ ms}$ (procesamiento local $< 25\text{ ms}$) y el nuevo precio líder debe difundirse a todos los participantes de la sala en tiempo real, incluso con $2,000+$ usuarios conectados. |
| **AC02** | **Escalabilidad** | El sistema debe soportar $2,000+$ espectadores/pujadores concurrentes por evento (con proyección a $5,000+$) mediante el `AuctionRoomHub` en Go ($O(1)$ suscripción a Redis Pub/Sub por sala activa), balanceo `least_conn` en Nginx (`worker_connections 16384`) y pooling transaccional en `PgBouncer`, permitiendo escalar vertical y horizontalmente en **Microsoft Azure**. |
| **AC03** | **Disponibilidad y Resiliencia** | El sistema debe mantenerse disponible al menos el $99.5\%$ del tiempo durante una subasta en vivo y degradarse con elegancia (`Circuit Breaker` de $150\text{ ms}$) ante fallos o lentitud de servicios externos como Google Gemini IA. |
| **AC04** | **Seguridad (`Zero-Trust`)** | Los datos personales y financieros deben protegerse frente a accesos no autorizados: las tarjetas se tokenizan directamente en **Stripe** (`PCI-DSS`), los endpoints validan `JWT` + rol/estado activo (`RBAC`), y cada postor pasa por evaluación de riesgo y verificación de garantía en custodia (`Escrow`) antes de pujar. |
| **AC05** | **Integridad y Orden Verificable** | El orden en que se procesan las pujas debe ser estrictamente secuencial, atómico y trazable usando el timestamp oficial del servidor, bloqueo distribuido (`SETNX` en Redis), bloqueo pesimista (`SELECT ... FOR UPDATE` en PostgreSQL), cadena de hashes `SHA-256` y árbol de Merkle (`MerkleRoot`). |
| **AC06** | **Mantenibilidad y Evolución** | El núcleo de dominio (`internal/core/domain`) debe permanecer libre de dependencias de infraestructura para incorporar nuevas reglas de subasta, modelos de IA (`MultiModelRouter`) o pasarelas de pago/Web3 mediante puertos y adaptadores sin modificar la lógica central. |
| **AC07** | **Usabilidad en Tiempo Real** | El postor, martillero y vendedor deben percibir el precio líder, el cronómetro *Anti-Sniping* ($+30\text{ s}$ en los últimos $15\text{ s}$) y el estado de la sala sincronizados instantáneamente vía WebSockets sin necesidad de recargar la página. |
| **AC08** | **Portabilidad** | Todo el ecosistema (`livebid-gateway`, `livebid-api-1/2`, `livebid-pgbouncer`, `livebid-postgres`, `livebid-redis`) debe ejecutarse de forma idéntica en Linux, macOS, Windows y sobre infraestructura cloud **Microsoft Azure (Ubuntu VM `Standard_B2s` + Cloudflare Edge HTTPS/WSS)** mediante contenedores Docker y compilación multi-stage de Go. |

### ¿Por qué `AC05` y `AC07` no aparecen en un marketplace convencional?

En un marketplace tradicional (como un e-commerce de catálogo fijo), el orden exacto de milisegundos entre dos clics rara vez genera una disputa legal porque no existe competencia en vivo por un único lote irrepetible bajo conteo de martillo. En **LiveBid**, **el orden secuencial y auditable es el producto**: por ello `AC05` (integridad transaccional y orden verificable con cadena `SHA-256` y `MerkleRoot`) y `AC07` (sincronía en vivo por sala con extensión *Anti-Sniping*) son atributos de calidad fundacionales del dominio.

---

## 2. Catálogo de Requisitos No Funcionales (`RNF-01` – `RNF-08`)

| ID | Categoría | Requisito No Funcional | Métrica / Criterio de Verificación | Atributo Relacionado |
| :--- | :--- | :--- | :--- | :---: |
| **RNF-01** | **Seguridad Bancaria (`PCI-DSS` en Pagos)** | El Backend en Go y la base de datos PostgreSQL **jamás** deben recibir, procesar ni almacenar números completos de tarjeta de crédito (`PAN`) ni códigos de seguridad (`CVV`). Toda captura de tarjeta se realiza mediante intenciones de pago (`PaymentIntent` con `capture_method: "manual"` + `client_secret`) de **Stripe**. | `0` columnas de datos sensibles de tarjeta en PostgreSQL; solo se persiste el identificador `autorizacion_bancaria_id` (`pi_...`). | `AC04` |
| **RNF-02** | **Seguridad de Identidad y Roles (`Zero-Trust RBAC`)** | Todas las rutas protegidas deben validar un token `JWT` firmado (`HS256`) y comprobar en cada petición el rol (`ADMIN`, `MARTILLERO`, `VENDEDOR`, `POSTOR`) y estado actual del usuario (`ACTIVO`), con invalidación reactiva en `CachedUserRepository`. Las contraseñas se almacenan exclusivamente con `bcrypt`. | Rechazo `401/403` en `< 1 ms` ante tokens inválidos, usuarios `PENDIENTE`/`SUSPENDIDO`/`BLOQUEADO` o roles sin permiso. | `AC04` |
| **RNF-03** | **Simplicidad de Persistencia (Única Base de Datos ACID)** | Toda la información persistente del negocio debe almacenarse en **una única base de datos relacional (`PostgreSQL 16`)**, utilizando transacciones `ACID`, bloqueo pesimista `SELECT ... FOR UPDATE` y columnas `JSONB` para datos periciales (`encargado_json`, `detalles_lote_json`). | `1` único motor de base de datos relacional persistente en `docker-compose.yml`. | `AC05`, `AC06` |
| **RNF-04** | **Coordinación Efímera en Tiempo Real** | El uso de **Redis 7** debe limitarse exclusivamente a coordinación en memoria volátil (semáforo distribuido `SETNX` de `1.5s` con script Lua y difusión Pub/Sub), sin almacenar datos contables permanentes. | Pérdida de datos contables/legales = `0%` ante reinicio de Redis. | `AC02`, `AC03` |
| **RNF-05** | **Latencia de Sala en Vivo** | La validación de una puja y la difusión del nuevo precio líder por WebSocket (`AuctionRoomHub`) a los participantes conectados a la misma sala debe completarse en tiempo real. | Latencia de procesamiento local `< 25 ms` (P95 extremo a extremo `< 200 ms`). | `AC01`, `AC07` |
| **RNF-06** | **Resiliencia ante Servicios Externos (`Circuit Breaker`)** | Ninguna caída o lentitud del proveedor externo de Inteligencia Artificial (**Google Gemini**) debe impedir que un postor con garantía válida constituya su depósito o emita una puja. | Timeout máximo en ruta crítica hacia la IA con conmutación automática a evaluación heurística determinista local en `< 0.1 ms`. | `AC03` |
| **RNF-07** | **Idempotencia Financiera y de Pujas** | Tanto el envío de pujas como la retención de garantías deben ser idempotentes para evitar cobros o pujas duplicadas ante reintentos de red del cliente. | Cabecera `Idempotency-Key` (`uq_pujas_idempotency`) e índice único parcial `uq_garantia_retenida_subasta_postor` en PostgreSQL. | `AC05` |
| **RNF-08** | **Mantenibilidad para Desarrollo Continuo** | El backend debe mantenerse como un **Monolito Modular Hexagonal** en un solo binario Go (`livebid-backend`), donde `internal/core/domain` no dependa de paquetes externos de infraestructura. | `go test ./...` y `go vet ./...` ejecutables limpiamente en `< 5 segundos`. | `AC06`, `AC08` |
