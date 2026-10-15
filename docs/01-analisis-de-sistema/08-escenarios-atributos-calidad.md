# Escenarios de atributos de calidad — LiveBid

> **Proyecto:** LiveBid — Plataforma de Subastas en Tiempo Real  
> **Estándar:** Software Engineering Institute (SEI) · Escenarios de 6 partes  
> **Versión:** 1.0 · **Estado:** Base para Diseño Arquitectónico (SDD)

---

## Introducción

Un atributo de calidad no es solo una declaración genérica como *"el sistema debe ser rápido"* o *"debe ser seguro"*. Para ser verificable y guiar el diseño arquitectónico, cada atributo debe formularse mediante un **escenario concreto y medible de seis elementos**:

1. **Fuente del estímulo:** Quién o qué genera la solicitud o condición.
2. **Estímulo:** La acción o evento que llega al sistema.
3. **Artefacto:** El componente o subsistema específico que recibe y atiende el estímulo.
4. **Entorno:** Las condiciones operativas del sistema en ese momento (ej. carga pico, falla de red, operación normal).
5. **Respuesta:** Lo que el sistema debe hacer ante el estímulo.
6. **Medida de la respuesta:** El umbral cuantitativo que determina el éxito o fracaso del cumplimiento.

---

## EQ-01. Rendimiento: Latencia de procesamiento y difusión de pujas en tiempo real

| Elemento | Especificación en LiveBid |
|---|---|
| **Atributo de calidad** | Rendimiento (Performance) |
| **Fuente** | Postores concurrentes en una sala de subasta en vivo |
| **Estímulo** | Envío de ráfagas de ofertas de puja durante los últimos 10 segundos del conteo ("martillazo") |
| **Artefacto** | Realtime Bid Engine (WebSocket Server) + Redis In-Memory State + Broker RabbitMQ |
| **Entorno** | Evento en vivo con 1,000 postores conectados simultáneamente en la misma sala |
| **Respuesta esperada** | El motor valida la oferta, registra el nuevo precio líder y emite el broadcast a todos los clientes conectados sin degradar la conexión WebSocket |
| **Medida** | **Percentil 95 (P95) de confirmación al pujador < 200 ms**; difusión a toda la sala de espectadores **< 300 ms**; 0% de pérdidas de conexión por saturación |
| **Verificación** | Pruebas de estrés con herramientas de carga de WebSockets (ej. k6 / Artillery) simulando 1,000 conexiones concurrentes y 200 pujas/segundo |
| **Estado** | Aprobado para diseño de Nivel 2 y Nivel 3 C4 |

---

## EQ-02. Integridad y Consistencia: Resolución atómica de pujas concurrentes (Anti-Race Condition)

| Elemento | Especificación en LiveBid |
|---|---|
| **Atributo de calidad** | Integridad y Orden Verificable (Consistency & Integrity) |
| **Fuente** | Múltiples postores enviando ofertas simultáneas |
| **Estímulo** | Dos o más clientes emiten una puja por el mismo monto en el mismo milisegundo exacto |
| **Artefacto** | Secuenciador de Pujas en Redis (Atomic Lock / Mutex) + Cola transaccional FIFO en RabbitMQ |
| **Entorno** | Conteo regresivo final activo con alta contención de recursos |
| **Respuesta esperada** | El sistema procesa estrictamente la primera puja según el timestamp oficial del servidor, rechaza la segunda puja informando "oferta superada" y mantiene la coherencia del historial |
| **Medida** | **0 colisiones o duplicidades de adjudicación**. 100% de las pujas aceptadas tienen un ID correlativo secuencial e inmutable con hash de auditoría |
| **Verificación** | Pruebas de concurrencia automatizadas inyectando peticiones idénticas concurrentes en hilos paralelos y comprobando el log de transacciones |
| **Estado** | Aprobado como driver arquitectónico crítico (ADR-004) |

---

## EQ-03. Disponibilidad: Resiliencia de la sala en vivo y tolerancia a desconexión

