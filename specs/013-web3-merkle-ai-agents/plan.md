# Plan: Auditoría Criptográfica Web3, Router IA Multi-Modelo y Agentes Autónomos

1. Definir dominio Web3 (`web3.go`), agentes (`ai_agent.go`) y puertos (`AuditEventAppender`, `BlockchainEscrowPort`, `Web3SignatureVerifierPort`).
2. Implementar adaptadores `eip712_verifier.go`, `evm_escrow_adapter.go`, `multi_model_router.go` y `anomaly_detector.go`.
3. Integrar `Web3AuditService`, `AgentDelegationService` y registro de hitos en `AuctionCatalogService`.
4. Validar pruebas unitarias de Merkle, EIP-712, CircuitBreaker, Auto-Puja y flujo E2E de los 6 módulos.
