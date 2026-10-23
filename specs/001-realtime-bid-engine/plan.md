# Plan Técnico: Motor de Pujas en Tiempo Real (Realtime Bid Engine)

> **Feature ID:** `001-realtime-bid-engine`  
> **Fase:** Planificación Técnica (SDD) · **Basado en:** `spec.md`  
> **Patrones Clave:** Clean Architecture, State, Observer, Mutex Atómico, Adapter

---

## 1. Arquitectura Técnica del Feature

El motor de pujas se divide en dos trayectorias de datos para cumplir con el presupuesto de latencia ($< 200$ ms) y garantizar la durabilidad legal:

1. **Hot Path (Camino Caliente en Memoria):**  
   Recepción de WebSocket $\rightarrow$ Validación JWT $\rightarrow$ Verificación de Garantía $\rightarrow$ Mutex Atómico en Redis $\rightarrow$ Evaluación de Dominio $\rightarrow$ Publicación Pub/Sub en Redis $\rightarrow$ Broadcast WebSocket a la sala. **(Tiempo objetivo: $< 45$ ms)**.
2. **Cold Path (Camino Frío Asíncrono):**  
   Publicación de evento `PujaAceptada` en RabbitMQ $\rightarrow$ Worker en segundo plano $\rightarrow$ Persistencia transaccional en PostgreSQL 16 y generación de hash de auditoría. **(Tiempo objetivo: desacoplado, $< 500$ ms)**.

```mermaid
graph TD
    Client["📱 Cliente Postor (SPA)"]
    WSHub["⚡ WebSocket Gateway Handler"]
    UseCase["🎯 ProcesarPujaUseCase"]
    RedisLock[("🧠 Redis Mutex (SETNX)")]
    RedisPubSub[("📢 Redis Pub/Sub")]
    Domain["📦 Subasta (Aggregate)"]
    RabbitQueue[("📨 RabbitMQ (Cola 'pujas.confirmadas')")]
    PostgresDB[("🗄️ PostgreSQL 16")]

    Client -->|1. emit 'puja:emitir'| WSHub
    WSHub -->|2. Inyecta Command| UseCase
    UseCase -->|3. Adquiere lock en <2ms| RedisLock
    UseCase -->|4. Aplica reglas (Anti-sniping)| Domain
    UseCase -->|5. Publica nuevo precio| RedisPubSub
    UseCase -->|6. Encola para auditoría| RabbitQueue
    UseCase -->|7. Libera lock| RedisLock
    RedisPubSub -.->|8. Broadcast instantáneo| WSHub
    WSHub -.->|9. Evento 'sala:nuevo_precio'| Client
    RabbitQueue -->|10. Persistencia diferida| PostgresDB
```

---

## 2. Contratos de Comunicación (WebSocket Protocol)

### Eventos Entrantes (Cliente $\rightarrow$ Servidor)

#### `puja:emitir`
Payload transmitido por el cliente al enviar una postura:
```typescript
interface EmitirPujaPayload {
  subastaId: string;      // ID único del lote (ej. 'SUB-2026-001')
  monto: number;          // Monto total ofertado en soles (ej. 5200)
  postorId: string;       // Identificador del usuario emisor
  timestampCliente: number; // Timestamp Unix en cliente (solo para métricas de latencia de red)
}
```

### Eventos Salientes (Servidor $\rightarrow$ Cliente)

#### `puja:confirmada` (Unicast al postor emisor)
```typescript
interface PujaConfirmadaPayload {
  exito: true;
  pujaId: string;         // Identificador canónico (ej. 'PUJ-1729640000-1')
  monto: number;
  correlativo: number;    // Secuencia numérica en la subasta
  timestampServidor: string; // ISO-8601 del servidor oficial
  nuevoCierreProgramado: string; // Fecha de fin actualizada por Anti-Sniping
}
```

#### `sala:nuevo_precio` (Broadcast a todos los espectadores de la sala)
```typescript
interface SalaNuevoPrecioPayload {
  subastaId: string;
  precioActual: number;
  postorLiderId: string;
  aliasPostorLider: string; // Ej. 'Fernando O.***' (protección de privacidad)
  tiempoRestanteMs: number;
  esAntiSnipingActivado: boolean;
}
```

#### `puja:rechazada` (Unicast al postor emisor ante error)
```typescript
interface PujaRechazadaPayload {
  exito: false;
  codigoError: 'MONTO_INSUFICIENTE' | 'SUBASTA_CERRADA' | 'GARANTIA_REQUERIDA' | 'POSTOR_YA_ES_LIDER' | 'COLISION_CONCURRENTE';
  mensaje: string;
  precioActualVigente: number;
  incrementoMinimoRequerido: number;
}
```

---

## 3. Algoritmo del Lock Atómico Distribuido (Redis Mutex)

Para evitar condiciones de carrera cuando dos postores pujan en el mismo instante exacto:

1. **Adquisición del Lock:**
   ```bash
   SET lock:subasta:{subastaId} {workerUuid} NX PX 1500
   ```
   - `NX`: Solo asigna la clave si NO existe previamente.
   - `PX 1500`: Expiración automática a los 1,500 ms (evita deadlock si el nodo falla).
2. **Si `SET` devuelve `null`:**  
   Significa que otro proceso está evaluando una puja en esa misma sala. El sistema retorna de inmediato `COLISION_CONCURRENTE` para que el cliente reintente automáticamente o reciba el nuevo precio.
3. **Liberación Segura con Script Lua:**
   ```lua
   if redis.call("get", KEYS[1]) == ARGV[1] then
     return redis.call("del", KEYS[1])
   else
     return 0
   end
   ```
   Garantiza que un proceso no elimine el lock de otro proceso si hubo un retardo por recolección de basura o red.

---

## 4. Contratos de Puertos en Dominio (Clean Architecture)

```typescript
// Puerto de concurrencia
export interface IBidEngineLockPort {
  adquirirLock(subastaId: string, ttlMs?: number): Promise<boolean>;
  liberarLock(subastaId: string): Promise<void>;
}

// Puerto de persistencia y estado
export interface ISubastaRepository {
  obtenerPorId(id: string): Promise<Subasta | null>;
  guardar(subasta: Subasta): Promise<void>;
}

// Puerto de verificación financiera Zero Trust
export interface IPaymentEscrowPort {
  validarGarantiaActiva(postorId: string, subastaId: string): Promise<boolean>;
}

// Puerto de difusión en tiempo real
export interface IRoomNotifierPort {
  difundirNuevaPuja(puja: Puja, subasta: Subasta): Promise<void>;
}
```

---

## 5. Estrategia de Testing y Verificación

1. **Tests Unitarios Rápidos:** Ejecutables en memoria sin bases de datos externas para validar las reglas del dominio (`Subasta` y `ProcesarPujaUseCase`).
2. **Tests de Concurrencia Simultánea:** Lanzar 50 promesas asíncronas en paralelo compitiendo por la misma subasta con el mismo incremento. Verificar que exactamente 1 puja sea aprobada y 49 rechazadas con correlativo correlativo estricto.
3. **Tests de Anti-Sniping:** Simular una subasta con tiempo de fin a 5 segundos y verificar que la puja aceptada extienda el reloj a 35 segundos.

