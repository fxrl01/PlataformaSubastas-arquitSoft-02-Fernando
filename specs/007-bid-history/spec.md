# Especificación: Historial de pujas

> **Feature ID:** `007-bid-history`
> **Estado:** Implementado para el MVP tradicional

## Alcance

- Consultar las pujas aceptadas de una subasta con paginación.
- Mantener el orden cronológico de aceptación.
- Filtrar opcionalmente por postor y monto mínimo.
- Exponer únicamente identificadores internos, montos y marcas de tiempo.
- Devolver `404` cuando la subasta no existe.

## Endpoint

```text
GET /api/v1/subastas/{id}/pujas
```

Parámetros opcionales: `page` (por defecto `1`), `per_page` (por defecto `50`,
máximo `100`), `postor_id` y `monto_minimo`.

La consulta es pública porque el historial es parte de la transparencia de la
subasta. No expone contraseñas, tokens ni información de pago.
