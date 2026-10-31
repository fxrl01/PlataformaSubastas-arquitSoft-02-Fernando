# Especificación: Catálogo y ciclo básico de subastas

> **Feature ID:** `001-auction-catalog`
> **Estado:** En implementación para el MVP tradicional

## Propósito

Permitir que usuarios autorizados creen y administren subastas, mientras los
clientes pueden consultar el catálogo mediante la API HTTP.

## Alcance

- Crear una subasta como `VENDEDOR`.
- Consultar el catálogo y una subasta por identificador.
- Activar una subasta como `MARTILLERO` o `ADMIN`.
- Cerrar una subasta como `MARTILLERO` o `ADMIN`.
- Persistir las subastas en PostgreSQL.

Las pujas, locks Redis, WebSockets, escrow y anti-sniping quedan fuera de este
módulo.

## Criterios de aceptación

### Creación

```gherkin
Given un usuario autenticado con rol VENDEDOR
When solicita crear una subasta con fechas válidas y precio positivo
Then el sistema la guarda en estado PENDIENTE
And no acepta el vendedor desde el payload como identidad
```

### Consulta

```gherkin
Given una subasta persistida
When un cliente consulta el catálogo o su identificador
Then recibe sus datos públicos sin contraseñas ni secretos
```

### Ciclo

```gherkin
Given una subasta PENDIENTE
When un MARTILLERO o ADMIN la activa
Then cambia a ACTIVA
```

