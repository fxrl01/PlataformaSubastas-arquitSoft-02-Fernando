# Estilo Arquitectónico — Rematix

El estilo arquitectónico define la estructura global del sistema y cómo se organizan sus componentes a alto nivel.

## Estilo Seleccionado: Capas + Orientación a Eventos

Debido a la naturaleza de tiempo real y alta concurrencia de la plataforma de subastas, un estilo puro de N-capas (donde las llamadas son síncronas de arriba hacia abajo) no es suficiente. Por lo tanto, hemos seleccionado un estilo híbrido:

1. **Arquitectura en Capas:** Para organizar las responsabilidades generales del sistema (Presentación, Negocio, Datos).
2. **Arquitectura Orientada a Eventos (EDA):** Para gestionar el flujo crítico dentro de la capa de negocio (procesamiento de pujas, validación, evaluación de fraude y difusión).

## Descripción de las Capas

| Capa | Responsabilidad | Componentes |
|---|---|---|
| **Presentación** | Manejar la interacción con el usuario (pujador, rematador, etc.), mostrar la interfaz y gestionar las conexiones en tiempo real. | App Web (React), API Gateway (REST + WebSocket), Broadcast Service (difusión WS). |
| **Negocio (Orientada a Eventos)** | Contener la lógica central de las subastas, garantizar el orden, validar pujas y evaluar riesgos. La comunicación aquí es asíncrona mediante eventos. | RabbitMQ (Message Broker), Auction Engine, Trust Service. |
| **Datos** | Persistir la información del sistema y proveer acceso rápido a datos en caliente. | PostgreSQL (datos transaccionales), Redis (precio actual en memoria). |
| **Sistemas Externos** | Servicios de terceros que proveen funcionalidades especializadas que no desarrollamos internamente. | Pasarela de Pago, CDN de Streaming, Servicio de Notificaciones. |

## ¿Por qué este estilo?

1. **Desacoplamiento:** La orientación a eventos permite que el `Gateway` (Presentación) simplemente publique una intención de puja y responda rápido al usuario, mientras que el `Auction Engine` (Negocio) la procesa a su propio ritmo.
2. **Escalabilidad (DA01):** El `Message Broker` actúa como un amortiguador (buffer) durante picos de pujas, evitando que la base de datos o el motor de subasta colapsen.
3. **Integridad (DA02):** La cola de mensajes garantiza un orden estricto de llegada, crucial para determinar quién gana en una subasta.
4. **Seguridad no bloqueante (DA06):** El `Trust Service` puede evaluar el riesgo de fraude consumiendo eventos en paralelo, sin añadir latencia excesiva al camino crítico de aceptación de pujas.

## Diagrama del Estilo Arquitectónico

*(Ver `arquitectura/arquitectura-inicial.md` para el diagrama Mermaid y el enlace al diagrama interactivo Archify).*
