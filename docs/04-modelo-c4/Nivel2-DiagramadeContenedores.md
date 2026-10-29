# Nivel 2 · Diagrama de contenedores — LiveBid

> **Proyecto:** LiveBid — Plataforma de Subastas en Tiempo Real  
> **Modelo:** C4 · Nivel 2 de 4 · **Versión:** 2.0 (Startup Escalable & AWS Free Tier)  
> **Estándar:** Software Design Description (SDD)  
> **Stack Oficial:** Go (Golang) + Next.js + PostgreSQL + Redis + Gemini AI + AWS Free Tier

---

## 1. Objetivo

Abrir la caja «LiveBid» del nivel 1 y descomponerla en sus **contenedores ejecutables**: aplicaciones web, APIs de alto rendimiento, motores en tiempo real y almacenes de datos desacoplados que operan de forma independiente en la nube de **AWS (Free Tier Ready)**.

| Aspecto | Descripción |
|---|---|
| **Pregunta que responde** | ¿Qué piezas de software desplegables componen LiveBid y cómo se comunican entre sí? |
| **Audiencia** | Arquitectos de software, líderes técnicos, desarrolladores backend/frontend, DevOps/Cloud engineers |
| **Qué es un contenedor en C4** | Una unidad ejecutable o de almacenamiento independiente (un frontend web, un servicio backend compilado, una base de datos) |
| **Nivel anterior** | `Nivel1-DiagramadeContextodelSistema.md` |
| **Siguiente nivel** | `Nivel3-Diagrama-de-Componentes.md` |

---

## 2. Diagrama de Contenedores (C4 Nivel 2 — Cloud & Go Core)

