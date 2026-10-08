# Restricciones — LiveBid

| ID | Restricción | Descripción |
|---|---|---|
| RC01 | Control de versiones | El código debe gestionarse con Git y GitHub; los commits deben seguir Conventional Commits y ser trazables a la tarea de spec-kit que los originó. |
| RC02 | Lenguajes de backend | Los servicios de dominio (gateway, auction-engine, broadcast en tiempo real) deben construirse en Go; el servicio de detección de fraude debe construirse en Python. |
| RC03 | Frontend | La interfaz de espectador y rematador debe construirse en TypeScript con React. |
| RC04 | Comunicación | La comunicación entre frontend y servicios debe realizarse mediante API REST para operaciones puntuales y WebSocket para actualizaciones en tiempo real. |
| RC05 | Persistencia | La información transaccional (usuarios, subastas, pujas, pagos) debe almacenarse en PostgreSQL. |
| RC06 | Mensajería | El desacople entre el gateway y el motor de subasta debe implementarse con un message broker — RabbitMQ en la primera versión. |
| RC07 | Contenedores | Todos los servicios deben poder levantarse con Docker Compose, garantizando portabilidad entre Linux y Windows. |
| RC08 | Pasarela de pago | El procesamiento de pagos debe delegarse a un proveedor externo certificado (Stripe o PayPal); el sistema no debe almacenar datos de tarjetas. |
| RC09 | Streaming de video | La distribución del video en vivo debe delegarse a un proveedor de CDN especializado; no se auto-aloja en producción. |
| RC10 | Metodología de desarrollo | El proyecto debe seguir Spec-Driven Development (constitution → specify → plan → tasks → implement), con cobertura obligatoria de tests unitarios, de integración y de carga antes de cualquier despliegue. |
| RC11 | Estilo arquitectónico | Cada servicio debe organizarse con arquitectura hexagonal/limpia (dominio, casos de uso, adaptadores de infraestructura). |
