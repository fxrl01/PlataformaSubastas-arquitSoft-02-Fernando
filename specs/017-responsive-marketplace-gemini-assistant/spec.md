# Especificación: UI Responsiva Orientada al Usuario, Catálogo Funcional y Asistente Lateral con Gemini

> **Feature ID:** `017-responsive-marketplace-gemini-assistant`  
> **Estado:** En Implementación

## Alcance

- **Lenguaje Centrado en el Usuario (Cero Jerga Técnica Pública):** Eliminación de referencias internas de ingeniería (`JWT`, `PostgreSQL ACID`, `RBAC`, `capture_method: manual`) en botones, modales y etiquetas públicas, restringiendo accesos de desarrollo (`API Docs` / `Swagger` y autocompletado rápido de prueba) exclusivamente a entornos `localhost`.
- **Marketplace y Publicación Funcional de Bienes sobre PostgreSQL:** Incorporación de una sección explorable de **Colección de Bienes en Subasta** (con filtros por categoría, estado en vivo, imagen `aspect-ratio` y cambio instantáneo de sala), estado real de sesión (`Visitante` cuando no ha iniciado sesión, opción de cerrar sesión) y flujo claro para que `VENDEDOR`, `MARTILLERO` o `ADMIN` publiquen bienes de valor con fotografía, categoría, ubicación, tasador y precio de reserva en PostgreSQL.
- **Diseño Responsivo Moderno:** Aplicación de tipografía fluida (`clamp()`), unidades dinámicas de viewport (`dvh`/`svh`), `@container` queries en tarjetas y paneles, y validación diferida `:user-invalid`.
- **Asistente Lateral Contextual con Google Gemini (`POST /api/v1/ia/asistente`):** Panel deslizable lateral (`<dialog>` top-layer con scroll-snap) conectado a **Google Gemini (`gemini-2.5-flash`)** con conocimiento integral de todas las funciones de la aplicación y del lote activo en PostgreSQL, incluyendo motor experto local de respaldo cuando `GEMINI_API_KEY` no esté configurada.

## Criterios de Aceptación

- **Dado** un usuario final visitando la plataforma en móvil, tablet o escritorio, **cuando** navega por el registro, login, catálogo y garantía, **entonces** la interfaz se adapta fluidamente sin desbordes horizontales y muestra textos claros de casa de subastas sin tecnicismos internos.
- **Dado** un usuario con rol `VENDEDOR`, `MARTILLERO` o `ADMIN`, **cuando** publica un nuevo bien con imagen, categoría y precio base, **entonces** el bien se persiste en PostgreSQL y aparece inmediatamente en la grilla de bienes subastados y en la sala en vivo.
- **Dado** cualquier visitante o usuario autenticado, **cuando** abre el **Asistente IA** lateral y realiza una consulta sobre cómo vender, cómo pujar o sobre el lote actual, **entonces** `POST /api/v1/ia/asistente` responde con contexto real de la subasta usando Google Gemini (o el asistente experto local si no hay API key externa).
