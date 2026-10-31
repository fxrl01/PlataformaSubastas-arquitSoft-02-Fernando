# Especificación: Cierre automático de subastas vencidas

> **Feature ID:** `009-automatic-auction-closing`
> **Estado:** Implementado para el MVP tradicional

## Alcance

- Revisar periódicamente las subastas activas y en conteo.
- Cerrar las que superaron su fecha de cierre.
- Ejecutar la liquidación de garantías y publicar el evento de cierre existente.
- Detener el proceso durante el apagado ordenado del servidor.

El endpoint manual de cierre continúa disponible para operadores autorizados.
