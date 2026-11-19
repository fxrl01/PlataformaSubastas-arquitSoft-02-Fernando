# Especificación: Gobernanza RBAC, Encargado Oficial y Ficha del Lote

> **Feature ID:** `011-rbac-lot-management`  
> **Estado:** Implementado

## Alcance

- Bootstrap automático de cuenta raíz `ADMIN` al arrancar el servidor (`BootstrapAdmin`).
- Registro de usuarios con `rol_solicitado` (`POSTOR`, `VENDEDOR`, `MARTILLERO`), documento y teléfono, quedando en estado `PENDIENTE` hasta aprobación administrativa.
- Endpoints protegidos por rol (`RequireAuth` + `RequireRoles`) para que el `ADMIN` apruebe KYC, active/suspenda cuentas y asigne roles oficiales con invalidación inmediata de caché.
- Asignación formal del `EncargadoSubasta` (martillero oficial) y actualización de `DetallesLote` (`JSONB`), incluyendo carga de imágenes desde archivo local o URL desde la SPA.

## Criterios de Aceptación

- **Dado** un usuario en estado `PENDIENTE` o `SUSPENDIDO`, **cuando** intenta iniciar sesión o emitir una puja, **entonces** el sistema rechaza la operación.
- **Dado** un usuario con rol `VENDEDOR`, `MARTILLERO` o `ADMIN`, **cuando** actualiza la ficha del lote con imágenes y especificaciones, **entonces** se persiste en `detalles_lote_json` y queda auditable en `logs_auditoria`.
