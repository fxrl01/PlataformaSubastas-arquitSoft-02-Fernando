# Plan y Tareas SDD 022

- [x] Inspeccionar `playwright.config.js`, `nginx.conf` y middleware `security` de Go.
- [x] Crear `tests/ui/ui-accessibility-theme.spec.js` y `tests/security/owasp-dast.spec.js`, afinando `proxy_next_upstream` en `docker/nginx/nginx.conf`.
- [x] Crear `tests/stress/spike-antisniping.k6.js`, `tests/resilience/failover-ha.sh`, `tests/coverage/collect-coverage.sh`, `tests/run-all-tests.sh` y `tests/README.md`.
- [x] Integrar las nuevas suites en `.github/workflows/ci.yml`.
- [x] Ejecutar todas las suites localmente verificando `0` fallos y registrar commits limpios en `main` y `release-publica`.
