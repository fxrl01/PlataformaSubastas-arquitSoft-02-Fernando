# Plan: Cobertura Automatizada 100 %, Contrato OpenAPI 3.0.3 y Validación de Atributos de Calidad

1. Configurar `jest.config.cjs` con `coverageThreshold.global = 100` en `tecnologia/boilerplate-clean-arquitecture` y cubrir al 100 % las capas `dominio`, `aplicacion`, `infraestructura` y `presentacion` en `src/app/__tests__/`.
2. Reforzar la cobertura y detección de condiciones de carrera (`-race`) en `tecnologia/livebid-backend` sobre `domain`, `web3`, `security`, `ai`, `payments`, `realtime`, `services`, `handlers` y `cmd/server`.
3. Crear el adaptador `internal/adapters/docs` (`openapi.yaml`, `openapi.json`, `swagger.html`, `embed.go`) y el manejador `SwaggerHandler` (`internal/adapters/handlers/swagger_handler.go`), cableándolo en `cmd/server/main.go`, `docker/nginx/nginx.conf` y `public/index.html`.
4. Implementar guardián automático de contrato en `swagger_handler_test.go` (validación de `$ref`, `operationId` únicos y paridad 1-a-1 contra las rutas de `cmd/server/main.go`) y aserciones E2E en `tests/e2e/livebid.e2e.spec.js`.
5. Consolidar la matriz formal de evidencias y métricas verificadas para `AC01`–`AC08`, `RNF-01`–`RNF-08`, `EQ-01`–`EQ-05` y Artículos I–VI de la Constitución SDD en `docs/06-validacion-atributos-calidad/evidencias-validacion-atributos-calidad.md`.
