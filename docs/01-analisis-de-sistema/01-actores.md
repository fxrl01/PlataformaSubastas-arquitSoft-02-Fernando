# Actores del sistema — LiveBid

> **Proyecto:** LiveBid — Plataforma de Subastas en Tiempo Real de Alta Frecuencia (Web3 & IA Exponencial)  
> **Capa:** Análisis de Sistema · **Versión:** 3.0

---

## 1. Actores Humanos y Agentes Autónomos (RBAC)

| Actor | Rol en Sistema (`domain.UserRole`) | ¿Qué necesita realizar en LiveBid? |
|---|---|---|
| **Pujador / Inversionista VIP** | `POSTOR` | Registrarse y pasar verificación KYC (`ACTIVO`), congelar garantía fiduciaria o Web3 Escrow, explorar subastas activas y sus fichas periciales, unirse a la transmisión y sala en vivo (WebSocket), emitir pujas firmadas (HTTP/WS o EIP-712), delegar un **Agente Autónomo de IA** con techo presupuestario (`max_cap`), y auditar sus pruebas criptográficas Merkle. |
| **Martillero / Encargado Oficial** | `MARTILLERO` | Ser asignado como responsable legal del evento (`EncargadoSubasta`: nombre, número de colegiatura/registro SUNARP, contacto), activar la subasta en vivo, iniciar el conteo de martillo (`EN_CONTEO`), adjudicar el cierre (`CERRADA`) y supervisar la telemetría de la sala en tiempo real. |
| **Vendedor / Consignante** | `VENDEDOR` | Crear la subasta y subir la **ficha técnica y multimedia del lote** (`DetallesLote`: categoría, ubicación custodiada, precio base, precio de reserva, galería de fotos, certificado de peritaje oficial y especificaciones técnicas), asignar al encargado/martillero y recibir la liquidación atómica al cierre. |
| **Administrador / Auditor de Plataforma** | `ADMIN` | Aprobar el registro y KYC de usuarios (`PENDIENTE` $\rightarrow$ `ACTIVO`), asignar o revocar roles (`POSTOR`, `VENDEDOR`, `MARTILLERO`, `ADMIN`), suspender o bloquear cuentas ante fraude, monitorear las alertas del **Detector Asíncrono de Anomalías IA**, cambiar en caliente el modelo del **Multi-Model AI Router** y verificar la integridad forense (cadena SHA-256 y raíz de Merkle). |
| **Agente Autónomo de Puja (IA)** | Delegado por `POSTOR` | Operar dentro del motor de subastas en representación de un postor verificado, ejecutando estrategias (`CONSERVADORA`, `EQUILIBRADA`, `SNIPER_ANTI_EXTENSION`) sin exceder jamás el límite criptográfico autorizado ni competir contra su propio mandante. |

---

## 2. Sistemas Externos e Infraestructura Descentralizada

| Sistema Externo | Categoría | ¿Qué realiza en la arquitectura? |
|---|---|---|
| **Google Gemini AI (Multi-Model Engine)** | Inteligencia Artificial Exponencial | Evalúa el riesgo crediticio y de comportamiento en tiempo real (protegido por *Circuit Breaker* de 150 ms) y alimenta el radar asíncrono contra *Wash Trading*, *Shill Bidding* y *Bot Sniping*. |
| **Smart Contract Escrow & Red EVM** | Web3 / Blockchain Trustless | Verifica firmas criptográficas tipadas **EIP-712**, retiene colateral en contratos inteligentes (`EVMEscrowAdapter`) y ancla la **Raíz de Merkle (`MerkleRoot`)** de todas las pujas al cierre de la subasta. |
| **Pasarela de Pagos Fiduciaria (Stripe / Niubiz)** | Custodia Financiera Tradicional | Procesa retenciones preventivas (*hold*) y capturas finales (*capture*) de garantías en moneda fiduciaria mediante el puerto `EscrowPort`. |
| **Proveedor de Video Streaming (Cloudflare Stream / AWS IVS)** | Distribución Multimedia | Distribuye la señal de video en vivo del martillero con latencia sub-segundo, desacoplada del canal WebSocket de precios. |
| **Servicio de Identidad (RENIEC / KYC) y Notificaciones** | Cumplimiento y Mensajería | Verifica documentos de identidad de postores/vendedores/martilleros y envía notificaciones transaccionales de adjudicación. |

---

## 3. Nota de Diseño Arquitectónico

A diferencia de un marketplace convencional (compra asíncrona a precio fijo), **LiveBid** modela explícitamente cuatro roles operativos diferenciados (`POSTOR`, `VENDEDOR`, `MARTILLERO`, `ADMIN`) más la figura de **Agentes Autónomos de IA** bajo supervisión humana. El **Vendedor** aporta el activo y su peritaje técnico (`DetallesLote`), el **Martillero / Encargado** aporta la fe pública y conducción del remate (`EncargadoSubasta`), el **Administrador** gobierna la seguridad Zero-Trust (KYC, RBAC y enrutamiento de modelos IA), y el **Postor** compite en ventanas de milisegundos con respaldo criptográfico Web3.
