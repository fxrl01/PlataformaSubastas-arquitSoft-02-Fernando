# Nivel 2 · Diagrama de contenedores — LiveBid

> **Proyecto:** LiveBid — Plataforma de Subastas en Tiempo Real  
> **Modelo:** C4 · Nivel 2 de 4 · **Versión:** 1.0  
> **Estándar:** Software Design Description (SDD)

---

## 1. Objetivo

Abrir la caja «LiveBid» del nivel 1 y descomponerla en sus **contenedores ejecutables**: aplicaciones web, APIs, motores en tiempo real y almacenes de datos que operan de forma independiente, detallando sus tecnologías y mecanismos de comunicación.

| Aspecto | Descripción |
|---|---|
| **Pregunta que responde** | ¿Qué piezas de software desplegables componen LiveBid y cómo se comunican entre sí? |
| **Audiencia** | Arquitectos de software, desarrolladores fullstack, ingenieros DevOps/Cloud |
| **Qué es un contenedor en C4** | Una unidad ejecutable o de almacenamiento independiente (una SPA, un servicio backend, una base de datos) |
| **Nivel anterior** | `Nivel1-DiagramadeContextodelSistema.md` |
| **Siguiente nivel** | `Nivel3-Diagrama-de-Componentes.md` |

---

## 2. Diagrama de Contenedores (C4 Nivel 2)

```mermaid
graph TD
    classDef person fill:#08427B,stroke:#073B6F,color:#ffffff,stroke-width:2px;
    classDef container fill:#1168BD,stroke:#0B4884,color:#ffffff,stroke-width:2px;
    classDef db fill:#1C5A96,stroke:#0B4884,color:#ffffff,stroke-width:2px;
    classDef external fill:#999999,stroke:#666666,color:#ffffff,stroke-width:2px;

    Pujador["👤 Postor / Pujador"]:::person
    Martillero["👤 Martillero / Subastador"]:::person

    subgraph LiveBidSystem["Boundary: Plataforma LiveBid"]
        SPA["🌐 Aplicación Web SPA<br><i>[Contenedor: React / TypeScript / Vite]</i><br>Interfaz reactiva para sala de remate, emisión de pujas y visualización de video"]:::container
        Gateway["🛡️ API Gateway & Reverse Proxy<br><i>[Contenedor: NGINX / Envoy]</i><br>Terminación SSL, rate limiting, balanceo de carga y ruteo HTTP / WSS"]:::container
        BidEngine["⚡ Realtime Bid Engine<br><i>[Contenedor: Node.js / WebSockets]</i><br>Maneja conexiones de sockets concurrentes, cronómetro del martillo y broadcast de eventos"]:::container
        CoreAPI["🏛️ Core API Backend<br><i>[Contenedor: Node.js + Express / TypeScript]</i><br>Monolito modular con Clean Architecture: usuarios, catálogo, depósitos en garantía y liquidación"]:::container
        Redis["🧠 Cache & Pub/Sub Store<br><i>[Contenedor: Redis In-Memory]</i><br>Gestión de estado en caliente, locks atómicos (Anti-Race Condition) y canales Pub/Sub"]:::db
        Postgres["🗄️ Base de Datos Principal<br><i>[Contenedor: PostgreSQL 16]</i><br>Persistencia relacional ACID para usuarios, bienes, pujas históricas y registros contables"]:::db
        Broker["📨 Message Broker<br><i>[Contenedor: RabbitMQ]</i><br>Cola de mensajería asíncrona para eventos de auditoría, facturación y notificaciones"]:::container
    end

    Pasarela["🏛️ Pasarela Pagos (Niubiz)"]:::external
    NotifService["📲 Servicio Notificaciones"]:::external

    Pujador -->|Usa la plataforma en navegador [HTTPS]| SPA
    Martillero -->|Controla el remate [HTTPS]| SPA

    SPA -->|Peticiones REST [JSON/HTTPS]| Gateway
    SPA -->|Canal bidireccional [WSS / Socket.io]| Gateway

    Gateway -->|Ruta tráfico REST| CoreAPI
    Gateway -->|Ruta conexiones socket| BidEngine

    BidEngine -->|Locks atómicos y broadcast Pub/Sub| Redis
    BidEngine -->|Publica evento 'PujaAceptada'| Broker

    CoreAPI -->|Lectura / Escritura relacional| Postgres
    CoreAPI -->|Cacheo y verificación de garantías| Redis
    CoreAPI -->|Publica eventos de dominio| Broker
    CoreAPI -->|Retiene depósitos de garantía| Pasarela

    Broker -->|Consume eventos para envío diferido| CoreAPI
    CoreAPI -->|Envía alertas SMS/Email| NotifService
```

