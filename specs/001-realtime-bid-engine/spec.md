# Especificación Funcional: Motor de Pujas en Tiempo Real (Realtime Bid Engine)

> **Feature ID:** `001-realtime-bid-engine`  
> **Estado:** APROBADO · **Fase:** Especificación (SDD)  
> **Constitución Aplicable:** Artículos I, II, III, IV y V

---

## 1. Propósito y Motivación de Negocio

El **Motor de Pujas en Tiempo Real** es el componente nuclear de LiveBid. Su función es procesar las ofertas concurrentes de los postores durante la fase de remate en vivo, garantizar el orden secuencial estricto de llegada, prevenir condiciones de carrera (Race Conditions), hacer cumplir la regla de extensión de tiempo (*Anti-Sniping*) y difundir el estado actualizado a toda la sala en menos de 300 ms.

---

## 2. Historias de Usuario

### US-01: Emisión de Puja en Sala Activa
**Como** postor registrado con garantía habilitada,  
**quiero** emitir una oferta superando el precio actual en al menos el incremento mínimo,  
**para** convertirme en el nuevo postor líder del lote en subasta.

### US-02: Difusión Instantánea del Precio Líder
**Como** espectador o postor en la sala en vivo,  
**quiero** ver en mi pantalla la actualización inmediata del precio y del postor líder en milisegundos,  
**para** tomar decisiones de puja informadas sin retrasos respecto a otros participantes.

### US-03: Extensión Dinámica Anti-Sniping
**Como** vendedor o martillero,  
**quiero** que el cronómetro se extienda automáticamente si entra una puja en los últimos segundos,  
**para** evitar que postores con bots o tácticas de "sniping" ganen el bien en el último milisegundo sin dar oportunidad a contraofertas legítimas.

### US-04: Notificación Inmediata de Rechazo
**Como** postor que envía una oferta inválida o superada en el mismo milisegundo,  
**quiero** recibir un mensaje claro con el motivo exacto del rechazo,  
**para** poder corregir mi oferta o reintentar antes de que caiga el martillo.

---

## 3. Reglas de Negocio e Invariantes del Dominio

1. **Estados Válidos:** Solo se admiten pujas si el lote se encuentra en estado `ACTIVA` o `EN_CONTEO_MARTILLO`. Cualquier puja en estado `PROGRAMADA`, `ADJUDICADA`, `DESIERTA` o `CANCELADA` debe ser rechazada de inmediato.
2. **Incremento Mínimo Obligatorio:**  
   $$MontoOfertado \ge PrecioActual + IncrementoMinimo$$
3. **Restricción de Autopuja:** El postor que actualmente es líder de la subasta no puede emitir una nueva puja sobre su propia oferta líder vigente.
4. **Regla de Anti-Sniping (Extensión de Tiempo):**  
   Si una puja válida es recibida cuando el tiempo restante del lote es menor a **15 segundos**, el tiempo de finalización programado se extiende automáticamente en **30 segundos** adicionales.
5. **Garantía Requerida (Zero Trust):** El postor debe contar con un registro de garantía activa en custodia (*Escrow*) asociado al lote.
6. **Marca Temporal Canónica:** El timestamp oficial de la puja es asignado exclusivamente por el servidor al momento de obtener el lock atómico en Redis.

---

## 4. Criterios de Aceptación (Gherkin / Given-When-Then)

### Escenario 1: Puja válida exitosa
```gherkin
Given que la subasta "SUB-01" está en estado "ACTIVA" con precio actual de S/. 5,000 y un incremento mínimo de S/. 200
And el postor "USR-88" tiene un depósito de garantía activo en custodia
When el postor "USR-88" emite una puja de S/. 5,200
Then el sistema adquiere el lock atómico en Redis
And registra la puja con correlativo incremental y timestamp del servidor
And el precio actual de la subasta pasa a ser S/. 5,200 con postor líder "USR-88"
And se emite un broadcast a toda la sala con el evento "sala:nuevo_precio" en menos de 200 ms
And el postor recibe confirmación de éxito con status 200
```

### Escenario 2: Activación de la regla Anti-Sniping
```gherkin
Given que la subasta "SUB-01" tiene un tiempo restante de 8 segundos para finalizar
When el postor "USR-42" emite una puja válida de S/. 5,500
Then la puja es aceptada como nuevo líder
And el cronómetro de la subasta se extiende automáticamente agregando 30 segundos al tiempo de cierre
And se difunde a la sala el evento "sala:tiempo_extendido" con el nuevo timestamp de cierre
```

### Escenario 3: Resolución atómica ante colisión concurrente (Race Condition)
```gherkin
Given que el precio actual de la subasta es S/. 5,000
When dos postores ("USR-A" y "USR-B") emiten simultáneamente una puja de S/. 5,200 en el mismo milisegundo exacto
Then el secuenciador atómico de Redis otorga el lock al primer hilo que ejecuta SETNX (ej. "USR-A")
And la puja de "USR-A" es aprobada, convirtiéndose en el nuevo precio líder (S/. 5,200)
And la puja concurrente de "USR-B" es rechazada con código de error "OFERTA_SUPERADA"
And el postor "USR-B" recibe una notificación privada invitándolo a pujar S/. 5,400 o superior
And no existe ninguna inconsistencia o doble adjudicación en el historial
```

### Escenario 4: Rechazo por falta de depósito de garantía (Zero Trust)
```gherkin
Given que el usuario "USR-99" está autenticado pero no ha depositado garantía en custodia para "SUB-01"
When intenta emitir una puja de S/. 6,000
Then el sistema rechaza la solicitud en la capa de aplicación sin adquirir lock en Redis
And retorna el error "GARANTIA_REQUERIDA" con código HTTP 403 / Evento WS "puja:rechazada"
And no se modifica el estado de la subasta
```

### Escenario 5: Rechazo de autopuja del líder vigente
```gherkin
Given que el postor "USR-88" ya es el líder actual con la oferta de S/. 5,200
When el postor "USR-88" intenta emitir otra puja de S/. 5,400
Then el sistema rechaza la operación con el error "POSTOR_YA_ES_LIDER"
And no se ejecuta ninguna transacción en la base de datos
```

---

## 5. Casos de Borde y Manejo de Errores

| Código de Error | Causa Raíz | Respuesta HTTP / WebSocket | Acción Requerida del Cliente |
|---|---|---|---|
| `MONTO_INSUFICIENTE` | El monto es menor a `precioActual + incremento` | 409 Conflict / `puja:rechazada` | Aumentar el monto ofertado |
| `SUBASTA_CERRADA` | El lote ya fue adjudicado o el tiempo expiró | 400 Bad Request / `puja:rechazada` | Deshabilitar botón de puja en UI |
| `GARANTIA_REQUERIDA` | Postor sin fondos congelados en Escrow | 403 Forbidden / `puja:rechazada` | Redirigir a pasarela de depósito |
| `POSTOR_YA_ES_LIDER` | El usuario intenta autopujarse | 400 Bad Request / `puja:rechazada` | Esperar a ser superado |
| `COLISION_CONCURRENTE`| Timeout en adquisición de lock en Redis | 503 Unavailable / `puja:reintentar` | Reintentar automáticamente en 50 ms |

