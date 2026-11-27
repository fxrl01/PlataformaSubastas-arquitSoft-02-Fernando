# Resumen Consolidado de Cobertura — LiveBid

| Motor / Capa Arquitectónica | Herramienta | Resultado Medido | Artefacto HTML Local |
| :--- | :--- | :---: | :--- |
| **Clean Architecture Core (TypeScript)** | Jest 30 + Experimental VM | **100% Global** (Stmts, Branch, Funcs, Lines) | `tests/coverage/reports/jest-lcov-report/index.html` |
| **Backend Producción (Go 1.23 + `-race`)** | `go test -race -cover` | **73.8%** total (`domain` 100%, `web3` 100%, `payments` 89.4%, `ai` 88.9%) | `tests/coverage/reports/go-coverage.html` |
