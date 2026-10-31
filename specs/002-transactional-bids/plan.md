# Plan técnico: Pujas transaccionales

```text
HTTP handler -> AuctionBidService -> Domain Subasta
                              -> BidRepository
                                      -> PostgreSQL transaction
```

El servicio valida el agregado y el repositorio ejecuta el cambio de precio y
el historial dentro de una transacción. El `UPDATE` condiciona el precio
observado para detectar que otra puja ganó la carrera antes de confirmar.

