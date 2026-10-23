# Constitución de Ingeniería — LiveBid

> **Proyecto:** LiveBid — Plataforma de Subastas en Tiempo Real  
> **Estándar:** GitHub Spec Kit · Spec-Driven Development (SDD)  
> **Versión:** 1.0.0 · **Estado:** VIGENTE E INMUTABLE

---

## 1. Declaración de Principios Innegociables

Esta Constitución establece las reglas arquitectónicas y de ingeniería fundamentales que rigen el desarrollo de **LiveBid**. Todo agente de IA, desarrollador o arquitecto debe adherirse a estos principios sin excepción. Ninguna tarea de código será considerada aprobada si viola alguno de los artículos constitucionales.

---

### Artículo I: Clean Architecture e Inversión de Dependencias
1. **Aislamiento del Dominio:** El núcleo de negocio (entidades, agregados y reglas de martillazo) no debe importar ni conocer frameworks web (Express, NestJS), librerías de red (Socket.io), ni clientes de bases de datos (TypeORM, Prisma, pg, ioredis).
2. **Regla de Dependencia Unidireccional:** El flujo de dependencias apunta siempre hacia adentro:  
   `Presentación` $\rightarrow$ `Aplicación` $\rightarrow$ `Dominio` $\leftarrow$ `Infraestructura`.
3. **Puertos y Adaptadores:** Toda comunicación con el exterior (bases de datos, pasarelas de pago, brokers y sockets) se define mediante interfaces (`puertos`) en el dominio y se implementa en la capa de `infraestructura`.

---

### Artículo II: Principio Zero Trust y Custodia Previa (Escrow)
1. **Cero Confianza Implícita:** Ninguna solicitud de puja es confiable por el simple hecho de provenir de una conexión WebSocket abierta. Cada mensaje debe portar un token de sesión criptográfico verificado.
2. **Garantía Financiera Obligatoria:** Ningún postor puede emitir ofertas en una subasta activa sin contar con un depósito de garantía previamente retenido y congelado (*Escrow*) por la pasarela de pagos autorizada.
3. **Rechazo en Capa de Borde:** Las ofertas de usuarios sin saldo en custodia o sin validación de identidad (KYC) deben ser rechazadas en la capa de presentación/aplicación antes de tocar el motor transaccional.

---

### Artículo III: Concurrencia Atómica y Fuente de Verdad Legal
1. **El Orden es el Producto:** En subastas en vivo, la secuencia de llegada de las ofertas determina el derecho de compra. Dos pujas emitidas en el mismo milisegundo deben resolverse de forma estrictamente secuencial y atómica.
2. **Locks Distribuidos en Memoria:** Toda evaluación de puja debe adquirir un Mutex atómico en Redis (`SETNX` o script Lua) para evitar condiciones de carrera (*Anti-Race Condition*).
3. **Timestamp del Servidor Inapelable:** La única fuente legal de tiempo válida es el reloj sincronizado NTP del servidor de LiveBid. Se rechaza cualquier reclamo sustentado en la hora del dispositivo cliente.

---

### Artículo IV: Estrategia de Almacenamiento Dual (Dual-Store)
1. **Almacén Caliente (In-Memory - Redis):** Gestiona el estado en vivo de la sala, cronómetros regresivos, locks de concurrencia y canales Pub/Sub. Su latencia de respuesta debe ser $< 2$ ms.
2. **Almacén Frío (Relacional ACID - PostgreSQL):** Base de datos principal de persistencia definitiva. Almacena usuarios, catálogos, peritajes, consolidación contable y el log inmutable de auditoría.
3. **Persistencia Desacoplada:** La escritura en PostgreSQL durante el remate se realiza de forma asíncrona mediante colas en RabbitMQ para no bloquear el hilo de emisión en tiempo real de WebSockets.

---

### Artículo V: Presupuestos de Calidad y Rendimiento (SLAs)

| Métrica de Rendimiento | Umbral Máximo Permitido |
|---|---|
| **Latencia de confirmación de puja (P95)** | $\le 200$ ms |
| **Difusión de nuevo precio líder (Broadcast)** | $\le 300$ ms a 1,000 postores concurrentes |
| **Reconexión y resincronización de cliente** | $\le 1.5$ segundos |
| **Disponibilidad de sala en vivo (Uptime)** | $\ge 99.9\%$ durante eventos programados |
| **Tolerancia a condiciones de carrera** | **0%** (Cero colisiones permitidas) |

---

### Artículo VI: Ciclo de Vida SDD (Spec-Driven Development)
1. **No Vibe-Coding:** Ningún archivo de código se creará o modificará sin una especificación previa aprobada en la carpeta `specs/`.
2. **Ciclo en Tres Fases:**
   - **`spec.md` (Qué y Por qué):** Historias de usuario, reglas de negocio y criterios de aceptación en formato Given-When-Then (Gherkin).
   - **`plan.md` (Cómo técnico):** Contratos de interfaz, esquemas de datos, diagramas de secuencia y mapeo C4.
   - **`tasks.md` (Plan de acción):** Tareas atómicas, verificables y ordenadas por dependencias.
3. **Verificación Automatizada:** Toda feature implementada debe incluir pruebas unitarias o de integración que certifiquen el 100% de los criterios de aceptación estipulados en su `spec.md`.

