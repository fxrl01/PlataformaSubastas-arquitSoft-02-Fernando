# Plan Técnico: Identidad y Acceso

> **Implementación objetivo:** Go 1.23+ · Clean/Hexagonal Architecture

## Capas

```text
HTTP handler -> AuthService -> UserRepository
                         -> PasswordHasher
                         -> TokenIssuer
```

El dominio no depende de HTTP, JWT, PostgreSQL ni de una librería criptográfica.
Los puertos se definen en `internal/core/domain/ports.go`.

## Decisiones del MVP

- El registro asigna `POSTOR` y `PENDIENTE`.
- El administrador o un flujo posterior activa la cuenta.
- Las contraseñas se almacenan mediante un hash con salt aleatorio.
- El access token es firmado por un adaptador configurable y tiene expiración.
- El identificador autenticado se transporta en el `context.Context`.
- El repositorio en memoria solo sirve para desarrollo y pruebas; PostgreSQL será
  el adaptador persistente antes de desplegar.

## Seguridad

- No registrar contraseñas, hashes ni tokens.
- Responder errores genéricos ante credenciales inválidas.
- No aceptar `postorId` del payload como identidad.
- La clave de firma debe provenir de configuración secreta.
- Usar HTTPS en entornos no locales.
