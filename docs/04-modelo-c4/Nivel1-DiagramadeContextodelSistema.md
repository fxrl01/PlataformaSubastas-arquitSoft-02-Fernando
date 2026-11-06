# Nivel 1 · Diagrama de contexto del sistema — LiveBid

> **Proyecto:** LiveBid — Plataforma de Subastas en Tiempo Real  
> **Modelo:** C4 · Nivel 1 de 4 · **Versión:** 3.0 (RBAC, Encargado Oficial, Web3 & IA Exponencial)  
> **Estándar:** Software Design Description (SDD)

---

## 1. Objetivo

Representar la **Plataforma LiveBid** como una caja central y mostrar cómo interactúa con sus **cuatro perfiles de usuarios humanos (RBAC)**, **agentes autónomos delegados** y **sistemas externos** (IA generativa, red Web3/EVM, pasarelas fiduciarias y CDN de streaming).

---

## 2. Diagrama de Contexto (C4 Nivel 1)

```mermaid
graph TD
    classDef person fill:#08427B,stroke:#38BDF8,color:#ffffff,stroke-width:2px;
    classDef system fill:#0F172A,stroke:#0EA5E9,color:#ffffff,stroke-width:3px;
    classDef external fill:#334155,stroke:#94A3B8,color:#ffffff,stroke-width:2px;
    classDef web3 fill:#3B0764,stroke:#C084FC,color:#ffffff,stroke-width:2px;
    classDef ai fill:#312E81,stroke:#818CF8,color:#ffffff,stroke-width:2px;

    Pujador["👤 Postor VIP / Inversionista (POSTOR)<br><i>[Persona]</i><br>Deposita garantía, puja en tiempo real, firma EIP-712, delega Agentes IA y verifica Merkle Proofs"]:::person
    Martillero["🔨 Martillero / Encargado Oficial (MARTILLERO)<br><i>[Persona]</i><br>Responsable acreditado de la subasta: activa sala, canta conteo (EN_CONTEO) y adjudica el cierre"]:::person
    Vendedor["🏛️ Vendedor / Consignante (VENDEDOR)<br><i>[Persona]</i><br>Publica el lote con ficha pericial/fotos, asigna al encargado y recibe la liquidación"]:::person
    Admin["🛡️ Administrador / Auditor (ADMIN)<br><i>[Persona]</i><br>Aprueba registro y KYC, asigna roles RBAC, audita alertas de fraude IA y conmuta modelos LLM"]:::person

    LiveBid["🖥️ LiveBid: High-Frequency Web3 & AI Auction Platform<br><i>[Sistema de Software]</i><br>Motor de subastas en tiempo real (<25ms) con Arquitectura Hexagonal en Go, custodia híbrida fiduciaria/Web3, pruebas Merkle y agentes autónomos IA"]:::system

    GeminiAI["✨ Google Gemini Multi-Model AI<br><i>[Sistema Externo IA]</i><br>Evaluación predictiva de riesgo en tiempo real y análisis antifraude"]:::ai
    EVMChain["⛓️ Red Blockchain EVM & Smart Contract Escrow<br><i>[Sistema Externo Web3]</i><br>Verificación de colateral on-chain, firmas EIP-712 y anclaje de MerkleRoot al cierre"]:::web3
    Pasarela["💳 Pasarela Bancaria (Stripe / Niubiz)<br><i>[Sistema Externo]</i><br>Retención y captura de garantías fiduciarias"]:::external
    KYC["🆔 Servicio de Identidad (RENIEC / KYC)<br><i>[Sistema Externo]</i><br>Validación de identidad legal para activación de cuentas"]:::external
    VideoLive["🎥 Proveedor de Video Streaming (Cloudflare / AWS IVS)<br><i>[Sistema Externo]</i><br>Transmisión de video en vivo del martillero con baja latencia"]:::external

    Pujador -->|"Puja en vivo (WS/REST), delega Agente IA y audita Merkle"| LiveBid
    Martillero -->|"Registra credenciales de encargado, inicia conteo y cierra"| LiveBid
    Vendedor -->|"Registra subasta, sube DetallesLote y peritaje"| LiveBid
    Admin -->|"Gestiona usuarios/roles RBAC, aprueba KYC y opera Router IA"| LiveBid

    LiveBid -->|"Consulta score de riesgo (<150ms con Circuit Breaker)"| GeminiAI
    LiveBid -->|"Verifica colateral y emite recibo de liquidación con MerkleRoot"| EVMChain
    LiveBid -->|"Retiene y liquida garantías fiduciarias"| Pasarela
    LiveBid -->|"Verifica identidad documental"| KYC
    LiveBid -->|"Sincroniza estado de sala con stream en vivo"| VideoLive
```

---

## 3. Matriz de Actores y Responsabilidades de Contexto

| Actor | Rol (`UserRole`) | Responsabilidad en el Ecosistema LiveBid |
|---|---|---|
| **Postor / Pujador** | `POSTOR` | Se registra, obtiene aprobación KYC, congela su garantía en custodia, compite en vivo por WebSocket o delega su puja a un **Agente Autónomo de IA** con techo `max_cap`, y verifica criptográficamente el resultado mediante `MerkleProof`. |
| **Martillero / Encargado** | `MARTILLERO` | Figura en la subasta como `EncargadoSubasta` (con número de licencia/registro oficial y contacto), conduce la sala (`ACTIVA` $\rightarrow$ `EN_CONTEO` $\rightarrow$ `CERRADA`) y da fe pública de la adjudicación. |
| **Vendedor / Consignante** | `VENDEDOR` | Registra la subasta, carga la ficha técnica y pericial del bien (`DetallesLote`: categoría, reserva, ubicación, imágenes y certificado pericial) y designa al martillero encargado. |
| **Administrador / Auditor** | `ADMIN` | Gobierna el ciclo de vida de usuarios (`PENDIENTE` $\rightarrow$ `ACTIVO`, `SUSPENDIDO`, `BLOQUEADO`), asigna roles (`POSTOR`, `VENDEDOR`, `MARTILLERO`, `ADMIN`), supervisa alertas de *Wash Trading*/*Shill Bidding* y administra el `MultiModelRouter` de IA. |
