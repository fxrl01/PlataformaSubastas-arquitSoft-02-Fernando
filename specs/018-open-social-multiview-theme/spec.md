# Especificación SDD 018: Plataforma Abierta Multi-Sección, Modo Claro/Oscuro, Social Feed y Perfil Interactivo

## 1. Contexto y Motivación del Usuario
Tras revisar la versión previa en navegador, el usuario identificó oportunidades clave de experiencia y producto:
1. **Contraste de controles y temas:** el selector de lotes mostraba fondo blanco con letras claras casi invisibles hasta pasar el cursor; además, la aplicación requiere **Modo Claro y Modo Oscuro** bien combinados en todos los componentes.
2. **Caché de estáticos y visibilidad del Asistente Gemini:** el navegador retenía hojas CSS/JS previas (`max-age=3600`) sin versionado `?v=...`, impidiendo apreciar el asistente dinámico; además, se requiere un widget flotante vivo y explicar el esquema gratuito (*Free Tier*) de Google Gemini.
3. **Arquitectura de vistas separadas y centrada en el usuario:**
   - La vista inicial del usuario debe ser un **Cronograma / Buscador de Subastas** claro.
   - El **Catálogo de Bienes** debe vivir en su propia sección dedicada, no apilado como lista encima de la sala.
   - La **Sala en Vivo** no debe exponer detalles ni métricas internas de otros postores; solo debe guiar al usuario sobre las acciones que él realizará (con paletas anónimas en el historial).
4. **Apertura social e interacción tipo app moderna:** incorporar secciones reales de **Social Feed (Comunidad)** y **Mi Perfil y Ajustes** respaldadas en PostgreSQL.

## 2. Criterios de Aceptación
- **CA-01 (Modo Claro / Modo Oscuro y Contraste WCAG):** soporte nativo `color-scheme: light dark` con persistencia libre de FOUC, selector de 3 estados (*Sistema*, *Claro*, *Oscuro*) en Ajustes y botón rápido en cabecera; todos los `<select>` y `<option>` mantienen contraste legible tanto en reposo como al desplegarse.
- **CA-02 (Navegación Multi-Sección Abierta):** barra de navegación semántica (`aria-current="page"`) con 5 secciones independientes: **Cronograma**, **Sala en Vivo**, **Catálogo**, **Social Feed** y **Mi Perfil**, más versionado `?v=...` en activos CSS/JS.
- **CA-03 (Sala Centrada en el Usuario):** se eliminan puntajes de solvencia e identificadores de otros postores; se presenta una guía accionable paso a paso para el usuario actual y paletas anónimas en el historial.
- **CA-04 (Social Feed y Perfil en PostgreSQL):** endpoints `PATCH /api/v1/auth/me`, `GET /api/v1/social/posts`, `POST /api/v1/social/posts`, `POST /api/v1/social/posts/{id}/like` y `POST /api/v1/social/posts/{id}/comentarios` operativos en PostgreSQL y documentados en OpenAPI 3.0.3.
- **CA-05 (Asistente Dinámico Gemini):** burbuja flotante proactiva con indicador en línea que abre el drawer interactivo conectado a `POST /api/v1/ia/asistente` con atajos directos a las secciones de la app.
