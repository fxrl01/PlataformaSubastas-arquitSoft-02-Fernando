# Especificación SDD 019: Interacción en Vivo (Chat/Reacciones WS), Vitrina Pública de Vendedores, Campana de Alertas y Blindaje de Endpoints

## 1. Objetivo
Incorporar las tres capacidades de plataforma social en tiempo real sugeridas y blindar todos los endpoints sensibles antes de compartir el repositorio en GitHub:
1. **Chat y Reacciones en Vivo por WebSocket en la Sala (`room.social`):** reacciones instantáneas (`🔥`, `👏`, `💎`, `🎉`) y mensajes breves de espectadores/participantes transmitidos por Redis Pub/Sub hacia todos los clientes conectados a la sala, conservando anonimato por paleta.
2. **Vitrina Pública de Vendedores Verificados (`GET /api/v1/vendedores/{id}/vitrina`):** ficha pública sanitizada (sin exponer correo, DNI, teléfono ni billetera privada) con biografía, insignia de verificación, reputación comunitaria y todos los bienes publicados por ese vendedor.
3. **Campana Superior de Notificaciones (`🔔 Alertas`):** panel `popover="auto"` anclado con avisos en tiempo real cuando superan la oferta del usuario, cuando una subasta guardada (`⭐`) está próxima a cerrar o cuando finaliza un lote seguido.
4. **Blindaje de Endpoints contra Vulnerabilidades (API Security Hardening):**
   - Autenticación obligatoria (`RequireAuth`) en escrituras sociales (`like`, comentarios, publicaciones y mensajes de sala).
   - Limitador de tasa por IP (`IPRateLimiter` -> `429 Too Many Requests`) en login, registro y asistente IA para prevenir fuerza bruta y agotamiento de cuota `GEMINI_API_KEY`.
   - Límite estricto de tamaño de payload (`http.MaxBytesReader` 2 MB), cabeceras OWASP (`X-Content-Type-Options`, `X-Frame-Options`, `Referrer-Policy`) y desactivación automática de `/swagger` y contraseñas demo cuando `APP_ENV=production`.
   - Puertos de PostgreSQL y Redis acotados a `127.0.0.1` en `docker-compose.yml`.

## 2. Criterios de Aceptación
- **CA-01:** Dos navegadores conectados a una sala reciben al instante reacciones y mensajes de chat vía WebSocket (`room.social`).
- **CA-02:** Al pulsar sobre un vendedor en Social Feed o Cronograma se abre su Vitrina Pública sin filtrar datos personales sensibles.
- **CA-03:** La campana superior muestra contador de alertas no leídas y permite saltar en un clic a la subasta afectada.
- **CA-04:** Los endpoints protegidos rechazan peticiones anónimas (`401`), excesos de ráfaga (`429`) y payloads sobredimensionados, manteniendo `go test -race`, `Jest 100%` y `Playwright E2E` en verde sin errores.
