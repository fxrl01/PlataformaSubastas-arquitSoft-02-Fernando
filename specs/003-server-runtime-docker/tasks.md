# Tareas de Implementación: Runtime Go, Orquestación y Docker

> **Feature ID:** `003-server-runtime-docker`  
> **Implementación objetivo:** `tecnologia/livebid-backend`
> **Estado:** En progreso; no certificada para producción

## Runtime y API

- [x] Mantener el punto de entrada en
  `tecnologia/livebid-backend/cmd/server/main.go`.
- [ ] Exponer `/health` y las rutas de subastas mediante handlers Go.
- [ ] Añadir configuración tipada desde variables de entorno.
- [ ] Implementar apagado ordenado y propagación de señales.

## Infraestructura

- [ ] Reemplazar adaptadores locales por Redis y PostgreSQL reales.
- [ ] Actualizar [Dockerfile](../../Dockerfile) para compilar y ejecutar el
  módulo Go.
- [ ] Alinear `docker-compose.yml` con el binario Go y sus healthchecks.
- [ ] Verificar el esquema de `docker/init.sql` contra los repositorios Go.

## Verificación

- [ ] Ejecutar pruebas de integración HTTP contra el binario Go.
- [ ] Verificar arranque reproducible con `docker compose up`.
- [ ] Probar disponibilidad y reconexión de Redis/PostgreSQL.
- [ ] Documentar únicamente escenarios certificados por pruebas reales.

El antiguo servidor TypeScript no es el runtime objetivo de esta feature.
