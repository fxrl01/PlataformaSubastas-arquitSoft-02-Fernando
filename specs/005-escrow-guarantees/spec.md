# Especificación: Garantías de participación y escrow básico

> **Feature ID:** `005-escrow-guarantees`
> **Estado:** En implementación para el MVP tradicional

## Alcance

- Calcular una garantía base del 10% del precio base.
- Registrar una retención local en estado `RETENIDA`.
- Evitar dos garantías activas para el mismo postor y subasta.
- Exigir una garantía activa antes de aceptar una puja.
- Exponer la autorización y el monto retenido, nunca credenciales de pago.

La integración con una pasarela bancaria real, captura, liberación y evaluación
externa de IA quedan fuera de este módulo.

## Endpoint

```text
POST /api/v1/subastas/{id}/garantias
```

```json
{
  "monto": 250
}
```

El postor se obtiene del JWT y el monto no puede ser inferior al 10% del precio
base.
