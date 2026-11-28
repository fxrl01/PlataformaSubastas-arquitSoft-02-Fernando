#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# EJECUTOR MAESTRO 360° DE CALIDAD DE SOFTWARE — LIVEBID
# Ejecuta las 6 validaciones automatizadas sobre código y clúster Docker activo:
#   1. Cobertura Unitaria + Integración + Race Detector (Go & Jest 100%)
#   2. End-to-End Transaccional con 2 Navegadores (Playwright)
#   3. UI, Accesibilidad WCAG 2.2, Modo Claro/Oscuro y Móvil 375px (Playwright)
#   4. Seguridad Dinámica OWASP DAST: 401, 403, 429, 413, XSS/SQLi (Playwright)
#   5. Estrés por Pico Súbito / Anti-Sniping Stampede (k6)
#   6. Tolerancia a Fallos y Alta Disponibilidad 0% Downtime (Failover HA)
# ==============================================================================

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

echo "======================================================================"
echo " [1/6] COBERTURA UNITARIA, CONCURRENCIA (-race) Y CLEAN ARCHITECTURE"
echo "======================================================================"
bash "${ROOT_DIR}/tests/coverage/collect-coverage.sh"

echo "======================================================================"
echo " [2/6] END-TO-END TRANSACCIONAL (2 NAVEGADORES + WS + STRIPE + MERKLE)"
echo "======================================================================"
(cd "${ROOT_DIR}/tests/e2e" && npm run test:e2e)

echo "======================================================================"
echo " [3/6] UI, ACCESIBILIDAD WCAG, MODO CLARO/OSCURO Y MÓVIL TÁCTIL"
echo "======================================================================"
(cd "${ROOT_DIR}/tests/e2e" && npm run test:ui)

echo "======================================================================"
echo " [4/6] SEGURIDAD DINÁMICA OWASP (DAST: RBAC, RATE LIMIT 429, DOS, XSS)"
echo "======================================================================"
(cd "${ROOT_DIR}/tests/e2e" && npm run test:security)

echo "======================================================================"
echo " [5/6] ESTRÉS POR PICO SÚBITO / ANTI-SNIPING STAMPEDE (k6)"
echo "======================================================================"
mkdir -p "${ROOT_DIR}/tests/stress/results"
docker run --rm --network host -u "$(id -u):$(id -g)" \
  -v "${ROOT_DIR}/tests/stress:/scripts" \
  grafana/k6 run --summary-export=/scripts/results/spike-summary.json /scripts/spike-antisniping.k6.js

echo "======================================================================"
echo " [6/6] RESILIENCIA Y ALTA DISPONIBILIDAD (FAILOVER NGINX 0% DOWNTIME)"
echo "======================================================================"
bash "${ROOT_DIR}/tests/resilience/failover-ha.sh"

echo "======================================================================"
echo " TODAS LAS 6 SUITES DE CALIDAD 360° APROBARON CON CERO ERRORES"
echo "======================================================================"
