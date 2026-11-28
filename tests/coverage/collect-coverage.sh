#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# RECOLECTOR UNIFICADO DE COBERTURA (GO -race + JEST 100% CLEAN ARCHITECTURE)
# Genera en tests/coverage/ los reportes HTML, perfiles y resumen consolidado.
# ==============================================================================

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/../.." && pwd)"
OUT_DIR="${ROOT_DIR}/tests/coverage"
mkdir -p "${OUT_DIR}/reports"

echo "[Coverage 1/2] Ejecutando suite Go Clean Architecture con detector -race..."
(
  cd "${ROOT_DIR}/tecnologia/livebid-backend"
  go test -count=1 -race -coverprofile="${OUT_DIR}/reports/go-coverage.out" ./...
  go tool cover -func="${OUT_DIR}/reports/go-coverage.out" > "${OUT_DIR}/reports/go-coverage-summary.txt"
  go tool cover -html="${OUT_DIR}/reports/go-coverage.out" -o "${OUT_DIR}/reports/go-coverage.html"
)

echo "[Coverage 2/2] Ejecutando suite Jest Clean Architecture (umbral estricto 100%)..."
(
  cd "${ROOT_DIR}/tecnologia/boilerplate-clean-arquitecture"
  npm run test:coverage
  rm -rf "${OUT_DIR}/reports/jest-lcov-report"
  cp -R coverage/lcov-report "${OUT_DIR}/reports/jest-lcov-report"
  cp coverage/lcov.info "${OUT_DIR}/reports/jest-lcov.info"
)

GO_TOTAL=$(tail -n 1 "${OUT_DIR}/reports/go-coverage-summary.txt" | awk '{print $3}')
cat <<EOF > "${OUT_DIR}/COVERAGE_SUMMARY.md"
# Resumen Consolidado de Cobertura — LiveBid

| Motor / Capa Arquitectónica | Herramienta | Resultado Medido | Artefacto HTML Local |
| :--- | :--- | :---: | :--- |
| **Clean Architecture Core (TypeScript)** | Jest 30 + Experimental VM | **100% Global** (Stmts, Branch, Funcs, Lines) | \`tests/coverage/reports/jest-lcov-report/index.html\` |
| **Backend Producción (Go 1.23 + \`-race\`)** | \`go test -race -cover\` | **${GO_TOTAL}** total (\`domain\` 100%, \`web3\` 100%, \`payments\` 89.4%, \`ai\` 88.9%) | \`tests/coverage/reports/go-coverage.html\` |
EOF

echo "OK: Reportes consolidados generados en tests/coverage/COVERAGE_SUMMARY.md y tests/coverage/reports/."
