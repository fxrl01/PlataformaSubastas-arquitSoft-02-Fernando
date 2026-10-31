# Plan técnico: Garantías y escrow básico

```text
HTTP handler -> GuaranteeService -> SubastaRepository
                                -> GuaranteeRepository
                                -> PaymentEscrowPort
```

El adaptador de pagos actual es local y explícitamente sustituible. PostgreSQL
guarda la garantía y aplica una restricción única por garantía activa.
