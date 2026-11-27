# Tareas SDD 018: Plataforma Abierta Multi-Sección, Modo Claro/Oscuro, Social Feed y Perfil

- [x] Extender esquema PostgreSQL (`bio`, `social_posts`, `social_comments`) e implementar `PATCH /api/v1/auth/me` y endpoints `/api/v1/social/*` en Go.
- [x] Actualizar `openapi.yaml` + `openapi.json` y pruebas automatizadas Go `-race`.
- [x] Rediseñar `index.html`, `styles.css` y `app.js` con navegación de 5 vistas (Cronograma, Sala en Vivo centrada en el usuario, Catálogo, Social Feed, Mi Perfil), Modo Claro/Oscuro, corrección de `<select>` y widget dinámico del Asistente Gemini.
- [x] Validar suites Go, Jest 100%, reconstrucción Docker y Playwright E2E, registrando commit local limpio sin `git push`.
