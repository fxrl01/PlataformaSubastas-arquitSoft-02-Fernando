# Tareas SDD 019

- [x] Crear `http_hardening.go` + `http_hardening_test.go` (Rate Limiter IP, límite de body, cabeceras OWASP y guardias de producción).
- [x] Implementar Chat/Reacciones en vivo por WebSocket (`sala_mensajes` + Redis Pub/Sub) y Vitrina Pública de Vendedor (`GET /api/v1/vendedores/{id}/vitrina`).
- [x] Proteger `POST /api/v1/social/posts/{id}/like` con `RequireAuth` y acotar puertos de BD/caché a `127.0.0.1`.
- [x] Actualizar OpenAPI 3.0.3 (`openapi.yaml` y `openapi.json`) y pruebas Go `-race`.
- [x] Integrar Campana de Notificaciones (`popover`), Vitrina de Vendedor y Chat/Reacciones en Vivo en `index.html`, `styles.css` y `app.js`.
- [x] Ejecutar Go, Jest 100%, Docker rebuild y Playwright E2E con 0 errores, crear commit local y entregar guía segura de `git push`.
