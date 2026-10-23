# Especificación Funcional: Billetera de Custodia (Escrow) y Evaluación de Riesgo con IA

> **Feature ID:** `002-escrow-wallet-ai`  
> **Estado:** APROBADO · **Fase:** Especificación (SDD)  
> **Constitución Aplicable:** Artículos I, II, III y V  
> **Patrones:** Clean Architecture, Adapter, Circuit Breaker, Strategy

---

## 1. Propósito y Motivación de Negocio

La **Billetera de Custodia Zero Trust con IA** resuelve el fraude en subastas (postores que ganan y luego abandonan la compra, o cuentas robot/falsas que inflan precios artificialmente). 

Para lograrlo, el sistema implementa un flujo dual:
1. **Evaluación de Confianza con Agente de IA:** Antes de habilitar la participación, un agente de IA (modelo Gemini mediante API) analiza el historial del postor, velocidad de transacciones previas y reputación, asignando un **Score de Confianza (0 a 100)** y determinando el porcentaje de garantía exigido (ej. 10% estándar o 20% si hay riesgo elevado).
2. **Retención Preventiva en Custodia (Escrow):** Se congelan los fondos en la pasarela bancaria mediante pre-autorización (*hold*).
3. **Liquidación Inmediata al Martillazo Final:** Cuando la subasta concluye, los fondos del ganador son capturados (*capture*), mientras que las garantías de los postores perdedores son liberadas automáticamente de forma inmediata (*release*).
4. **Resiliencia ante Caída de la IA (Circuit Breaker):** Si la API de Gemini sufre latencia excesiva o caída, el sistema **nunca se detiene**: conmuta automáticamente a un motor de contingencia heurístico local.

---

## 2. Historias de Usuario

### US-01: Constitución de Garantía para Ingreso a Sala
**Como** postor interesado en un lote de alto valor,  
**quiero** depositar un monto de garantía preventiva mediante tarjeta de crédito/débito,  
**para** obtener autorización de ingreso y poder pujar en la sala en vivo.

### US-02: Evaluación Inteligente de Confianza del Postor (Agente IA)
**Como** martillero o administrador de la plataforma,  
**quiero** que el sistema evalúe automáticamente el perfil de riesgo del postor con IA,  
**para** exigir una garantía mayor a usuarios con comportamiento anómalo o historial sospechoso.

### US-03: Resiliencia del Sistema si la IA Falla (Circuit Breaker)
**Como** arquitecto de software y operador de la sala,  
**quiero** que el sistema active un mecanismo de contingencia si la API de IA no responde,  
**para** que los postores sigan ingresando sin interrupción de la subasta.

### US-04: Liquidación Automática Post-Martillazo
**Como** postor perdedor,  
**quiero** que mi garantía sea liberada de inmediato sin comisiones al terminar el remate,  
**para** disponer de mis fondos bancarios sin demoras.

---

## 3. Reglas de Negocio e Invariantes del Dominio

1. **Garantía Base Mínima:** El depósito base exigido es del **10% del precio base de la subasta**.
2. **Ajuste Dinámico por Score de IA:**
   - **Score $\ge 80$ (Riesgo Bajo):** Garantía estándar (10%).
   - **Score entre 50 y 79 (Riesgo Medio):** Garantía estándar (10%) + verificación de teléfono por SMS.
   - **Score $< 50$ (Riesgo Alto / Anomalía):** Garantía reforzada del **20%** para mitigar riesgo de abandono.
3. **Estado de la Garantía:** Ciclo de vida estricto: `RETENIDA` $\rightarrow$ `CAPTURADA_COBRO` (ganador) ó `LIBERADA` (perdedor).
4. **Operación Atómica de Liquidación:** Al adjudicar la subasta, se emite el cobro definitivo al ganador y se dispara la liberación en masa de todos los postores perdedores en menos de 3 segundos.
5. **Circuit Breaker para IA:** Si la API de IA acumula 3 fallos consecutivos o tarda más de 2,000 ms, el circuito se abre (`OPEN`) y aplica la regla heurística predeterminada (garantía estándar del 10% con score neutro de 75).

---

## 4. Criterios de Aceptación (Gherkin / Given-When-Then)

### Escenario 1: Depósito de garantía exitoso con evaluación positiva de IA
```gherkin
Given una subasta con precio base de S/. 50,000 (garantía base 10% = S/. 5,000)
And el postor "USR-01" tiene un perfil limpio
When el postor solicita habilitar su garantía para la subasta
Then el agente de IA evalúa al postor y otorga un Score de 90 (Riesgo Bajo)
And la pasarela bancaria congela S/. 5,000 en su tarjeta
And se genera la garantía "GAR-001" en estado "RETENIDA"
And el postor queda formalmente habilitado para emitir pujas en la sala
```

### Escenario 2: Ajuste de garantía aumentada por alerta de riesgo de IA
```gherkin
Given una subasta con precio base de S/. 50,000
And el postor "USR-99" presenta patrones anómalos o antecedentes de ofertas no pagadas
When solicita habilitar su garantía
Then el agente de IA detecta un Score de 40 (Riesgo Alto)
And el sistema exige una garantía reforzada del 20% (S/. 10,000)
And si el postor autoriza los S/. 10,000, su garantía se registra en estado "RETENIDA"
```

### Escenario 3: Resiliencia con Circuit Breaker ante caída de la API de IA (Gemini Offline)
```gherkin
Given que el servicio externo de IA (Gemini) está caído o sin conexión de red
When el postor "USR-02" solicita ingresar a la subasta
Then el Circuit Breaker detecta la falla e intercepta el error
And activa el modo contingencia (Fallback Heurístico Local)
And asigna un score neutro de 75 y solicita la garantía estándar del 10%
And la subasta continúa operando con normalidad sin bloquear al usuario
And el log registra "CIRCUIT_BREAKER_FALLBACK_ACTIVATED"
```

### Escenario 4: Liquidación automática al concluir la subasta
```gherkin
Given la subasta "SUB-01" finalizada con ganador "USR-01" por S/. 58,000
And 3 postores perdedores con garantías de S/. 5,000 retenidas
When el martillero ejecuta el martillazo de adjudicación
Then la garantía del ganador "USR-01" cambia a "CAPTURADA_COBRO" como abono inicial
And las garantías de los 3 postores perdedores cambian a "LIBERADA" de forma atómica
And se emiten las notificaciones de liberación bancaria a cada cliente
```

---

## 5. Matriz de Errores y Excepciones

| Código de Error | Descripción | Acción del Sistema |
|---|---|---|
| `FONDOS_INSUFICIENTES` | La tarjeta no tiene saldo para la retención | Notificar al usuario para cambiar de medio de pago |
| `GARANTIA_YA_ACTIVA` | El postor ya tiene una garantía congelada vigente | Evitar doble retención |
| `SUBASTA_NO_DISPONIBLE` | El remate ya inició martillazo o finalizó | Rechazar ingreso |
| `CIRCUIT_BREAKER_OPEN` | IA en modo contingencia | Aplicar regla determinista de respaldo sin detener el flujo |

