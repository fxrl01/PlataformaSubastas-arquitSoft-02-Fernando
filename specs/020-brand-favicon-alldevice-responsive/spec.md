# Especificación SDD 020: Identidad LiveBid, Favicon Vectorial, Retiro de API Docs Público y Adaptación Universal de Factor de Forma (320px–2560px)

## 1. Motivación
1. **Higiene de pantalla principal:** `API Docs` no debe mostrarse en la barra principal pública ni siquiera en `localhost`; el acceso interactivo a `/swagger` se reubica exclusivamente dentro del panel privado de `ADMIN` (y por URL directa `/swagger` en desarrollo).
2. **Identidad de marca coherente:** unificar el nombre visible de la plataforma como **LiveBid** (*Subastas en Vivo & Comunidad*), alineado al 100% con la documentación arquitectónica, diagramas C4 y backend del proyecto.
3. **Favicon en pestañas del navegador:** incorporar `/favicon.svg` vectorial de alta nitidez servido por Nginx y Go.
4. **Adaptación responsiva total (`form-factor-adaptation`):** garantizar usabilidad sin desbordamiento horizontal desde `320px` (móvil compacto con barra de navegación inferior fija y `safe-area-inset-bottom`), pasando por tablets (`768px`, `1024px`) hasta monitores de escritorio (`1440px+`), cumpliendo áreas táctiles de `44x44px` (`pointer: coarse`), `scroll-padding` y tipografía balanceada.

## 2. Criterios de Aceptación
- **CA-01:** Ningún enlace a `API Docs` aparece en la cabecera principal pública.
- **CA-02:** `/favicon.svg` responde `200 OK` (`image/svg+xml`) y está enlazado en `<head>`.
- **CA-03:** En pantallas móviles (`<= 768px`), las 5 secciones pasan a una barra inferior fija ergonómica con `safe-area-inset-bottom`, la cabecera superior se compacta en una sola franja limpia y `scrollWidth <= clientWidth` se cumple estrictamente en `320px`, `390px`, `768px` y `1440px` sin usar `overflow-x: hidden` en `body`.