```mermaid
graph TD
    classDef person fill:#08427B,stroke:#073B6F,color:#ffffff,stroke-width:2px;
    classDef frontend fill:#1E293B,stroke:#0F172A,color:#ffffff,stroke-width:2px;
    classDef backend fill:#00ADD8,stroke:#007D9C,color:#000000,stroke-width:2px;
    classDef db fill:#1C5A96,stroke:#0B4884,color:#ffffff,stroke-width:2px;
    classDef redis fill:#DC382D,stroke:#9C1F17,color:#ffffff,stroke-width:2px;
    classDef ai fill:#6366F1,stroke:#4338CA,color:#ffffff,stroke-width:2px;
    classDef external fill:#64748B,stroke:#475569,color:#ffffff,stroke-width:2px;
    classDef aws fill:#232F3E,stroke:#FF9900,color:#ffffff,stroke-width:2px;

    Pujador["👤 Postor VIP / Pujador<br><i>[Inversionista / Coleccionista]</i>"]:::person
    Martillero["👤 Martillero Oficial<br><i>[Operador de Subasta]</i>"]:::person

    subgraph AWS["☁️ Boundary: Plataforma LiveBid (AWS Cloud - Free Tier Optimized)"]
        Edge["🛡️ CloudFront CDN & Reverse Proxy<br><i>[AWS CloudFront / NGINX]</i><br>Terminación SSL, compresión Brotli, protección DDoS y ruteo HTTP / WSS"]:::aws

        NextApp["🌐 Aplicación Web Frontend<br><i>[Contenedor: Next.js 15 / React 19]</i><br>Experiencia de usuario cinematográfica y exclusiva, SSR para catálogo SEO, audio en vivo y cliente WSS"]:::frontend

        GoGateway["⚡ API Gateway & WebSocket Hub<br><i>[Contenedor: Go / Golang]</i><br>Terminación de WebSockets bidireccionales concurrentes, validación JWT, heartbeat y room manager"]:::backend

        AuctionEngine["🔨 Auction & Core Engine<br><i>[Contenedor: Go / Golang - Clean Architecture]</i><br>Motor de subastas ultrarrápido (< 5ms): Anti-Sniping (extensión 15s), incrementos por tramo y adjudicación"]:::backend

        RiskAssessor["🛡️ AI Risk Assessor & Circuit Breaker<br><i>[Contenedor: Go / Golang]</i><br>Evaluador de riesgo por postor con fallback en memoria en caso de latencia o corte de API externa"]:::backend

        Redis["🧠 Cache & Concurrency Store<br><i>[Almacén: Redis 7.2 In-Memory]</i><br>Locks atómicos distribuidos (SETNX / Redlock), caché caliente de precio actual y canales Pub/Sub"]:::redis

        Postgres["🗄️ Base de Datos Principal<br><i>[Almacén: PostgreSQL 16 - AWS RDS / EBS]</i><br>Persistencia relacional ACID: catálogo de bienes, peritajes, usuarios, garantías retenidas y actas legales"]:::db

        Broker["📨 Message Broker<br><i>[Contenedor: RabbitMQ 3.13]</i><br>Cola asíncrona de eventos de dominio para auditoría legal, facturación diferida y notificaciones masivas"]:::backend
    end

    Pasarela["🏛️ Pasarela de Pagos & Custodia<br><i>[Stripe / Niubiz API]</i>"]:::external
    GeminiAPI["🤖 Google Gemini 1.5 / 2.0 API<br><i>[Servicio Externo de Inteligencia Artificial]</i>"]:::ai
    NotifService["📲 Servicio Externo Notificaciones<br><i>[AWS SES / WhatsApp Twilio]</i>"]:::external

    Pujador -->|Navega en sala exclusiva [HTTPS]| Edge
    Martillero -->|Panel de control de remate [HTTPS]| Edge

    Edge -->|Ruta páginas y SSR| NextApp
    Edge -->|Ruta conexiones socket WSS| GoGateway
    Edge -->|Ruta API REST /api/v1| GoGateway

    NextApp -.->|Canal bidireccional en vivo [WSS]| GoGateway

    GoGateway -->|Despacha comandos de puja| AuctionEngine
    GoGateway -->|Consulta catálogo y perfiles| AuctionEngine

    AuctionEngine -->|Locks atómicos y consulta precio caliente| Redis
    AuctionEngine -->|Persistencia transaccional duradera (ACID)| Postgres
    AuctionEngine -->|Encola eventos de dominio aprobados| Broker
    AuctionEngine -->|Verifica riesgo crediticio / comportamiento| RiskAssessor

    RiskAssessor -->|Análisis predictivo de patrones de riesgo| GeminiAPI
    AuctionEngine -->|Congela / captura depósitos en garantía| Pasarela

    Broker -->|Consume eventos para auditoría y facturas| AuctionEngine
    Broker -->|Dispara alertas de adjudicación| NotifService

    Redis -.->|Broadcast Pub/Sub a instancias de socket| GoGateway
```

---

## 3. Catálogo de Contenedores

