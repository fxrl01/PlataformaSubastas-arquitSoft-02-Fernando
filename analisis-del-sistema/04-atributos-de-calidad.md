# Atributos de calidad — LiveBid

Escenario analizado: durante un evento en vivo, 1,000 o más espectadores pueden estar conectados simultáneamente, varios pujando en la misma ventana de segundos sobre el mismo lote, mientras el video se transmite sin interrupciones.

| ID | Atributo de calidad | Escenario de calidad |
|---|---|---|
| AC01 | Rendimiento | Una puja debe confirmarse en menos de 300 ms y el nuevo precio debe difundirse a todos los espectadores en menos de 500 ms, incluso con 1,000+ usuarios conectados. |
| AC02 | Escalabilidad | El sistema debe soportar 1,000 o más espectadores/pujadores concurrentes por evento, y debe poder escalar horizontalmente agregando instancias del gateway y del servicio de difusión. |
| AC03 | Disponibilidad | El sistema debe mantenerse disponible al menos el 99.5% del tiempo durante una subasta en vivo — una caída durante el evento es un daño irreversible, a diferencia de un marketplace asíncrono donde el usuario puede volver más tarde. |
| AC04 | Seguridad | Los datos personales y de pago deben protegerse frente a accesos no autorizados; cada puja debe pasar por un filtro de detección de fraude antes de ser aceptada. |
| AC05 | Integridad y orden verificable | El orden en que se procesan las pujas debe ser verificable y trazable, usando el timestamp del servidor (broker) como fuente de verdad legal — nunca el reloj del dispositivo del cliente. |
| AC06 | Mantenibilidad | El motor de reglas de subasta debe poder incorporar nuevos formatos (descendente, sellada, benéfica) sin modificar el resto del sistema. |
| AC07 | Usabilidad | El espectador debe percibir el precio en pantalla como sincronizado con el video, aunque exista un desfase real de varios segundos entre la señal de video y el dato de precio. |
| AC08 | Portabilidad | El sistema debe poder ejecutarse tanto en Linux como en Windows sin cambios de código, mediante contenedores. |

## Por qué AC05 y AC07 no aparecen en un marketplace convencional

En un marketplace como el del caso GoPet, el orden de las operaciones casi nunca es motivo de disputa porque no hay competencia directa por el mismo ítem en el mismo instante. En una subasta en vivo, el orden **es** el producto — de ahí que AC05 (integridad y orden verificable) sea un atributo de calidad propio de este dominio, no una adaptación genérica de "seguridad". Lo mismo ocurre con AC07: la sincronía percibida entre dos fuentes de datos independientes (video y precio) no tiene equivalente en un sistema de compra tradicional.
