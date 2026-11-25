# Tareas: Cobertura Automatizada 100 %, Contrato OpenAPI 3.0.3 y Validación de Atributos de Calidad

> **Estado:** Completado

- [x] Configurar Jest + `ts-jest` y alcanzar `100 %` de cobertura (`Statements`, `Branches`, `Functions`, `Lines`) con reporte HTML LCOV.
- [x] Ampliar pruebas unitarias e integración con detector `-race` en el backend Go (`domain` y `web3` al `100 %`, `internal/...` > `84 %`).
- [x] Construir especificación `OpenAPI 3.0.3` (`30 paths`, `32 operaciones`, `41 esquemas`) e interfaz `Swagger UI` embebidas con `//go:embed`.
- [x] Enrutar `/swagger`, `/docs`, `/openapi.json` y `/openapi.yaml` en Go y Nginx Gateway, y añadir guardián de contrato en `swagger_handler_test.go`.
- [x] Endurecer `StripeAdapter` para activar modo Sandbox automático ante claves de plantilla (`replace_with`) en `.env`.
- [x] Verificar extremo a extremo con PostgreSQL 16 + Redis 7 reales, Playwright E2E (2 navegadores + Swagger) y métricas `k6`, documentando las evidencias en `docs/06-validacion-atributos-calidad/evidencias-validacion-atributos-calidad.md`.
