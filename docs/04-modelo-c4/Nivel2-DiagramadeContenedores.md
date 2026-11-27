# Nivel 2 · Diagrama de contenedores — LiveBid

> **Proyecto:** LiveBid — Plataforma de Subastas en Tiempo Real  
> **Modelo:** C4 · Nivel 2 de 4 · **Versión:** 3.1 (Monolito Modular Go + 1 Sola Base de Datos PostgreSQL 16 + Pasarela Stripe & Servicios Externos)  
> **Estándar:** Software Design Description (SDD)

---

## 1. Objetivo

Descomponer el sistema **LiveBid** en sus contenedores ejecutables dejando **100% claro para el desarrollo continuo** que:
1. **PostgreSQL 16 es la ÚNICA Base de Datos del sistema (Single Source of Truth ACID)**.
2. **Redis NO es una segunda base de datos**: es únicamente un **bus efímero en memoria** (semáforo de `1.5s` y difusión WebSocket) que incluso puede reemplazarse en desarrollo local por memoria nativa de Go.
3. **Stripe API**, **Azure Blob Storage / Cloudflare CDN & KYC**, **Google Gemini IA** y **Web3** se conectan como servicios externos desacoplados, desplegando el clúster sobre **Microsoft Azure + Cloudflare Edge (`HTTPS`/`WSS`)**.

---

## 2. Diagrama de Contenedores (C4 Nivel 2)

```mermaid
graph LR
    classDef person fill:#08427B,stroke:#38BDF8,color:#ffffff,stroke-width:2px;
    classDef frontend fill:#1E293B,stroke:#60A5FA,color:#ffffff,stroke-width:2px;
    classDef backend fill:#082F49,stroke:#0EA5E9,color:#ffffff,stroke-width:2px;
    classDef membus fill:#451A03,stroke:#F59E0B,color:#FEF3C7,stroke-width:2px;
    classDef db fill:#052E16,stroke:#22C55E,color:#ffffff,stroke-width:3px;
    classDef ext fill:#2E1065,stroke:#C084FC,color:#ffffff,stroke-width:2px;

    subgraph Actores["1. Usuarios del Sistema"]
        Bidder["👤 Postor (POSTOR)"]:::person
        Staff["🔨 Vendedor / Martillero / Admin"]:::person
    end

    subgraph LiveBidBoundary["2. Plataforma LiveBid (Microsoft Azure VM + Cloudflare Edge + Docker)"]
        WebUI["🌐 Aplicación Web SPA<br><i>[HTML5 / JS / WebSockets]</i><br>Sala de subastas en vivo, pasarela de garantía Stripe, ficha del encargado/lote y panel Admin RBAC"]:::frontend

        GoServer["⚙️ Backend Core: Monolito Modular Hexagonal<br><i>[Contenedor Único: Go 1.23+ · :3000]</i><br>Contiene los 6 Módulos de Negocio:<br>M1. Usuarios/Roles · M2. Catálogo/Encargado/Lote<br>M3. Pagos Stripe · M4. Pujas en Vivo<br>M5. Copiloto IA · M6. Auditoría Legal"]:::backend

        InMemoryBus(("⚡ Bus en Memoria (Redis / Go)<br><i>[NO ES BASE DE DATOS]</i><br>Solo Semáforo 1.5s (SETNX)<br>y Megáfono Pub/Sub WS")):::membus

        PostgresDB[("🗄️ PostgreSQL 16 (ÚNICA BASE DE DATOS)<br><i>[Single Source of Truth ACID · :5432]</i><br>• usuarios & refresh_tokens<br>• subastas (Encargado + Lote JSONB)<br>• garantias (Hold Stripe)<br>• pujas & logs_auditoria (SHA-256)")]:::db
    end

    subgraph Externos["3. Pasarela Stripe y Servicios Externos"]
        StripeAPI["💳 Stripe API (Pasarela de Pagos)<br><i>PaymentIntents Hold, Capture, Cancel & Webhooks</i>"]:::ext
        CloudKYC["☁️ Azure Blob Storage / Cloudflare CDN & KYC<br><i>Fotos del lote, PDF peritaje y validación DNI/SUNARP</i>"]:::ext
        GeminiCloud["🤖 Google Gemini API (IA)<br><i>Scoring de riesgo con Circuit Breaker (<150ms)</i>"]:::ext
        Web3Network["⛓️ Pasarela Web3 Base L2 (Opcional)<br><i>Firmas EIP-712 y Sello Merkle</i>"]:::ext
    end

    Bidder & Staff -->|"HTTPS REST + WSS"| WebUI
    WebUI -->|"REST /api/v1/* + WS /ws"| GoServer

    GoServer <-->|"Lock 1.5s + Broadcast Sala"| InMemoryBus
    GoServer ==>|"Lectura y Escritura Persistente ACID"| PostgresDB

    GoServer <-->|"Retención / Cobro / Webhooks"| StripeAPI
    GoServer <-->|"URLs PDF/Fotos y Verificación"| CloudKYC
    GoServer -.->|"Consulta con Fallback Local"| GeminiCloud
    GoServer -.->|"Recibo Cripto Opcional"| Web3Network
```

---

## 3. Catálogo de Contenedores y Claridad Operativa

| Contenedor (`docker-compose.yml`) | Rol Arquitectónico | ¿Guarda Datos Persistentes? | Explicación para Desarrollo Continuo |
| :--- | :--- | :---: | :--- |
| **`livebid-api-server` (`:3000`)** | **Monolito Modular en Go 1.23+ + SPA Web** | No (Stateless) | Un solo binario que aloja los **6 módulos de negocio** y sirve el frontend. Cero complejidad de múltiples servidores. |
| **`livebid-postgres` (`:5432`)** | **ÚNICA BASE DE DATOS (`PostgreSQL 16`)** | **SÍ (100% ACID)** | Aquí vive toda la información del sistema (`usuarios`, `subastas`, `garantias`, `pujas`, `logs_auditoria`). |
| **`livebid-redis` (`:6379`)** | **Coordinador Efímero en Memoria** | **NO (0% Tablas)** | **No es una segunda base de datos.** Solo actúa como semáforo de `1.5s` durante una puja y canal de avisos en vivo. Si no está presente en local, Go usa memoria interna automáticamente. |
