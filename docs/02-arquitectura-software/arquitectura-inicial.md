# 🚀 Arquitectura del Sistema LiveBid: High-Frequency Auction Engine

> **Proyecto:** LiveBid — Plataforma de Subastas en Tiempo Real de Alta Gama  
> **Tipo de Sistema:** Startup Cloud-Native, Event-Driven & AI-Assisted  
> **Estándar:** Software Design Description (SDD) · **Versión:** 2.0  
> **Infraestructura:** AWS Free Tier Ready (EC2 t3.micro + RDS + Redis)

---

## 1. De la Arquitectura Monolítica Tradicional al Diseño Startup Moderno

El diseño inicial típico de laboratorios universitarios (monolito de 3 capas lineales: *Presentación → Lógica → Base de datos única*) **no es viable para una startup de subastas en vivo**. En el mundo real, someter una sola base de datos relacional a miles de pujas simultáneas por segundo mediante llamadas HTTP bloqueantes genera saturación de conexiones (*connection pool exhaustion*), colisiones de concurrencia y latencias superiores a 1.5 segundos (haciendo imposible la experiencia del martillazo).

LiveBid adopta la arquitectura utilizada por plataformas líderes de trading y subastas en tiempo real (como Robinhood, Sotheby’s y OpenSea): **Desacoplamiento Hot Path / Cold Path con Clean Architecture en Go, WebSockets nativos y evaluación de riesgo con Inteligencia Artificial (Gemini).**

---

## 2. Diagrama de Arquitectura Global (Cloud-Native Startup Topology)

```mermaid
flowchart TD
    %% =========================
    %% CLIENTES Y DISPOSITIVOS
    %% =========================
    subgraph CLIENTS[" CLIENTES Y DISPOSITIVOS VIP"]
        Bidder[" Postor VIP / Inversionista<br><i>(Next.js App / Mobile Browser)</i>"]
        Auctioneer[" Martillero Oficial<br><i>(Dashboard de Control en Vivo)</i>"]
    end

    %% =========================
    %% PERÍMETRO Y EDGE (AWS)
    %% =========================
    subgraph EDGE_LAYER[" EDGE, INGRESS & CDN (AWS CloudFront)"]
        WAF["AWS WAF + SSL Termination<br><i>(Protección DDoS y Rate Limiting)</i>"]
        NextSSR[" Frontend Tier: Next.js 15 (React 19)<br><i>(Catálogo con SSR / SEO + Audio FX)</i>"]
    end

    %% =========================
    %% HOT PATH (TIEMPO REAL < 25ms)
    %% =========================
    subgraph HOT_PATH["HOT PATH — CAMINO CALIENTE (En Memoria & Tiempo Real)"]
        WSHub["Realtime Gateway & WebSocket Hub (Go)<br><i>(10,000+ sockets concurrentes vía Goroutines)</i>"]
        Engine["Auction & Concurrency Engine (Go)<br><i>(Clean Architecture: Anti-Sniping + Tramos)</i>"]
        RedisCluster[("Redis 7.2 In-Memory Data Store<br>• Mutex Atómico SETNX (Zero Race Conditions)<br>• Precio Líder en Memoria (< 1ms)<br>• Canales Pub/Sub de Difusión Multihilo")]
    end

    %% =========================
    %% INTELLIGENCE & TRUST TIER
    %% =========================
    subgraph AI_TRUST["INTELLIGENCE & ZERO-TRUST LAYER"]
        CircuitBreaker["Circuit Breaker en Memoria (Go)<br><i>(Degradación elegante si timeout > 150ms)</i>"]
        GeminiAI["Google Gemini AI Engine<br><i>(Detección predictiva de fraude y scoring)</i>"]
    end

    %% =========================
    %% COLD PATH (DURABILIDAD Y AUDITORÍA)
    %% =========================
    subgraph COLD_PATH["COLD PATH — CAMINO FRÍO (Persistencia y Liquidación Asíncrona)"]
        Broker["📨 Message Broker (RabbitMQ / SQS)<br><i>(Cola de eventos: PujaAceptada, SubastaCerrada)</i>"]
        AsyncWorker["Background Persistence Worker (Go)"]
        PostgresDB[("PostgreSQL 16 (AWS RDS Free Tier)<br>• Catálogo oficial de bienes y peritajes<br>• Registro transaccional inmutable (ACID)<br>• Trazabilidad legal y actas de remate")]
    end

    %% =========================
    %% INTEGRACIONES EXTERNAS
    %% =========================
    subgraph EXTERNAL["SERVICIOS EXTERNOS & LIQUIDACIÓN"]
        PaymentEscrow["💳 Pasarela de Pagos / Escrow<br><i>(Stripe / Niubiz: Retención de Garantías)</i>"]
        VideoCDN["Video Streaming CDN<br><i>(Cloudflare Stream / AWS IVS)</i>"]
        Notif["Alertas & Mensajería<br><i>(AWS SES / WhatsApp Twilio)</i>"]
    end

    %% CONEXIONES Y FLUJOS
    Bidder -->|"HTTPS Navegación"| WAF
    Auctioneer -->|"HTTPS Operación"| WAF
    WAF --> NextSSR
    VideoCDN -.->|"Video en vivo directo (< 1s)"| Bidder

    Bidder ==>|"Canal WSS Bidireccional (< 25ms)"| WSHub
    Auctioneer ==>|"WSS Comandos de Martillo"| WSHub

    WSHub -->|"Despacha Comando Puja"| Engine
    Engine <==>|"1. Adquiere Lock Atómico & Consulta Precio Líder"| RedisCluster
    Engine -.->|"2. Evaluación Preventiva de Riesgo"| CircuitBreaker
    CircuitBreaker -->|"Score Predictivo"| GeminiAI

    Engine -->|"3. Actualiza Precio y Emite Evento"| RedisCluster
    RedisCluster -.->|"4. Broadcast instantáneo a todos los clientes"| WSHub

    Engine -->|"5. Encola evento confirmado (Desacoplado)"| Broker
    Broker --> AsyncWorker
    AsyncWorker -->|"Persistencia Duradera ACID"| PostgresDB

    Engine -->|"Retención / Cobro de Garantía"| PaymentEscrow
    AsyncWorker -->|"Notificaciones diferidas"| Notif

    %% ESTILOS VISUALES MODERNOS
    style CLIENTS fill:#1E293B,stroke:#0F172A,color:#FFF
    style EDGE_LAYER fill:#0F172A,stroke:#38BDF8,stroke-width:2px,color:#FFF
    style HOT_PATH fill:#082F49,stroke:#0EA5E9,stroke-width:2px,color:#FFF
    style AI_TRUST fill:#312E81,stroke:#818CF8,stroke-width:2px,color:#FFF
    style COLD_PATH fill:#14532D,stroke:#22C55E,stroke-width:2px,color:#FFF
    style EXTERNAL fill:#3B0764,stroke:#C084FC,stroke-width:2px,color:#FFF
```

