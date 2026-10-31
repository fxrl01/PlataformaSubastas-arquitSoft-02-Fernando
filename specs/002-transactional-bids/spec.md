# Especificación: Pujas transaccionales

> **Feature ID:** `002-transactional-bids`
> **Estado:** En implementación para el MVP tradicional

## Alcance

- Emitir una puja autenticada sobre una subasta `ACTIVA` o `EN_CONTEO`.
- Obtener el postor desde la identidad JWT.
- Validar incremento mínimo y reglas del agregado.
- Actualizar el precio líder y registrar la puja en una única transacción PostgreSQL.
- Rechazar conflictos concurrentes sin dejar cambios parciales.

Redis, WebSockets, escrow y difusión en tiempo real quedan fuera de este módulo.

## Endpoint

```text
POST /api/v1/subastas/{id}/pujas
```

```json
{
  "monto": 2750
}
```

## Criterios de aceptación

```gherkin
Given un usuario POSTOR autenticado y una subasta ACTIVA
When envía un monto que supera el incremento mínimo
Then el backend registra la puja y actualiza el precio en una transacción
And usa el usuario del JWT como postor
```

```gherkin
Given un monto inferior al mínimo o una subasta no activa
When solicita una puja
Then responde con un error de negocio y no modifica PostgreSQL
```

