# Plan técnico: Difusión WebSocket

```text
HTTP bid -> AuctionBidService -> RedisRoomNotifier -> Redis Pub/Sub
                                                    ↓
                                      WebSocket Hub -> clients in room
```

El token se valida antes del upgrade. El identificador de subasta proviene de
la ruta y se verifica en PostgreSQL. No se aceptan montos ni comandos de puja
por WebSocket en este módulo.

