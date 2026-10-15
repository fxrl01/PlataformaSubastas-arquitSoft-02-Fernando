# LiveBid — Plataforma de Subastas en Tiempo Real

> **Asignatura:** IS-488 Arquitectura de Software  
> **Docente:** Ing. Lizbeth Jaico Quispe  
> **Estudiante:** Fernando Olivares Chuhui  
> **Semestre:** 2026-II  
> **Enfoque de Diseño:** Software Design Description (SDD) · Clean Architecture · Modelo C4

---

## 📌 Descripción del Sistema

**LiveBid** es una plataforma transaccional de subastas en vivo de alto rendimiento y confianza cero (**Zero Trust**). Permite la concurrencia masiva de postores durante el conteo final del martillo, garantizando:

- **Resolución atómica de pujas:** Eliminación de condiciones de carrera (*Anti-Race Condition*) mediante locks distribuidos en memoria (Redis).
- **Latencia ultra baja:** Confirmación y difusión de nuevas ofertas en $< 200$ ms vía WebSockets y canales Pub/Sub.
- **Protección financiera previa (Escrow):** Retención preventiva de fondos antes de habilitar la emisión de pujas.
- **Trazabilidad legal inmutable:** Ordenamiento secuencial con timestamp oficial del servidor y broker de mensajería duradero (RabbitMQ).

---

## 📂 Estructura del Repositorio (Docs-as-Code)

El repositorio se organiza formalmente separando la especificación arquitectónica del código ejecutable:

```text
PlataformaSubastas-arquitSoft-02-Fernando/
├── docs/
│   ├── 01-analisis-de-sistema/             # Requisitos, actores, drivers y escenarios SEI
│   │   ├── 01-actores.md
│   │   ├── 02-historias-de-usuario.md
│   │   ├── 03-requisitos-funcionales.md
│   │   ├── 04-atributos-de-calidad.md
│   │   ├── 05-restricciones.md
│   │   ├── 06-drivers-arquitectonicos.md
│   │   ├── 07-decisiones-arquitectonicas.md
│   │   └── 08-escenarios-atributos-calidad.md
│   ├── 02-arquitectura-software/           # Estilo arquitectónico, componentes y diagramas
│   │   ├── arquitectura-inicial.md
│   │   ├── componentes-arquitectonicos.md
│   │   ├── estilo-arquitectonico.md
│   │   ├── enfoque-aquitectonico.md
│   │   ├── livebid-arquitectura.architecture.json
│   │   └── livebid-arquitectura.html       # Diagrama interactivo explorable
│   ├── 03-diseño-de-software/              # SDD: diseño interno de módulos, patrones y SOLID
│   │   └── diseño-interno/
│   │       ├── diseño-interno-de-modulos.md
│   │       ├── patrones-de-diseño.md
│   │       └── principios-de-diseño.md
│   ├── 04-modelo-c4/                       # Niveles 1 al 4 de Simon Brown
│   │   ├── Nivel1-DiagramadeContextodelSistema.md
│   │   ├── Nivel2-DiagramadeContenedores.md
│   │   ├── Nivel3-Diagrama-de-Componentes.md
│   │   └── Nivel4-DiagramadeCodigo.md
│   └── img/                                # Recursos gráficos y diagramas
├── tecnologia/
│   └── boilerplate-clean-arquitecture/     # Implementación referencial en TypeScript
│       ├── src/app/dominio/                # Entidades, agregados y contratos
│       ├── src/app/aplicacion/             # Caso de uso: ProcesarPuja
│       ├── src/app/infraestructura/        # Adaptadores en memoria y mocks
│       ├── src/app/presentacion/           # Controladores
│       └── src/app/pruebas/                # Pruebas unitarias de arquitectura
├── README.md
└── .gitignore
```

---

## 🗺️ Trazabilidad Arquitectónica (SDD)

| Nivel C4 | Documento | Pregunta que responde | Artefactos Clave |
| :--- | :--- | :--- | :--- |
| **Nivel 1** | [Contexto del Sistema](docs/04-modelo-c4/Nivel1-DiagramadeContextodelSistema.md) | ¿Quién usa el sistema y con qué servicios externos se conecta? | Postores, Martillero, Vendedores, Niubiz, Reniec, Streaming |
| **Nivel 2** | [Contenedores](docs/04-modelo-c4/Nivel2-DiagramadeContenedores.md) | ¿Qué aplicaciones y almacenes ejecutables existen? | SPA React, API Gateway, Realtime Bid Engine, Redis, Postgres, RabbitMQ |
| **Nivel 3** | [Componentes](docs/04-modelo-c4/Nivel3-Diagrama-de-Componentes.md) | ¿Cómo se estructuran los módulos internos del backend? | WebSocket Hub, Auth Zero Trust, Motor de Pujas, Custodia Escrow |
| **Nivel 4** | [Código](docs/04-modelo-c4/Nivel4-DiagramadeCodigo.md) | ¿Cómo interactúan las clases del caso de uso crítico? | Diagrama de clases UML y secuencia de `ProcesarPujaUseCase` |

---

## ⚙️ Ejecución de las Pruebas del Boilerplate Tecnológico

El proyecto cuenta con un conjunto de pruebas unitarias que verifican las reglas de dominio, el patrón Estado y el desacoplamiento de Clean Architecture:

```bash
# Ingresar al directorio del boilerplate
cd tecnologia/boilerplate-clean-arquitecture

# Ejecutar las pruebas directamente con Node (soporte nativo de TypeScript)
node --experimental-strip-types src/app/pruebas/subasta.pruebas.ts
```

### Casos de Prueba Verificados
1. **Zero Trust:** Rechazo automático de postor sin depósito de garantía en custodia.
2. **Puja Válida y Concurrencia:** Registro exitoso de oferta líder, cálculo de incremento mínimo y broadcast en tiempo real.
3. **Regla de Negocio:** Rechazo de ofertas que no superan el precio base + incremento mínimo.
4. **Anti-Sniping:** Extensión automática del cronómetro si entra una puja en los últimos 15 segundos.

---

## 🧭 Diagrama Interactivo de Arquitectura

Para explorar la vista interactiva con zoom, trazabilidad de flujos y navegación por componentes, abrir el archivo:
- [docs/02-arquitectura-software/livebid-arquitectura.html](docs/02-arquitectura-software/livebid-arquitectura.html)
