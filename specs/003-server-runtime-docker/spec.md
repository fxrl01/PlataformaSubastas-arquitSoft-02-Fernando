# Especificación Funcional: Runtime del Servidor, Orquestación y Entorno Docker

> **Feature ID:** `003-server-runtime-docker`  
> **Estado:** APROBADO · **Fase:** Especificación (SDD)  
> **Constitución Aplicable:** Artículos I, III, IV y V  
> **Componentes C4 Vinculados:** Nivel 2 (Contenedores) y Nivel 3 (Componentes)

---

## 1. Propósito y Motivación de Negocio

Para que la arquitectura de **LiveBid** sea completamente operativa, reproducible y óptima para el desarrollo en equipo y evaluación académica, se requiere un **punto de entrada unificado (`src/main.ts`)** y un **entorno de infraestructura multi-contenedor con Docker Compose**.

Esta especificación formaliza:
1. **Bootstrapping del Servidor:** Inicialización de todos los módulos mediante Inyección de Dependencias (DIP) conectando los casos de uso (`ProcesarPuja`, `RetenerGarantia`, `LiquidarSubasta`).
2. **Exposición de API REST y WebSockets:** Endpoints unificados `/health`, `/api/v1/subastas` y `/api/v1/garantias`.
3. **Almacenamiento Dual en Contenedores:** Levantar **PostgreSQL 16** (almacén frío ACID) y **Redis 7 Alpine** (almacén caliente en memoria) con un solo comando (`docker compose up`).
4. **Esquema de Base de Datos DDL (`docker/init.sql`):** Estructura relacional normalizada con índices de alto rendimiento para consultas en milisegundos.

---

## 2. Historias de Usuario

### US-01: Despliegue Local con un Solo Comando
**Como** desarrollador o docente evaluador,  
**quiero** ejecutar `docker compose up -d` en la raíz del proyecto,  
**para** tener levantada toda la infraestructura (PostgreSQL 16, Redis 7 y Servidor API) sin configuraciones manuales.

### US-02: Monitoreo de Salud del Sistema (Health Check)
**Como** operador de infraestructura o balanceador de carga,  
**quiero** consultar el endpoint `GET /health`,  
**para** verificar en milisegundos que el servidor, la memoria y los módulos del backend están operativos.

### US-03: API Unificada para Operaciones de Subasta
**Como** cliente frontend (SPA),  
**quiero** interactuar con endpoints HTTP REST estandarizados (`/api/v1/...`),  
**para** consultar el estado de los lotes, solicitar retención de garantías y coordinar el ingreso a las salas en vivo.

### US-04: Persistencia Relacional Estructurada
**Como** auditor legal de la subasta,  
**quiero** que la base de datos PostgreSQL contenga un esquema DDL con claves foráneas, restricciones de integridad y tabla de auditoría inmutable,  
**para** garantizar el respaldo legal definitivo de cada lote adjudicado.

---

## 3. Criterios de Aceptación (Gherkin / Given-When-Then)

### Escenario 1: Arranque exitoso del servidor y endpoint de salud
```gherkin
Given el entorno de ejecución Node.js con variables de configuración
When se inicia el servidor principal mediante "src/main.ts"
Then el servidor inicializa la inyección de dependencias de todos los módulos
And levanta el servidor HTTP en el puerto 3000
And una petición "GET /health" responde con status 200 y JSON { status: "OK", timestamp: "...", version: "1.0.0" }
```

### Escenario 2: Procesamiento de solicitud HTTP REST de garantía
```gherkin
Given el servidor en ejecución
When un cliente envía "POST /api/v1/subastas/SUB-2026-001/garantia" con payload { postorId: "USR-01", precioBase: 50000 }
Then el controlador delega al caso de uso "RetenerGarantiaCasoUso"
And se evalúa el riesgo con IA (o Circuit Breaker)
And responde con status 200 y el DTO de la garantía constituida
```

### Escenario 3: Procesamiento de solicitud HTTP REST de emisión de puja
```gherkin
Given una subasta activa y un postor con garantía previamente constituida
When el cliente envía "POST /api/v1/subastas/SUB-2026-001/puja" con { postorId: "USR-01", monto: 46000 }
Then el controlador ejecuta el caso de uso "ProcesarPujaCasoUso"
And adquiere el lock atómico, aplica reglas anti-sniping y retorna status 200 con la puja líder confirmada
```

### Escenario 4: Inicialización del esquema DDL en PostgreSQL
```gherkin
Given un contenedor de PostgreSQL 16 ejecutándose desde "docker-compose.yml"
When se monta el volumen "docker/init.sql"
Then se crean automáticamente las tablas "usuarios", "subastas", "pujas", "garantias" y "logs_auditoria"
And se aplican constraints de unicidad y los índices B-Tree en "subasta_id" y "created_at"
```

---

## 4. Esquema de Entidades DDL Relacionales

| Tabla | Propósito | Clave Primaria | Claves Foráneas | Índices Clave |
|---|---|---|---|---|
| `usuarios` | Registro de postores, martilleros y vendedores | `id` (VARCHAR) | — | `email` (UNIQUE) |
| `subastas` | Lotes de remate con precio base, estado y fin programado | `id` (VARCHAR) | `vendedor_id` | `estado`, `fin_programado` |
| `pujas` | Historial inmutable de ofertas aceptadas | `id` (VARCHAR) | `subasta_id`, `postor_id` | `(subasta_id, correlativo)` |
| `garantias` | Depósitos congelados en custodia (Escrow) | `id` (VARCHAR) | `subasta_id`, `postor_id` | `(subasta_id, postor_id, estado)` |
| `logs_auditoria` | Bitácora forense de eventos con hash SHA-256 | `id` (BIGSERIAL)| `subasta_id` | `created_at` |

