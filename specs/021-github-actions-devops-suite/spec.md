# SDD 021 — Suite DevOps y DevSecOps en GitHub Actions (CI, CD Azure, Seguridad y k6 2000+ VUs)

## 1. Contexto y Objetivo
Habilitar la ejecución automática y manual de pipelines en GitHub Actions tanto en `main` como en `release-publica`, cubriendo pruebas unitarias/integración, entrega continua hacia GHCR y Microsoft Azure, auditoría DevSecOps OWASP/contenedores y laboratorio interactivo de carga con k6 (`155`, `2,000` y `5,000` usuarios virtuales).

## 2. Criterios de Aceptación
1. **`ci.yml`:** Se ejecuta automáticamente en `push` y `pull_request` sobre `main` y `release-publica` (además de `workflow_dispatch`), publicando resúmenes en `$GITHUB_STEP_SUMMARY` y artefactos de Go `-race`, Jest `100%`, Playwright E2E y k6.
2. **`cd.yml`:** Construye y publica la imagen multi-etapa en `ghcr.io` en minúsculas sobre `main` y `release-publica`, e incluye el job parametrizado de despliegue SSH para la VM de Microsoft Azure (`ENABLE_AZURE_DEPLOY`).
3. **`security-devsecops.yml`:** Verifica ausencia de secretos filtrados, aislamiento `127.0.0.1` de PostgreSQL/Redis, usuario sin privilegios (`livebiduser`) en Docker, auditoría de dependencias Go/Node y escaneo de contenedor con Trivy.
4. **`load-test-k6.yml`:** Permite lanzar pruebas de carga bajo demanda desde la interfaz web de GitHub (`workflow_dispatch`) eligiendo el perfil (`ci`, `scale2000`, `stress5000`) y descargando las evidencias JSON.