| Elemento | Especificación en LiveBid |
|---|---|
| **Atributo de calidad** | Disponibilidad y Resiliencia (Availability & Fault Tolerance) |
| **Fuente** | Conexión inestable de red del cliente o reinicio de nodo en el cluster |
| **Estímulo** | Pérdida de conexión WebSocket en medio de una subasta activa |
| **Artefacto** | Cliente SPA (Reconnection Engine) + API Gateway con Balanceador de Carga + Heartbeat |
| **Entorno** | Subasta en curso con dinero en custodia y tiempo límite corriendo |
| **Respuesta esperada** | El cliente reconecta automáticamente con el socket más próximo y sincroniza en un solo payload el estado del lote (precio actual, tiempo restante, último postor) |
| **Medida** | **Reconexión y resincronización de estado en < 1.5 segundos** sin recarga completa de la página; disponibilidad mensual de la infraestructura en vivo del **99.9%** durante eventos programados |
| **Verificación** | Inyección de fallas de red (Chaos Testing / desconexión forzada de sockets) verificando tiempo de recuperación y consistencia del estado en pantalla |
| **Estado** | Aprobado para implementación en capa de cliente y Gateway |

---

## EQ-04. Seguridad: Zero Trust, autenticación estricta y fondos en custodia

| Elemento | Especificación en LiveBid |
|---|---|
| **Atributo de calidad** | Seguridad y Confianza (Security & Zero Trust) |
| **Fuente** | Usuario no verificado, token expirado o intento de inyección maliciosa en WebSocket |
| **Estímulo** | Intento de enviar una puja sin contar con depósito de garantía (escrow) activo o con firma digital manipulada |
| **Artefacto** | Middleware de Autenticación JWT en WebSocket + Servicio de Validación de Garantías (Escrow Service) |
| **Entorno** | Sala de subastas abierta al público |
| **Respuesta esperada** | El sistema rechaza la puja en la capa de borde antes de llegar a la cola transaccional, envía mensaje de error privado al emisor y registra evento de seguridad sospechoso |
| **Medida** | **Rechazo del 100% de pujas sin garantía validada**; 0 exposición de tokens internos o datos financieros en payloads públicos de la sala |
| **Verificación** | Pruebas de penetración de WebSockets, simulación de firmas inválidas y peticiones con balance insuficiente |
| **Estado** | Aprobado según lineamiento Zero Trust del proyecto |

---

## EQ-05. Mantenibilidad: Desacoplamiento modular y extensibilidad de reglas

| Elemento | Especificación en LiveBid |
|---|---|
| **Atributo de calidad** | Mantenibilidad (Maintainability & Extensibility) |
| **Fuente** | Equipo de ingeniería / Nuevos requisitos de negocio |
| **Estímulo** | Se solicita incorporar una nueva modalidad de subasta (ej. Subasta Holandesa descendente) o cambiar el proveedor de pasarela de pagos para custodia |
| **Artefacto** | Módulo de Dominio de Subastas (Patrón State / Strategy) y Puertos de Infraestructura (Patrón Adapter) |
| **Entorno** | Desarrollo y despliegue continuo (CI/CD) |
| **Respuesta esperada** | El cambio se realiza creando una nueva estrategia o adaptador sin modificar el núcleo de procesamiento de eventos en tiempo real ni la estructura de datos base |
| **Medida** | Modificación localizada exclusivamente en el módulo respectivo. **100% de pruebas de regresión aprobadas**. El tiempo de integración de un nuevo adaptador de pago no supera 1 sprint (3 días) |
| **Verificación** | Revisión de análisis estático de código, métricas de acoplamiento (Aferente/Eferente) e inspección de pull requests |
| **Estado** | Aprobado para guiar la adopción de Clean Architecture |

---

## Resumen de escenarios de calidad

| Código | Atributo de calidad | Qué se busca comprobar en LiveBid | Métrica de Aceptación |
|---|---|---|---|
| **EQ-01** | Rendimiento | Que la puja se procese y difunda instantáneamente | P95 < 200 ms confirmación; < 300 ms difusión global |
| **EQ-02** | Integridad y Orden | Que no haya colisiones ni doble ganador ante pujas simultáneas | 0 carreras críticas; orden FIFO verificado con timestamp servidor |
| **EQ-03** | Disponibilidad | Que un corte de red no arruine la experiencia de subasta | Reconexión y resincronización de estado en < 1.5 s; 99.9% uptime |
| **EQ-04** | Seguridad | Que solo pujen usuarios verificados con fondos en garantía | 100% de ofertas sin escrow rechazadas en borde |
| **EQ-05** | Mantenibilidad | Que se puedan agregar reglas y servicios sin romper el motor | Aislamiento por Clean Architecture; 0 regresiones |

