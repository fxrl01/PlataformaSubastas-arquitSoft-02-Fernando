# Laboratorio Integral de Pruebas 360° (`/tests`) — LiveBid

Este directorio agrupa todas las familias de validación automatizada de **Arquitectura de Software** alineadas a los escenarios de atributos de calidad de los Sprints 1 y 2.

## Mapa de Carpetas y Atributos de Calidad Validados

| Carpeta | Tipo de Prueba | Herramienta | Atributo de Calidad Demostrado | Comando Individual |
| :--- | :--- | :--- | :--- | :--- |
| **`tests/coverage/`** | Unitarias, Concurrencia `-race` y Cobertura | `go test -race` + `Jest` (`100%`) | **Mantenibilidad y Confiabilidad** | `bash tests/coverage/collect-coverage.sh` |
| **`tests/e2e/`** | End-to-End Multi-Navegador Real | Playwright (Chromium) | **Integridad Funcional y Trazabilidad Merkle** | `cd tests/e2e && npm run test:e2e` |
| **`tests/ui/`** | Interfaz, Accesibilidad WCAG 2.2, Temas y Móvil | Playwright (`375px` touch + Desktop) | **Usabilidad, Accesibilidad y Responsividad** | `cd tests/e2e && npm run test:ui` |
| **`tests/security/`** | Pentesting Dinámico OWASP (DAST) | Playwright API/DOM | **Seguridad (`401`, `403` RBAC, `429` Rate Limit, `413` DoS, XSS/SQLi)** | `cd tests/e2e && npm run test:security` |
| **`tests/load/`** | Carga Progresiva (`155`, `2,000` y `5,000` VUs) | Grafana k6 (`HTTP + WS`) | **Escalabilidad Horizontal (+2,000 usuarios)** | `docker run --rm --network host -v "$PWD/tests/load:/scripts" -e PROFILE=scale2000 grafana/k6 run /scripts/livebid-load.js` |
| **`tests/stress/`** | Estrés por Pico Súbito (*Spike / Stampede*) | Grafana k6 | **Consistencia Atómica bajo Estampida Anti-Sniping** | `docker run --rm --network host -v "$PWD/tests/stress:/scripts" grafana/k6 run /scripts/spike-antisniping.k6.js` |
| **`tests/resilience/`** | Tolerancia a Fallos / Chaos Engineering | Docker + Nginx + Bash | **Disponibilidad `99.95%` (`0%` Downtime ante caída de réplica)** | `bash tests/resilience/failover-ha.sh` |

## Ejecución Maestra en un Solo Comando

Con el clúster levantado (`docker compose up -d`), ejecuta desde la raíz del repositorio:

```bash
bash tests/run-all-tests.sh
```
