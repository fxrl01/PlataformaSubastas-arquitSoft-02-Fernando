# Plan y Tareas SDD 021

- [x] Actualizar `.github/workflows/ci.yml` con soporte para `release-publica`, `workflow_dispatch`, `go-version-file` y resúmenes en `$GITHUB_STEP_SUMMARY`.
- [x] Actualizar `.github/workflows/cd.yml` para `main` y `release-publica`, publicación GHCR y despliegue condicional en Microsoft Azure VM.
- [x] Crear `.github/workflows/security-devsecops.yml` (control Zero-Trust de secretos/puertos, `govulncheck`, `npm audit` y escaneo de imagen con Trivy).
- [x] Crear `.github/workflows/load-test-k6.yml` (ejecución manual con selector `ci` / `scale2000` / `stress5000` y exportación de artefactos).
- [x] Validar sintaxis YAML y sincronizar limpiamente `main` y `release-publica`.
