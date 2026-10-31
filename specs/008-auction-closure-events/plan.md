# Plan: Eventos de cierre

1. Extender el puerto de notificaciones en tiempo real.
2. Implementar el evento `auction.closed` en Redis Pub/Sub.
3. Emitirlo desde el caso de uso de cierre.
4. Validar ganador y cierre desierto.
5. Ejecutar pruebas y compilación.
