# Documentacion de API mediante OpenAPI 3.0.3 y Swagger UI (LiveBid)

Este documento describe los artefactos OpenAPI implementados para **LiveBid** y detalla paso a paso el proceso tecnico de construccion, integracion hexagonal y validacion automatizada de contratos.

---

## 1. Artefactos Generados

| Artefacto | Ruta en el Repositorio | Proposito |
| :--- | :--- | :--- |
| **Contrato Canonico OpenAPI 3.0.3 (YAML)** | [`openapi.yaml`](../../tecnologia/livebid-backend/internal/adapters/docs/openapi.yaml) | Fuente unica de verdad del contrato HTTP/WS (30 rutas, 32 operaciones, 41 esquemas y seguridad JWT `BearerAuth`). |
| **Contrato Compilado OpenAPI 3.0.3 (JSON)** | [`openapi.json`](../../tecnologia/livebid-backend/internal/adapters/docs/openapi.json) | Derivado determinista desde `openapi.yaml` para carga instantanea de Swagger UI y validadores JSON. |
| **Interfaz Web Swagger UI** | [`swagger.html`](../../tecnologia/livebid-backend/internal/adapters/docs/swagger.html) | Portal interactivo basado en Swagger UI v5 con boton **Authorize**, filtro y consola **Try it out**. |
| **Embebido en Binario Go (`//go:embed`)** | [`embed.go`](../../tecnologia/livebid-backend/internal/adapters/docs/embed.go) | Incrusta `openapi.yaml`, `openapi.json` y `swagger.html` dentro del binario `livebid-server` sin dependencias de disco en runtime. |
| **Handler HTTP de Documentacion** | [`swagger_handler.go`](../../tecnologia/livebid-backend/internal/adapters/handlers/swagger_handler.go) | Expone `/swagger`, `/docs`, `/api/v1/docs`, `/openapi.json`, `/api/v1/openapi.json`, `/openapi.yaml` y `/api/v1/openapi.yaml`. |
| **Cableado en Router Principal Go** | [`main.go`](../../tecnologia/livebid-backend/cmd/server/main.go) | Invoca `handlers.RegisterSwaggerRoutes(mux)` manteniendo CORS habilitado (`PUT`, `PATCH`, headers `Idempotency-Key`, `X-Web3-*`). |
| **Enrutamiento en Reverse Proxy Nginx** | [`nginx.conf`](../../docker/nginx/nginx.conf) | Redirige `/swagger`, `/docs`, `/openapi.json` y `/openapi.yaml` hacia el cluster Go (`livebid_api_cluster`). |
| **Acceso Directo desde Frontend SPA** | [`index.html`](../../tecnologia/boilerplate-clean-arquitecture/public/index.html) | Incluye el enlace `API Docs` hacia `/swagger` en la barra superior de navegacion. |
| **Suite de Pruebas de Contrato en Go** | [`swagger_handler_test.go`](../../tecnologia/livebid-backend/internal/adapters/handlers/swagger_handler_test.go) | Verifica codigos HTTP, `Content-Type`, `$ref` validos, `operationId` unicos y paridad del 100 % con las rutas de `main.go`. |

---

## 2. Como se Realizo el Trabajo (Paso a Paso del Proceso)

### Paso 1 — Auditoria de Rutas Reales, DTOs y Seguridad
Se inspeccionaron directamente las capas del backend productivo en Go ([`cmd/server/main.go`](../../tecnologia/livebid-backend/cmd/server/main.go)), los adaptadores HTTP/WebSocket ([`internal/adapters/handlers/`](../../tecnologia/livebid-backend/internal/adapters/handlers/auth_handler.go), [`websocket_handler.go`](../../tecnologia/livebid-backend/internal/adapters/realtime/websocket_handler.go)) y los modelos/servicios del nucleo ([`internal/core/domain/`](../../tecnologia/livebid-backend/internal/core/domain/auction.go), [`internal/core/services/`](../../tecnologia/livebid-backend/internal/core/services/bid_service.go)).

De esta auditoria se extrajeron con exactitud:
- **32 operaciones totales sobre 30 paths** agrupadas en los **6 modulos arquitectonicos (M1–M6)** + Salud (`GET /health`).
- Los nombres reales de campos JSON (`json:"..."`) de cada solicitud y respuesta, evitando discrepancias entre codigo y documentacion.
- Los mecanismos reales de seguridad y cabeceras especiales:
  - JWT `BearerAuth` (`Authorization: Bearer <token>`, validez de 15 minutos + rotacion en `/api/v1/auth/refresh`).
  - Cabecera de idempotencia `Idempotency-Key` en `POST /api/v1/subastas/{id}/pujas`.
  - Cabeceras opcionales Web3 (`X-Web3-Wallet`, `X-Web3-Signature`, `X-Web3-Nonce`, `X-Web3-Deadline`), mandato de agente IA (`X-AI-Agent-Mandate`) y firma HMAC de webhook (`Stripe-Signature`).

