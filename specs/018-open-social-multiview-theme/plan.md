# Plan Técnico SDD 018: Arquitectura Multi-Vista, Temas Claro/Oscuro, Comunidad y Perfil

## 1. Backend Go + PostgreSQL (Clean Architecture)
- **Esquema (`internal/adapters/persistence/schema_migrations.go`):**
  - Extender `usuarios` con `bio TEXT NOT NULL DEFAULT ''`.
  - Crear tablas `social_posts` y `social_comments` con índices por fecha y semilla inicial idempotente para que la comunidad tenga contenido real desde el primer arranque.
- **Perfil de Usuario (`PATCH /api/v1/auth/me`):**
  - Añadir `UpdateProfile` en `domain.ProfileUpdater` / `PostgresUserRepository` / `CachedUserRepository` (con invalidación Redis), `AuthService.UpdateProfile` y `AuthHandler.UpdateMe`.
- **Muro Social (`internal/adapters/handlers/social_handler.go` + repositorio):**
  - Puerto y adaptador PostgreSQL/Memoria para listar publicaciones con comentarios, crear publicaciones (con vínculo opcional a una subasta e imagen), registrar reacciones (`like`) y comentar.
- **Contrato OpenAPI 3.0.3 (`openapi.yaml` + `openapi.json`):**
  - Documentar las 5 rutas nuevas para mantener el guardián automático `swagger_handler_test.go` al 100%.

## 2. Frontend Responsivo Multi-Sección (`public/index.html`, `styles.css`, `app.js`)
- **Sistema de Temas (`dark-mode` + `dark-mode-toggle`):** `<meta name="color-scheme" content="light dark">` + script inline anti-FOUC + tokens CSS semánticos claros y oscuros con `light-dark()` y fallback `@media (prefers-color-scheme: dark)`.
- **Selectores Accesibles (`branded-select-styling`):** colores sólidos contrastados en `select` y `option` + mejora progresiva `appearance: base-select`.
- **5 Secciones SPA:**
  1. `Cronograma`: buscador instantáneo, filtro temporal (*En vivo*, *Próximas*, *Cerradas*), guardar en lista de seguimiento (*Watchlist*) y acceso en 1 clic a sala.
  2. `Sala en Vivo`: enfocada exclusivamente en las acciones del usuario actual, sin métricas privadas de otros postores.
  3. `Catálogo`: galería dedicada de bienes verificados y consignación con fotografías.
  4. `Social Feed`: publicaciones de vendedores, coleccionistas y martilleros con reacciones y comentarios persistidos en PostgreSQL.
  5. `Mi Perfil y Ajustes`: edición real de perfil (`PATCH /api/v1/auth/me`), bienes seguidos, mis lotes y preferencias de tema (*Sistema / Claro / Oscuro*).
- **Cache-busting:** sufijo `?v=20261126-v3` en `/styles.css` y `/app.js`.