---

## 3. Catálogo de Contenedores

| Contenedor | Tecnología | Responsabilidad Principal | Persistencia / Estado |
|---|---|---|---|
| **Aplicación Web SPA** | React 18, TypeScript, Vite, Tailwind | Renderizado de sala de subasta interactiva, gráficos de puja en vivo, reproductor de streaming y validación de entrada. | Estado volátil en memoria del navegador (React State / Zustand). |
| **API Gateway & Proxy** | NGINX / Envoy | Punto único de entrada de red. Control de tráfico, protección DDoS, terminación TLS y distribución balanceada. | Stateless (sin estado). |
| **Realtime Bid Engine** | Node.js + Socket.io / ws | Mantiene abiertas las sesiones de socket de miles de usuarios, sincroniza el cronómetro del martillo y difunde las nuevas ofertas en $< 30$ ms. | Sin persistencia de disco; sincroniza estado inmediato con Redis. |
| **Core API Backend** | Node.js, Express, TypeScript | Expone la API REST modular. Aplica reglas de negocio mediante Clean Architecture: usuarios, lotes, peritajes, custodia y cobros. | Accede a PostgreSQL y Redis. |
| **In-Memory Store (Redis)** | Redis 7.2 | Proporciona locks distribuidos atómicos (`SETNX`) para garantizar cero colisiones de pujas (Anti-Race Condition) y canal Pub/Sub multihilo. | Memoria RAM ultrarrápida ($< 2$ ms de latencia) con persistencia periódica RDB/AOF. |
| **Base de Datos (PostgreSQL)** | PostgreSQL 16 | Almacén de verdad definitivo con transacciones ACID, constraints de integridad y tablas particionadas para historial de pujas. | Disco SSD persistente con replicación y backups automatizados. |
| **Message Broker (RabbitMQ)** | RabbitMQ 3.13 | Desacopla tareas asíncronas pesadas (emisión de recibos, registro forense de auditoría, envío de emails masivos) para no degradar el flujo en vivo. | Colas duraderas persistidas en disco. |

---

## 4. Flujo de Datos y Protocolos de Comunicación

1. **Flujo en Vivo (Hot Path):**
   - El cliente emite una oferta por `WSS` al **Bid Engine** a través del Gateway.
   - El **Bid Engine** valida la concurrencia adquiriendo un lock en **Redis** en $\approx 1.5$ ms.
   - Si es válida, emite el evento al canal Pub/Sub de **Redis** y a la cola de **RabbitMQ**.
   - Todas las instancias del **Bid Engine** reciben el mensaje y actualizan a los clientes por WebSocket en $< 50$ ms.
2. **Flujo Transaccional Frío (Cold Path):**
   - Un worker en segundo plano consume el mensaje de **RabbitMQ** y consolida la puja en **PostgreSQL** sin estresar el hilo principal de WebSockets.

---

## 5. Decisiones de Diseño en Contenedores

- **Separación de Bid Engine y Core API:** Separar el motor de WebSockets del API REST tradicional evita que consultas pesadas (ej. reportes administrativos o catálogos grandes) bloqueen el Event Loop del motor de tiempo real.
- **Doble Persistencia (Redis + Postgres):** Redis garantiza la velocidad en el milisegundo de la subasta; Postgres garantiza la seguridad legal y contable a largo plazo.

