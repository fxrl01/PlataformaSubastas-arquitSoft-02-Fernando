# Especificación: Pasarela Stripe Escrow y Sincronización por Webhooks

> **Feature ID:** `012-stripe-escrow-webhooks`  
> **Estado:** Implementado

## Alcance

- Creación de `PaymentIntent` en Stripe con `capture_method: "manual"` calculando dinámicamente 10% o 20% del precio base según score de riesgo IA, sin almacenar PAN ni CVV (`RNF-01`).
- Verificación de retención (`requires_capture`) al registrar la garantía en `garantias` (`RETENIDA`).
- Liquidación automática al cierre: captura al ganador (`CAPTURADA_COBRO`) y cancelación de retención a los demás postores (`LIBERADA`).
- Endpoint de Webhook firmado (`Stripe-Signature` HMAC-SHA256) que sincroniza cambios asíncronos de estado (`payment_intent.succeeded`, `payment_intent.canceled`, `payment_intent.payment_failed`).
