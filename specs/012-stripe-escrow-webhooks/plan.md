# Plan: Pasarela Stripe Escrow y Sincronización por Webhooks

1. Definir `PaymentGatewayPort` y `GuaranteeStatusSynchronizer` en `internal/core/domain/ports.go`.
2. Implementar `StripeAdapter` en `internal/adapters/payments/stripe_adapter.go` con soporte HTTP real y fallback determinista local para desarrollo/tests.
3. Conectar `GuaranteeService` (`PrepareStripeIntent`, `Create`, `HandleStripeWebhook`) y `PostgresGuaranteeRepository` (`SettleAuction`, `UpdateStatusByAuthorizationID`).
4. Validar firma HMAC-SHA256 y transiciones de estado mediante pruebas unitarias.
