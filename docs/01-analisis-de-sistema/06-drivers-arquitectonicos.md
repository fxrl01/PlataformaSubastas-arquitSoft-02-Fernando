# Drivers arquitectónicos — LiveBid

| ID | Driver arquitectónico | Origen | ¿Por qué influye en la arquitectura? |
|---|---|---|---|
| DA01 | El sistema debe soportar 1,000+ usuarios concurrentes por evento. | AC02 — Escalabilidad | Obliga a una arquitectura orientada a eventos con broker, en vez de llamadas síncronas directas entre el gateway y la lógica de negocio. |
| DA02 | El orden de las pujas debe ser verificable con el timestamp del servidor. | AC05 — Integridad | Obliga a usar una cola dedicada por subasta y a tratar el broker (no la base de datos ni el reloj del cliente) como fuente de verdad del orden. |
| DA03 | La confirmación de puja y la difusión del precio deben ser casi instantáneas. | AC01 — Rendimiento | Obliga a usar WebSocket para la difusión y a separar la validación (rápida, síncrona) de la persistencia (asíncrona). |
| DA04 | El pago debe delegarse a un proveedor externo. | RC08 | Obliga a una capa de adaptador de pagos aislada del dominio, para poder cambiar de proveedor sin tocar la lógica de negocio. |
| DA05 | El video debe delegarse a un CDN externo. | RC09 | Obliga a separar el "canal de precio" (propio, tiempo real) del "canal de video" (de terceros), sincronizados solo a nivel de interfaz de usuario. |
| DA06 | Las pujas deben pasar por un filtro de fraude antes de aceptarse. | AC04 — Seguridad / RF14 | Obliga a insertar el trust-service como un consumidor adicional del broker, en el camino crítico, antes de validar la puja. |
| DA07 | El motor de subasta debe soportar múltiples formatos. | AC06 — Mantenibilidad / RF08 | Obliga a un patrón de diseño tipo *strategy* para las reglas de cada formato de subasta, en vez de lógica condicional acoplada. |
| DA08 | El sistema debe permitir modificar funcionalidades sin afectar innecesariamente otros módulos. | AC06 — Mantenibilidad | Influye en la separación de responsabilidades, modularidad y dependencias internas. Justifica la adopción de Clean/Hexagonal Architecture como enfoque de organización interna. |

## Resumen de drivers y decisiones que responden

| Driver | Problema que plantea | Decisión que responde |
|---|---|---|
| **DA01 — Escalabilidad** | 1,000+ usuarios concurrentes por evento | Arquitectura orientada a eventos con RabbitMQ como broker central |
| **DA02 — Integridad** | Orden de pujas debe ser verificable | Cola dedicada por subasta; broker como fuente de verdad del orden |
| **DA03 — Rendimiento** | Confirmación y difusión casi instantánea | WebSocket para difusión; validación síncrona, persistencia asíncrona |
| **DA04 — Pago externo** | Comunicarse con pasarela de pagos | Integración mediante interfaces y adaptadores desacoplados |
| **DA05 — Video externo** | Distribuir video en vivo a espectadores | CDN externo; separar canal de precio del canal de video |
| **DA06 — Seguridad** | Filtrar pujas fraudulentas | Trust Service como consumidor del broker en el camino crítico |
| **DA07 — Formatos** | Soportar múltiples formatos de subasta | Patrón Strategy para reglas de cada formato |
| **DA08 — Mantenibilidad** | Cambios no deben afectar otros módulos | Clean/Hexagonal Architecture + modularidad interna |

## Los drivers que más condicionan el diseño

De los ocho, **DA01, DA02 y DA06** son los que más se alejan de una arquitectura en capas convencional: exigen pensar el sistema como un flujo de eventos con un punto de control de fraude en el camino. **DA08** complementa exigiendo que la organización interna de cada servicio siga Clean Architecture para facilitar el mantenimiento y la evolución.
