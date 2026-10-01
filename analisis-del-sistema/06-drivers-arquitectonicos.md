# Drivers arquitectónicos — Rematix

| ID | Driver arquitectónico | Origen | ¿Por qué influye en la arquitectura? |
|---|---|---|---|
| DA01 | El sistema debe soportar 1,000+ usuarios concurrentes por evento. | AC02 — Escalabilidad | Obliga a una arquitectura orientada a eventos con broker, en vez de llamadas síncronas directas entre el gateway y la lógica de negocio. |
| DA02 | El orden de las pujas debe ser verificable con el timestamp del servidor. | AC05 — Integridad | Obliga a usar una cola dedicada por subasta y a tratar el broker (no la base de datos ni el reloj del cliente) como fuente de verdad del orden. |
| DA03 | La confirmación de puja y la difusión del precio deben ser casi instantáneas. | AC01 — Rendimiento | Obliga a usar WebSocket para la difusión y a separar la validación (rápida, síncrona) de la persistencia (asíncrona). |
| DA04 | El pago debe delegarse a un proveedor externo. | RC08 | Obliga a una capa de adaptador de pagos aislada del dominio, para poder cambiar de proveedor sin tocar la lógica de negocio. |
| DA05 | El video debe delegarse a un CDN externo. | RC09 | Obliga a separar el "canal de precio" (propio, tiempo real) del "canal de video" (de terceros), sincronizados solo a nivel de interfaz de usuario. |
| DA06 | Las pujas deben pasar por un filtro de fraude antes de aceptarse. | AC04 — Seguridad / RF14 | Obliga a insertar el trust-service como un consumidor adicional del broker, en el camino crítico, antes de validar la puja. |
| DA07 | El motor de subasta debe soportar múltiples formatos. | AC06 — Mantenibilidad / RF08 | Obliga a un patrón de diseño tipo *strategy* para las reglas de cada formato de subasta, en vez de lógica condicional acoplada. |

## Los tres drivers que más van a condicionar el diseño

De los siete, **DA01, DA02 y DA06** son los que más se alejan de una arquitectura en capas convencional (como la del ejemplo del marketplace): exigen pensar el sistema como un flujo de eventos con un punto de control de fraude en el camino, no como una simple cadena Presentación → Negocio → Datos. Esto se va a reflejar directamente en la Etapa 2 (diseño arquitectónico), que probablemente necesite más de tres capas para representarse con honestidad.
