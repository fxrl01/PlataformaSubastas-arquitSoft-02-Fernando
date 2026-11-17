# Historias de Usuario y Criterios de Aceptación — LiveBid

> **Proyecto:** LiveBid — Plataforma de Subastas en Tiempo Real  
> **Capa:** Análisis de Sistema · **Versión:** 3.1  
> **Formato Profesional:** Historia de Usuario (*Como / Quiero / Para*) + **Criterios de Aceptación verificables (*Dado / Cuando / Entonces*)**.

---

## Módulo 1: Registro, Inicio de Sesión y Administración de Roles (`RBAC` & `KYC`)

### HU-01: Registro de Usuario con Solicitud de Rol
- **Historia:** Como visitante, quiero registrarme ingresando mi nombre, correo, contraseña, documento de identidad, teléfono y el rol que deseo ejercer (`POSTOR`, `VENDEDOR` o `MARTILLERO`), para solicitar acceso a la plataforma.
- **Criterios de Aceptación:**
  - **CA-01.1:** *Dado* un correo no registrado y una contraseña de al menos 8 caracteres, *cuando* envío `POST /api/v1/auth/register` con `rol_solicitado = "VENDEDOR"`, *entonces* el sistema crea el usuario en estado `PENDIENTE` con rol inicial `POSTOR` y guarda `rol_solicitado = "VENDEDOR"` sin exponer el hash de la contraseña.
  - **CA-01.2:** *Dado* que un usuario intenta registrarse pidiendo `rol_solicitado = "ADMIN"`, *entonces* el sistema ignora el escalamiento de privilegios y mantiene su solicitud como `POSTOR`.

### HU-02: Inicio de Sesión Seguro y Renovación de Sesión (`JWT` + `Refresh Token`)
- **Historia:** Como usuario registrado y con cuenta `ACTIVA`, quiero iniciar sesión con mi correo y contraseña para obtener mi token de acceso (`JWT`) con mi rol actual y operar en los módulos autorizados.
- **Criterios de Aceptación:**
  - **CA-02.1:** *Dado* un usuario en estado `ACTIVO`, *cuando* envía credenciales válidas a `POST /api/v1/auth/login`, *entonces* recibe un `access_token` JWT (válido por 15 min), un `refresh_token` de un solo uso y los datos de su perfil (`role`, `status`, `kyc_aprobado`).
  - **CA-02.2:** *Dado* un usuario en estado `PENDIENTE`, `SUSPENDIDO` o `BLOQUEADO`, *cuando* intenta iniciar sesión, *entonces* el sistema rechaza el acceso con código `401 Unauthorized`.

### HU-03: Panel de Administración de Usuarios, Aprobación KYC y Asignación de Roles (`ADMIN`)
- **Historia:** Como `ADMIN` del sistema, quiero ver la lista de todos los usuarios registrados, aprobar su verificación KYC, cambiar su estado (`ACTIVO`, `SUSPENDIDO`, `BLOQUEADO`) y asignarles su rol oficial (`POSTOR`, `VENDEDOR`, `MARTILLERO`, `ADMIN`), para gobernar la seguridad de la plataforma.
- **Criterios de Aceptación:**
  - **CA-03.1:** *Dado* que el servidor arranca por primera vez, *entonces* ejecuta `BootstrapAdmin` garantizando que exista la cuenta `admin@livebid.pe` en estado `ACTIVO` y con rol `ADMIN`.
  - **CA-03.2:** *Dado* un usuario con `rol_solicitado = "MARTILLERO"` en estado `PENDIENTE`, *cuando* el `ADMIN` aprueba su cuenta y le asigna el rol `MARTILLERO`, *entonces* la caché en memoria se invalida al instante y el usuario puede dirigir subastas inmediatamente.

---

## Módulo 2: Catálogo, Imagen Referencial del Lote y Ficha del Encargado Oficial

### HU-04: Creación de Subasta y Subida de Imagen Referencial + Detalles del Lote
- **Historia:** Como `VENDEDOR` (o `ADMIN`), quiero publicar un lote subiendo **una imagen referencial desde mi propio dispositivo** (o URL), junto con su categoría, ubicación, precio base, precio de reserva, certificado de tasación y especificaciones técnicas, para que los postores vean exactamente qué se está subastando.
- **Criterios de Aceptación:**
  - **CA-04.1:** *Dado* un usuario con rol `VENDEDOR`, `MARTILLERO` o `ADMIN`, *cuando* selecciona una imagen referencial en el formulario y envía `PUT /api/v1/subastas/{id}/detalles-lote`, *entonces* PostgreSQL almacena la imagen dentro de `detalles_lote_json.imagenes_urls` y el catálogo muestra la fotografía inmediatamente.
  - **CA-04.2:** *Dado* que un usuario con rol `POSTOR` intenta modificar los detalles de un lote, *entonces* el middleware RBAC bloquea la petición con `403 Forbidden`.

