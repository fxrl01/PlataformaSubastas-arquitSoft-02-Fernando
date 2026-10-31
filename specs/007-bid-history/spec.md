# Especificación: Historial de pujas

> **Feature ID:** `007-bid-history`
> **Estado:** Implementado para el MVP tradicional

## Alcance

- Consultar las pujas aceptadas de una subasta.
- Mantener el orden cronológico de aceptación.
- Exponer únicamente identificadores internos, montos y marcas de tiempo.
- Devolver `404` cuando la subasta no existe.

## Endpoint

```text
GET /api/v1/subastas/{id}/pujas
```

La consulta es pública porque el historial es parte de la transparencia de la
subasta. No expone contraseñas, tokens ni información de pago.
