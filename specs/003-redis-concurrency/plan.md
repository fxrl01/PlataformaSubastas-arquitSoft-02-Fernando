# Plan técnico: Concurrencia distribuida con Redis

```text
HTTP handler -> AuctionBidService -> RedisBidLock
                                  -> PostgreSQL transaction
```

El servicio adquiere el lock antes de leer y validar el agregado, libera el
lock con `defer` y devuelve el error de contención sin ocultarlo. El adaptador
Redis queda fuera del dominio mediante `BidLockPort`.