| Contenedor | Tecnología | Responsabilidad Principal | Consumo de Recursos en AWS Free Tier | Persistencia / Estado |
|---|---|---|---|---|
| **Aplicación Web Frontend** | Next.js 15, React 19, TypeScript, Vanilla/Tailwind CSS | Interfaz cinematográfica de alta gama, catálogo con SSR para indexación SEO, reproducción de audio dinámico y cliente WebSocket para pujas en vivo. | ~120 MB RAM (o desplegado serverless en Vercel / AWS Amplify Free Tier) | Stateless (estado reactivo en cliente con Zustand / React State). |
| **API Gateway & WebSocket Hub** | Go (Golang 1.23+), Gorilla / Coder WebSocket | Mantiene abiertas miles de conexiones WebSocket concurrentes con cero sobrecarga, gestiona autenticación JWT y rutea tráfico REST. | **~15 MB – 25 MB RAM** (ultra liviano) | En memoria volátil de proceso (mapa concurrente seguro con `sync.RWMutex`). |
| **Auction & Core Engine** | Go (Golang 1.23+), Clean Architecture | Ejecuta las reglas de negocio estrictas: validación de tramos de puja, extensión automática anti-sniping, verificación de garantías y martillazo. | **~20 MB – 30 MB RAM** | Desacoplado: delega a Redis y PostgreSQL. |
| **AI Risk Assessor & Circuit Breaker** | Go (Golang 1.23+) + Gemini SDK | Filtro de confianza y evaluación de riesgo en tiempo de puja. Incorpora Circuit Breaker para conmutar a reglas heurísticas locales si la API de IA supera los 150 ms. | ~10 MB RAM | Stateless con métricas en memoria. |
| **In-Memory Store (Redis)** | Redis 7.2 | Proporciona locks atómicos distribuidos (`SETNX`), almacenamiento de la oferta líder en memoria ($< 1$ ms) y canal Pub/Sub multihilo. | ~30 MB – 50 MB RAM | En memoria con snapshot periódico RDB. |
| **Base de Datos (PostgreSQL)** | PostgreSQL 16 (AWS RDS Free Tier db.t3.micro o contenedor) | Almacén de verdad contable y legal con transacciones ACID, constraints de integridad y tablas históricas particionadas. | Capa gratuita AWS RDS (20 GB SSD) | Disco persistente con backups automáticos diarios. |
| **Message Broker (RabbitMQ)** | RabbitMQ 3.13 / Amazon SQS | Desacopla tareas asíncronas pesadas (emisión de actas forenses, facturación y notificaciones) para no frenar la sala en vivo. | ~60 MB – 80 MB RAM | Colas duraderas persistidas en disco. |

---

## 4. Por qué esta arquitectura es viable en AWS Free Tier

Una de las observaciones críticas de la evaluación arquitectónica era la viabilidad de recursos y costos. Desplegar un clúster de microservicios en Node.js sobre una instancia `t2.micro` o `t3.micro` (1 GB de RAM) colapsa rápidamente por OOM (Out Of Memory).

Al adoptar **Go (Golang)**:
1. **Consumo total del backend Go:** Menos de **60 MB de RAM**, dejando más del 80% de la memoria del servidor disponible para Redis y el sistema operativo.
2. **Latencia Sub-milisegundo:** El motor de subastas en Go procesa el martillazo y la puja en $< 3$ ms, cumpliendo con los estándares de subastas de alta frecuencia.
3. **Escalabilidad Horizontal:** Si la startup crece, cada contenedor Go se empaqueta en imágenes Docker de apenas **18 MB** (utilizando `scratch` o `alpine`), listas para escalar en AWS ECS Fargate sin cambios de código.

---

## 5. Flujos de Datos y Protocolos de Comunicación

### A. Flujo en Tiempo Real (Hot Path — Subastas en Vivo)
1. El postor presiona "Ofertar $55,000" en la interfaz **Next.js**.
2. La oferta viaja por `WSS` al **WebSocket Hub (Go)** autenticada con JWT.
3. El **Auction Engine (Go)** adquiere un lock atómico en **Redis** en $pprox 1$ ms para evitar colisiones.
4. Se validan las reglas de dominio: incremento mínimo, extensión de 15s si es anti-sniping y garantía activa.
5. Se actualiza el precio en **Redis** y se publica un evento en el canal Pub/Sub.
6. El **WebSocket Hub** difunde el nuevo precio a todos los postores conectados en $< 25$ ms.
7. Se libera el lock en **Redis** y se emite el evento al Broker para persistencia asíncrona.

### B. Flujo Transaccional Frío (Cold Path — Persistencia & Auditoría)
1. El worker asíncrono consume el evento de puja confirmada desde el **Broker**.
2. Escribe el registro forense en **PostgreSQL** con timestamp de alta precisión y hash de integridad.
3. El hilo en vivo jamás se bloquea por operaciones de disco en la base de datos relacional.
