# Boilerplate Clean Architecture — LiveBid

Este directorio contiene la implementación técnica de referencia del motor de subastas de **LiveBid**, estructurado según los principios de **Clean Architecture** (Arquitectura Limpia / Puertos y Adaptadores).

## Estructura de Capas

```text
src/app/
├── dominio/            # Reglas de negocio puras (sin dependencias externas)
│   ├── modelos/        # Entidades y objetos de valor (Subasta, Puja)
│   └── contratos/      # Interfaces (Puertos) de persistencia y servicios
├── aplicacion/         # Casos de uso de la plataforma
│   └── procesar-puja.caso-uso.ts
├── infraestructura/    # Adaptadores concretos (en memoria, mocks, DB)
│   ├── repositorio-subastas-memoria.ts
│   └── procesador-pagos-simulado.ts
├── presentacion/       # Controladores que reciben peticiones y delegan al caso de uso
└── pruebas/            # Pruebas unitarias de las reglas de dominio
```

## Reglas de Arquitectura
1. **El Dominio es agnóstico:** No importa librerías de red, frameworks web ni ORMs.
2. **Inversión de dependencias:** Los casos de uso dependen exclusivamente de las interfaces definidas en el dominio.
3. **Sustituibilidad:** Las implementaciones de infraestructura pueden ser reemplazadas sin modificar los casos de uso.
