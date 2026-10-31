# Plan: Idempotencia de pujas

1. Añadir la clave opcional al contrato de pujas.
2. Persistirla con restricción única en PostgreSQL.
3. Reproducir solicitudes repetidas y rechazar conflictos.
4. Añadir pruebas unitarias.
5. Validar pruebas y compilación.
