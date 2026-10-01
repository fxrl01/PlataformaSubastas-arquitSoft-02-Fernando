# Actores del sistema — Rematix (nombre de trabajo)

## Actores humanos

| Actor | ¿Qué necesita realizar? |
|---|---|
| **Pujador / Espectador** | Explorar subastas activas y programadas, unirse a la transmisión en vivo, enviar pujas en tiempo real, ver el precio actual, consultar su historial de pujas y pagos, recibir el resultado del lote. |
| **Rematador / Organizador del evento** | Crear una subasta y elegir su formato (ascendente, descendente, sellada, benéfica), registrar los lotes, iniciar y cerrar la transmisión en vivo, cerrar cada lote manualmente o por temporizador. |
| **Vendedor / Consignante** | Registrar el ítem o lote a subastar (descripción, fotos, precio de reserva) y recibir el pago una vez cerrada la venta. Puede coincidir con el rematador en eventos pequeños, o ser un tercero en un modelo de consignación. |
| **Administrador de la plataforma** | Gestionar cuentas de usuarios y rematadores, revisar alertas de fraude, resolver disputas sobre el resultado de una puja, suspender cuentas, generar reportes. |

## Sistemas externos

| Sistema externo | ¿Qué necesita realizar? |
|---|---|
| **Pasarela de pagos** (Stripe / PayPal) | Procesar el cobro al pujador ganador y la liquidación al vendedor. |
| **Proveedor de streaming / CDN de video** (Cloudflare Stream, Mux o AWS IVS) | Distribuir la señal de video en vivo a todos los espectadores conectados. |
| **Servicio de notificaciones** (email / push) | Notificar resultados de puja, recordatorios de evento y confirmaciones de pago. |
| **Servicio de verificación de identidad (KYC)** | Validar la identidad de un pujador antes de habilitarlo a pujar por encima de un monto umbral. |

## Nota de diseño

A diferencia de un marketplace tradicional (compra asíncrona, un solo precio fijo), aquí aparecen dos actores que no existen en ese modelo — el **Rematador** y el **Vendedor** como roles potencialmente distintos — y el sistema debe funcionar en una ventana de tiempo real, no asíncrona. Esta diferencia ya condiciona el análisis desde las historias de usuario.
