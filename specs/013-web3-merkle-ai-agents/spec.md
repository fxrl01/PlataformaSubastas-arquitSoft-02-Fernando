# Especificación: Auditoría Criptográfica Web3, Router IA Multi-Modelo y Agentes Autónomos

> **Feature ID:** `013-web3-merkle-ai-agents`  
> **Estado:** Implementado

## Alcance

- Verificación opcional de pujas firmadas bajo estándar `EIP-712` (`Web3SignatureVerifierPort`).
- Cadena de auditoría `SHA-256` en `logs_auditoria` para hitos del ciclo de vida y pujas, con generación de árbol de Merkle (`MerkleRoot` y `MerkleProof`) y recibo de liquidación on-chain (`BlockchainEscrowPort`).
- Enrutador de modelos IA en caliente (`MultiModelRouter`) protegido por `CircuitBreaker`, detector asíncrono de anomalías (*Wash Trading* / *Bot Sniping*) y delegación de Auto-Puja (`AgentDelegationService`) con techo presupuestario estricto `max_cap`.
