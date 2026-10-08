# Enfoque Arquitectónico — LiveBid

Mientras que el *Estilo Arquitectónico* describe cómo se relacionan los grandes componentes del sistema (como el Gateway, el Motor de Subastas y el Broker), el **Enfoque Arquitectónico** define cómo se organiza el código y las dependencias *dentro* de cada uno de esos componentes (servicios).

Para LiveBid, hemos adoptado **Clean Architecture (Arquitectura Limpia) / Arquitectura Hexagonal** como el estándar para los servicios principales (especialmente el `Auction Engine`).

## ¿Qué es Clean Architecture en el contexto de LiveBid?

Es una filosofía de diseño que coloca las reglas de negocio (el Dominio) en el centro de la aplicación, aisladas completamente de detalles técnicos como la base de datos, el framework web, el broker de mensajería o las APIs externas.

### Dependencias internas

La regla de oro es la **Regla de Dependencia**: el código en las capas externas puede depender del código de las capas internas, pero el código de las capas internas *nunca* debe conocer nada sobre las capas externas.

| Capa | Responsabilidad en LiveBid | ¿Qué contiene? |
|---|---|---|
| **Dominio (Centro)** | Reglas puras del negocio. | Entidades (`Lote`, `Puja`, `Subasta`), Value Objects (`Monto`, `TrustScore`), Interfaces de repositorios (ej. `PujaRepository`), Lógica de validación de incrementos mínimos. |
| **Aplicación / Casos de Uso** | Orquesta el flujo de las operaciones de negocio. | `ProcesarPujaUseCase`, `CerrarLoteUseCase`. Usa el dominio para validar y las interfaces de infraestructura para guardar. |
| **Infraestructura (Adaptadores Secundarios)** | Implementa la persistencia y comunicación con el exterior. | `PostgresPujaRepository`, `RedisPriceCache`, `RabbitMqEventPublisher`, `StripePaymentAdapter`. |
| **Presentación (Adaptadores Primarios)** | Puntos de entrada al servicio. | Controladores REST, Handlers de WebSocket, Consumidores de eventos (ej. `RabbitMqBidConsumer`). |

## Beneficios para LiveBid

| Problema / Desafío | ¿Cómo lo resuelve Clean Architecture? |
|---|---|
| **Múltiples formatos de subasta (DA07)** | El `Auction Engine` aísla la lógica de formato en el Dominio (usando el patrón Strategy). Añadir un nuevo formato (ej. subasta holandesa) no requiere tocar la infraestructura ni los consumidores de RabbitMQ. |
| **Evolución del modelo de pago (DA04)** | Si pasamos de Stripe a PayPal, solo creamos un nuevo `PayPalPaymentAdapter` en la capa de Infraestructura. Los casos de uso y el dominio no cambian porque dependen de una abstracción (`PaymentGateway`). |
| **Testing del motor central (Mantenibilidad - DA08)** | Podemos escribir tests unitarios exhaustivos para `ProcesarPujaUseCase` simulando (mocking) la base de datos y el broker de mensajes, asegurando que la lógica crítica (quién gana, si el monto es válido) funciona perfectamente sin necesidad de levantar contenedores. |

## Diagrama del Enfoque (Arquitectura Hexagonal / Puertos y Adaptadores)

```mermaid
flowchart TD
    subgraph Adaptadores Primarios ["Adaptadores Primarios (Entrada)"]
        REST["REST API Controller\n(Peticiones web)"]
        Consumer["RabbitMQ Consumer\n(Nuevas pujas)"]
    end

    subgraph Aplicacion ["Capa de Aplicación (Casos de Uso)"]
        UseCase["ProcesarPujaUseCase\nCerrarLoteUseCase"]
    end

    subgraph Dominio ["Capa de Dominio (Núcleo)"]
        Entities["Entidades: Puja, Lote, Subasta\nValidación de Reglas de Negocio"]
        Ports["Puertos (Interfaces):\nIPujaRepository, IPaymentGateway"]
    end

    subgraph Adaptadores Secundarios ["Adaptadores Secundarios (Salida)"]
        DB["PostgresRepository\n(Persistencia)"]
        Cache["RedisCache\n(Precio actual)"]
        Publisher["RabbitMqPublisher\n(Eventos de dominio)"]
        Pago["StripeAdapter\n(Pasarela)"]
    end

    REST --> UseCase
    Consumer --> UseCase
    UseCase --> Entities
    UseCase --> Ports
    Ports -.->|"implementados por"| Adaptadores Secundarios

    style Adaptadores Primarios fill:#FEE2E2,stroke:#B91C1C
    style Aplicacion fill:#FEF3C7,stroke:#B45309
    style Dominio fill:#D1FAE5,stroke:#059669
    style Adaptadores Secundarios fill:#DBEAFE,stroke:#1D4ED8
```