> **Nota arquitectonica:** El prototipo historico en TypeScript ([`tecnologia/boilerplate-clean-arquitecture/src/main.ts`](../../tecnologia/boilerplate-clean-arquitecture/src/main.ts)) implementa 5 rutas iniciales de validacion en memoria (`GET /health`, `GET /api/v1/subastas/:id`, `POST .../garantia`, `POST .../puja`, `POST .../liquidar`). El contrato OpenAPI 3.0.3 oficial documenta las rutas productivas completas del motor Go en cluster (`/api/v1/subastas/{id}/garantias`, `.../pujas`, `.../close`, WebSocket `/ws/subastas/{id}`, etc.).

### Paso 2 — Diseno Hexagonal Docs-as-Code (`internal/adapters/docs`)
En lugar de depender de archivos sueltos vulnerables a desfase en el contenedor minimalista de Alpine (~18-20 MB), se adopto un adaptador autocontenido con `//go:embed`:
1. Se escribio [`openapi.yaml`](../../tecnologia/livebid-backend/internal/adapters/docs/openapi.yaml) siguiendo la especificacion **OpenAPI 3.0.3**.
2. Se compilo deterministamente a [`openapi.json`](../../tecnologia/livebid-backend/internal/adapters/docs/openapi.json) para alimentar Swagger UI y consumidores automaticos.
3. Se disenho [`swagger.html`](../../tecnologia/livebid-backend/internal/adapters/docs/swagger.html) con Swagger UI v5 apuntando a `/api/v1/openapi.json`, opcion `persistAuthorization: true` (conserva el JWT al recargar), filtro de endpoints y enlaces de descarga directa a JSON/YAML.

### Paso 3 — Exposicion HTTP en Go y Reverse Proxy en Nginx
- El handler [`SwaggerHandler`](../../tecnologia/livebid-backend/internal/adapters/handlers/swagger_handler.go) registra rutas duales (con y sin prefijo `/api/v1`) para servir tanto detras de Nginx (`http://localhost:3000/swagger`) como en ejecucion directa de Go (`http://localhost:8080/swagger`).
- En [`nginx.conf`](../../docker/nginx/nginx.conf), se anadio un bloque `location ~ ^/(swagger/?|docs/?|openapi\.(json|yaml))$` antes del fallback SPA (`try_files $uri $uri/ /index.html`) para evitar que Nginx devolviera `index.html` al abrir `/swagger`. Las rutas `/api/v1/docs`, `/api/v1/openapi.json` y `/api/v1/openapi.yaml` ya quedan cubiertas automaticamente por `location /api/`.

### Paso 4 — Validacion Automatizada de Contrato (Contract Guard)
En [`swagger_handler_test.go`](../../tecnologia/livebid-backend/internal/adapters/handlers/swagger_handler_test.go) se implementaron pruebas automatizadas que corren con `go test -race ./...`:
- Comprueban codigos `200 OK`, `Content-Type` correcto (`text/html`, `application/yaml`, `application/json`) y presencia de `SwaggerUIBundle`.
- Verifican unicidad de todos los `operationId` e integridad de cada `$ref` local dentro de `#/components/*`.
- Analizan directamente [`cmd/server/main.go`](../../tecnologia/livebid-backend/cmd/server/main.go) y fallan el build/CI si se agrega una ruta HTTP/WS nueva en Go que no haya sido documentada en OpenAPI.

---

## 3. Como Visualizar y Probar la Documentacion

### Opcion A: Con el Cluster Docker Completo (`docker compose up --build -d`)
- **Swagger UI:** `http://localhost:3000/swagger` (o `http://localhost:3000/api/v1/docs`)
- **Contrato JSON:** `http://localhost:3000/openapi.json`
- **Contrato YAML:** `http://localhost:3000/openapi.yaml`

### Opcion B: Ejecutando Swagger UI desde el Boton de Autenticacion
1. Abra `POST /api/v1/auth/login` en **M1 - Identidad, Auth & RBAC** y envie las credenciales del usuario administrador o de prueba (por ejemplo `admin@livebid.pe` / `AdminLiveBid2026!`).
2. Copie el campo `access_token` de la respuesta `200 OK`.
3. Pulse **Authorize** (arriba a la derecha en Swagger UI), pegue el token JWT y confirme.
4. Ejecute directamente con **Try it out** operaciones protegidas por RBAC (`POST /api/v1/subastas`, `POST /api/v1/subastas/{id}/garantias/stripe-intent`, `POST /api/v1/subastas/{id}/pujas`, `GET /api/v1/subastas/{id}/auditoria-web3`).