### HU-05: Registro de los Datos del Encargado / Martillero Oficial de la Subasta
- **Historia:** Como `MARTILLERO`, `VENDEDOR` o `ADMIN`, quiero subir y mostrar en la subasta la ficha completa del encargado oficial (nombre, cargo, número de licencia/registro, teléfono/email de contacto y experiencia), para dar respaldo legal y transparencia al evento.
- **Criterios de Aceptación:**
  - **CA-05.1:** *Dado* una subasta creada, *cuando* se envía `PUT /api/v1/subastas/{id}/encargado`, *entonces* la subasta actualiza su `martillero_id` y persiste `encargado_json` en PostgreSQL, mostrándolo en la vista pública de la sala.

---

## Módulo 3: Pasarela de Pagos con Stripe y Custodia de Garantías (`Frontend + Backend`)

### HU-06: Retención Preventiva de Garantía con Tarjeta en Stripe (`Pre-Auth Hold`)
- **Historia:** Como `POSTOR`, quiero autorizar el depósito de garantía con mi tarjeta a través de **Stripe** antes de pujar, sabiendo que el dinero solo queda retenido preventivamente (*hold*) y no se me cobrará si no gano la subasta.
- **Criterios de Aceptación (Esfuerzo Combinado Frontend + Backend):**
  - **CA-06.1 (Backend crea intención segura):** *Dado* un `POSTOR` autenticado que solicita constituir garantía para una subasta, *cuando* llama al Backend, *entonces* el Backend calcula el monto obligatorio (10% del precio base, o 20% si el score de riesgo IA es alto), crea un `PaymentIntent` en Stripe con `capture_method = "manual"` y devuelve al Frontend el `client_secret` y el `payment_intent_id` sin recibir datos crudos de tarjeta.
  - **CA-06.2 (Frontend tokeniza y confirma):** *Dado* el `client_secret` en el Frontend, *cuando* el postor confirma su tarjeta en el widget de pago y el Backend valida el estado `requires_capture` (directamente o vía `POST /api/v1/webhooks/stripe`), *entonces* se guarda la garantía en PostgreSQL con estado `RETENIDA` y el postor queda habilitado para pujar.

### HU-07: Liquidación Automática en Stripe al Cerrar la Subasta (`Capture` al Ganador / `Cancel` al Resto)
- **Historia:** Como `VENDEDOR` y `POSTOR`, quiero que al cerrarse la subasta el sistema cobre automáticamente la garantía únicamente al postor ganador y libere de inmediato las retenciones en las tarjetas de todos los demás postores.
- **Criterios de Aceptación:**
  - **CA-07.1:** *Dado* que una subasta pasa a estado `CERRADA` con un `postor_lider_id`, *cuando* se ejecuta la liquidación, *entonces* el Backend invoca `Capture` en Stripe para el `PaymentIntent` del ganador (`estado = CAPTURADA_COBRO`) e invoca `Cancel` en Stripe para los `PaymentIntents` de los postores no ganadores (`estado = LIBERADA`).

---

## Módulo 4, 5 y 6: Sala en Vivo, Copiloto IA y Auditoría Forense

### HU-08: Pujas en Tiempo Real con Regla Anti-Sniping y Conteo de Martillero
- **Historia:** Como `POSTOR` con garantía `RETENIDA`, quiero enviar pujas en vivo y ver instantáneamente los cambios de precio y el conteo del martillero (`EN_CONTEO`), contando con 30 segundos extra si alguien puja en los últimos 15 segundos.
- **Criterios de Aceptación:**
  - **CA-08.1:** *Dado* un postor sin garantía `RETENIDA`, *cuando* intenta pujar, *entonces* el sistema rechaza la oferta con `402 Payment Required / 422 Garantía Insuficiente`.
  - **CA-08.2:** *Dado* una puja válida dentro de los últimos 15 segundos antes de `fin_programado`, *entonces* el sistema suma `+30 segundos` al cierre y notifica a toda la sala por WebSocket en `< 25 ms`.

### HU-09: Auto-Puja con Copiloto IA y Certificado de Auditoría Legal (`SHA-256`)
- **Historia:** Como `POSTOR` o `AUDITOR`, quiero poder delegar pujas automáticas hasta un tope (`max_cap`) y consultar al final el acta criptográfica (`SHA-256` + `MerkleRoot`) para verificar que el remate fue 100% íntegro.
- **Criterios de Aceptación:**
  - **CA-09.1:** *Dado* cualquier puja aceptada, *entonces* se inserta un registro en `logs_auditoria` cuyo `hash_actual` depende criptográficamente del `hash_anterior`, verificable mediante `GET /api/v1/subastas/{id}/auditoria-web3`.
