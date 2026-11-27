# Plan y Tareas SDD 020

- [x] Crear `public/favicon.svg` y registrar `/favicon.svg` en `cmd/server/main.go`.
- [x] Unificar marca **LiveBid** en `index.html` y `app.js`, retirando `API Docs` de la cabecera pública (reubicado dentro del modal exclusivo de `ADMIN`).
- [x] Aplicar `form-factor-adaptation` en `styles.css`: áreas táctiles `44px` en `@media (pointer: coarse)`, barra de navegación inferior fija en móviles (`<= 768px`) con `safe-area-inset-*`, `scroll-padding-top/bottom`, eliminación de `overflow-x: hidden` en `body` y `text-wrap: balance / pretty`.
- [x] Validar con Playwright E2E en múltiples resoluciones (`320px`, `390px`, `768px`, `1440px`), Go `-race` y Jest `100%`, y registrar commit local limpio.
