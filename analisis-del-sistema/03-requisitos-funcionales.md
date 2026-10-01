# Requisitos funcionales — Rematix

| ID | Requisito funcional |
|---|---|
| RF01 | El sistema debe permitir listar y filtrar subastas activas y programadas. |
| RF02 | El sistema debe transmitir el video en vivo de la subasta a todos los espectadores conectados. |
| RF03 | El sistema debe permitir a un pujador enviar una puja mientras el evento está en vivo. |
| RF04 | El sistema debe validar cada puja contra el precio actual y el incremento mínimo del lote antes de aceptarla. |
| RF05 | El sistema debe difundir el nuevo precio a todos los espectadores en tiempo real tras cada puja válida. |
| RF06 | El sistema debe notificar al pujador ganador y a los perdedores al cerrarse un lote. |
| RF07 | El sistema debe registrar el historial de pujas y pagos de cada usuario, consultable en cualquier momento. |
| RF08 | El sistema debe permitir crear una subasta seleccionando su formato (ascendente, descendente, sellada, benéfica). |
| RF09 | El sistema debe permitir registrar uno o más lotes con precio base, incremento mínimo y orden de presentación. |
| RF10 | El sistema debe permitir al rematador iniciar y finalizar la transmisión en vivo del evento. |
| RF11 | El sistema debe permitir cerrar un lote manualmente o automáticamente al vencer un temporizador configurado. |
| RF12 | El sistema debe permitir a un vendedor registrar un ítem con descripción, fotos y precio de reserva. |
| RF13 | El sistema debe procesar el pago del pujador ganador y la liquidación al vendedor a través de una pasarela externa. |
| RF14 | El sistema debe calcular un puntaje de confianza (trust score) por pujador antes de habilitarlo a pujar. |
| RF15 | El sistema debe generar alertas al administrador cuando el puntaje de confianza de una puja sea sospechoso. |
| RF16 | El sistema debe permitir al administrador suspender cuentas y consultar el registro íntegro y ordenado de pujas ante una disputa. |
| RF17 | El sistema debe permitir la verificación de identidad de un pujador antes de habilitar pujas por encima de un monto umbral. |

## Relación entre historias de usuario y requisitos funcionales

| Historia de usuario | Requisitos funcionales relacionados |
|---|---|
| HU01 Explorar subastas | RF01 |
| HU02 Ver transmisión en vivo | RF02 |
| HU03 Enviar puja | RF03, RF04 |
| HU04 Ver precio en tiempo real | RF05 |
| HU05 Recibir resultado del lote | RF06 |
| HU06 Consultar historial | RF07 |
| HU07 Verificar identidad | RF17 |
| HU08 Crear subasta con formato | RF08 |
| HU09 Registrar lotes | RF09 |
| HU10 Iniciar/cerrar transmisión | RF10 |
| HU11 Cerrar lote manual/temporizador | RF11 |
| HU12 Registrar ítem | RF12 |
| HU13 Recibir pago | RF13 |
| HU14 Ver alertas de fraude | RF14, RF15 |
| HU15 Suspender cuenta | RF16 |
| HU16 Consultar registro ante disputa | RF16 |
