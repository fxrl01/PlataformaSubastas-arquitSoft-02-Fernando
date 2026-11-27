# Plan: UI Responsiva Orientada al Usuario, Catálogo Funcional y Asistente Lateral con Gemini

1. Crear `internal/adapters/ai/gemini_assistant.go` (cliente HTTP hacia Google Gemini `gemini-2.5-flash` con prompt de sistema completo + motor contextual local de respaldo) y exponer `POST /api/v1/ia/asistente` en `AgentAIHandler`, `cmd/server/main.go` y `openapi.yaml`/`openapi.json`.
2. Actualizar `public/index.html` reemplazando textos técnicos por lenguaje comercial claro, añadiendo la sección **Colección de Bienes en Subasta** (filtros + grilla de tarjetas `<article>`), enriqueciendo el formulario de publicación de bienes y agregando el drawer lateral `<dialog id="aiAssistantDrawer">`.
3. Modernizar `public/styles.css` con grillas responsivas, `clamp()`, `@container` queries, estilos del drawer lateral con `scroll-snap-type: x mandatory` y validación `:is(:user-invalid, .user-invalid-fallback)`.
4. Actualizar `public/app.js` para manejar sesión real de visitante + cierre de sesión, renderizado reactivo de la grilla de subastas desde PostgreSQL, visibilidad `localhost` de herramientas técnicas y chat contextual en vivo con el Asistente IA.
5. Verificar `go test -race`, reconstruir el clúster Docker y validar `Playwright E2E` sin regresiones.