---

## 3. ¿Por qué este diseño es Innovador y Startup-Ready?

### A. La Separación Fundamental: *Hot Path* vs. *Cold Path*
Las startups de alto rendimiento no mezclan el procesamiento en vivo con el guardado en disco:
1. **Hot Path (Camino Crítico en Vivo — Latencia $< 25$ ms):**
   - Cuando un postor envía una oferta, el flujo viaja por **WebSockets nativos de Go** directamente a la memoria RAM.
   - Se valida la puja y se adquiere un **lock atómico distribuido en Redis (`SETNX`)** en $< 1.5$ ms.
   - El precio líder se actualiza en memoria y se difunde inmediatamente a toda la sala de postores.
   - **Resultado:** Los usuarios sienten la inmediatez absoluta del remate sin esperar a que el disco escriba.
2. **Cold Path (Camino Asíncrono de Persistencia y Auditoría):**
   - Una vez aprobada la puja en el *Hot Path*, se emite un evento al broker de mensajería.
   - Workers en segundo plano guardan el registro en **PostgreSQL 16**, calculan los hashes de auditoría legal y coordinan la facturación.
   - Si la base de datos relacional se ralentiza por una consulta pesada, **el remate en vivo jamás se detiene**.

---

### B. Inteligencia Artificial en el Flujo Crítico con *Circuit Breaker*
En lugar de depender pasivamente de reglas fijas, LiveBid integra a **Google Gemini AI** para evaluar anomalías de puja, suplantación o intentos de manipulación del precio:
- **Resiliencia Extrema (Circuit Breaker):** Si la API externa de IA tarda más de 150 ms o presenta microcortes, el Circuit Breaker conmuta de forma transparente a una heurística local en memoria de Go.
- La subasta **nunca se bloquea ni aborta** por fallos en servicios externos.

---

### C. Eficiencia Brutal de Recursos en AWS Free Tier (1 GB RAM)

| Componente | Enfoque Monolítico Tradicional | Enfoque LiveBid (Startup Moderna) | Impacto en AWS Free Tier |
|---|---|---|---|
| **Lenguaje Backend** | Node.js (~150 MB – 250 MB por proceso) | **Go compilado (~20 MB de RAM)** | 🟢 Ahorro de más del 80% de memoria en instancias `t2.micro` / `t3.micro`. |
| **Concurrencia** | Event Loop monohilo (riesgo de bloqueo) | **Goroutines nativas** (~2 KB de stack por socket) | 🟢 Soporta más de 10,000 conexiones concurrentes con mínima CPU. |
| **Estructura Interna** | Código acoplado a Express / TypeORM | **Clean Architecture (Hexagonal)** | 🟢 Dominio 100% puro, desacoplado y con pruebas unitarias en 0.002s. |
| **Frontend** | SPA básica sin SEO | **Next.js 15 (React 19) con SSR** | 🟢 Catálogo indexable en Google y experiencia cinematográfica exclusiva. |
| **Costos Operativos** | Exige múltiples servidores dedicados | **Arquitectura optimizada para Costo $0 inicial** | 🟢 Permite validar el producto en el mercado sin quemar capital inicial. |

---

## 4. Matriz de Componentes y Responsabilidades

| Contenedor / Módulo | Tecnología | Capa SDD | Responsabilidad Clave |
|---|---|---|---|
| **Frontend de Lujo** | Next.js 15, React 19, Tailwind CSS | Presentación | Sala de remate interactiva, gráficos en tiempo real, efectos sonoros del martillo y SSR para SEO. |
| **API Gateway & WS Hub** | Go (Golang 1.23+), Gorilla WebSockets | Presentación | Control de tráfico, autenticación JWT en handshake, heartbeat y broadcast concurrente. |
| **Auction Core Engine** | Go 1.23+, Clean Architecture | Dominio & Aplicación | Orquestador de la subasta: validación de tramos, extensión anti-sniping (+30s) y adjudicación. |
| **Caché & Mutex Atómico** | Redis 7.2 In-Memory | Infraestructura | Control de concurrencia atómica (anti-colisiones) y distribución de eventos Pub/Sub. |
| **Base de Datos Transaccional** | PostgreSQL 16 (AWS RDS Free Tier) | Infraestructura | Almacén de verdad contable y legal: usuarios, catálogo, depósitos en garantía y auditoría inmutable. |
| **Agente de Riesgo IA** | Gemini API + Circuit Breaker en Go | Infraestructura | Evaluación de patrones de fraude y validación de solvencia crediticia. |
s