# Requisitos No Funcionales (RNF) — LiveBid

> **Proyecto:** LiveBid — Plataforma de Subastas en Tiempo Real  
> **Capa:** Análisis de Sistema · **Versión:** 3.1  
> **Propósito:** Definir los estándares de calidad, seguridad bancaria, rendimiento y mantenibilidad que rigen los 6 módulos del sistema.

---

## 1. Catálogo de Requisitos No Funcionales

| ID | Categoría | Requisito No Funcional | Métrica / Criterio de Verificación |
| :--- | :--- | :--- | :--- |
| **RNF-01** | **Seguridad Bancaria (PCI-DSS en Pagos)** | El Backend en Go y la base de datos PostgreSQL **jamás** deben recibir, procesar ni almacenar números completos de tarjeta de crédito (PAN) ni códigos de seguridad (CVV). Toda captura de tarjeta debe realizarse en el Frontend mediante tokenización e intenciones de pago (`PaymentIntent` + `client_secret`) de **Stripe**. | `0` columnas de datos de tarjeta en PostgreSQL; solo se persiste el identificador `autorizacion_bancaria_id` (`pi_...`). |
| **RNF-02** | **Seguridad de Identidad y Roles (`Zero-Trust RBAC`)** | Todas las rutas protegidas deben validar un token `JWT` firmado (`HS256`) y comprobar en cada petición el rol y estado actual del usuario (`ACTIVO`). Las contraseñas deben almacenarse exclusivamente con hash `bcrypt`. | Rechazo `401/403` en `< 1 ms` ante tokens inválidos, usuarios `SUSPENDIDOS/BLOQUEADOS` o roles sin permiso. |
| **RNF-03** | **Simplicidad de Persistencia (1 Sola Base de Datos ACID)** | Toda la información persistente del negocio debe almacenarse en **una única base de datos relacional (`PostgreSQL 16`)**, utilizando transacciones `ACID` y columnas `JSONB` para datos semi-estructurados (`encargado_json`, `detalles_lote_json`). | `1` único motor de base de datos persistente en `docker-compose.yml`. |
| **RNF-04** | **Coordinación Efímera en Tiempo Real** | El uso de **Redis** debe limitarse exclusivamente a coordinación en memoria volátil (semáforo `SETNX` de `1.5s` y difusión Pub/Sub), garantizando que el backend en Go pueda arrancar y operar en entorno local con su fallback en memoria nativa si Redis no está disponible. | Pérdida de datos de negocio = `0%` ante reinicio de Redis. |
| **RNF-05** | **Latencia de Sala en Vivo** | La validación de una puja y la difusión del nuevo precio líder por WebSocket a los participantes conectados a la misma sala debe completarse en tiempo real. | Latencia de procesamiento local `< 25 ms` (P95 `< 100 ms`). |
| **RNF-06** | **Resiliencia ante Servicios Externos (`Circuit Breaker`)** | Ninguna caída o lentitud del proveedor de Inteligencia Artificial (**Google Gemini**) debe impedir que un postor con garantía válida realice una puja o constituya su depósito. | Timeout máximo de `150 ms` hacia la IA con conmutación automática a regla determinista local. |
| **RNF-07** | **Idempotencia Financiera y de Pujas** | Tanto el envío de pujas como la confirmación de garantías de Stripe deben ser idempotentes para evitar cobros o pujas duplicadas ante reintentos de red. | Uso de cabecera `Idempotency-Key` e índice único `uq_garantia_retenida_subasta_postor` en PostgreSQL. |
| **RNF-08** | **Mantenibilidad para Desarrollo Continuo** | El backend debe mantenerse como un **Monolito Modular Hexagonal** en un solo binario Go (`livebid-backend`), donde el paquete `internal/core/domain` no tenga dependencias de librerías externas (ni SQL, ni HTTP, ni Stripe SDK). | `go test ./...` y `go vet ./...` ejecutables en `< 5 segundos`. |

