# Especificación: Cobertura Automatizada 100 %, Contrato OpenAPI 3.0.3 y Validación de Atributos de Calidad

> **Feature ID:** `015-quality-assurance-openapi`  
> **Estado:** Completado

## Alcance

- **Pruebas Automatizadas con Jest al 100 %:** Incorporación de suite de pruebas unitarias y de integración con `Jest` + `ts-jest` sobre el módulo de referencia Clean Architecture (`tecnologia/boilerplate-clean-arquitecture`), exigiendo umbral obligatorio del `100 %` en `statements`, `branches`, `functions` y `lines`, más ampliación de pruebas `-race` en el motor Go (`tecnologia/livebid-backend`).
- **Documentación Interactiva de API (`OpenAPI 3.0.3` + `Swagger UI`):** Especificación canónica `openapi.yaml` y compilada `openapi.json` incrustadas en el binario Go (`//go:embed`), expuestas mediante `SwaggerHandler` (`/swagger`, `/docs`, `/api/v1/docs`, `/openapi.json`, `/openapi.yaml`), enrutadas en `Nginx` y protegidas con pruebas automáticas de paridad de contrato (`swagger_handler_test.go`).
- **Validación Empírica de Atributos de Calidad (`AC01`–`AC08`, `RNF-01`–`RNF-08`, `EQ-01`–`EQ-05`):** Consolidación trazable de evidencias reales de rendimiento (`k6`), concurrencia sin carreras (`go test -race` + PostgreSQL/Redis reales), flujo multi-navegador (`Playwright E2E`), aislamiento hexagonal (`Artículo I`) y seguridad bancaria `PCI-DSS` (`Artículo II`).

## Criterios de Aceptación

- **Dado** el comando `npm run test:coverage` en `tecnologia/boilerplate-clean-arquitecture`, **cuando** se ejecuta Jest, **entonces** finaliza con código `0` reportando `100 %` en `Statements`, `Branches`, `Functions` y `Lines` y generando `coverage/lcov-report/index.html`.
- **Dado** el clúster Docker activo detrás de Nginx (`http://localhost:3000`), **cuando** se solicita `GET /swagger`, `GET /openapi.json` o `GET /openapi.yaml`, **entonces** el servidor entrega Swagger UI v5 y la especificación `OpenAPI 3.0.3` cubriendo el `100 %` de las rutas HTTP/WS registradas en `cmd/server/main.go`.
- **Dado** el catálogo de atributos de calidad (`AC01`–`AC08`) y escenarios SEI (`EQ-01`–`EQ-05`), **cuando** se ejecutan las suites `Go -race`, `Jest`, `Playwright E2E` y `k6`, **entonces** cada criterio cuenta con evidencia cuantitativa verificada y documentada en `docs/06-validacion-atributos-calidad/`.
