# Especificación: Liquidación de garantías al cerrar subastas

> **Feature ID:** `006-escrow-settlement`
> **Estado:** Implementado para el MVP tradicional

## Alcance

- Capturar la garantía retenida del postor líder al cerrar una subasta.
- Liberar las garantías retenidas de los demás postores.
- Liberar todas las garantías retenidas cuando la subasta no tiene ganador.
- Ejecutar la actualización de estados dentro de una transacción PostgreSQL.
- Mantener la integración con pagos reales fuera de este módulo.

## Estados

| Condición | Estado final |
|---|---|
| Postor líder | `CAPTURADA_COBRO` |
| Otros postores | `LIBERADA` |
| Sin postor líder | `LIBERADA` |
