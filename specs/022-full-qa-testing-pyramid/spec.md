# SDD 022 — Laboratorio Integral de Pruebas 360° (`/tests`: Cobertura, UI/WCAG, Seguridad OWASP, Estrés Spike, Resiliencia HA, E2E y Carga)

## 1. Contexto y Objetivo
Consolidar y visibilizar dentro del directorio `/tests` las 7 familias de pruebas de Arquitectura de Software para demostrar cuantitativamente todos los atributos de calidad definidos en los Sprints 1 y 2: Mantenibilidad (Cobertura), Usabilidad/Accesibilidad (UI/WCAG), Seguridad (OWASP DAST), Concurrencia/Rendimiento (Carga y Spike Anti-Sniping), Disponibilidad 99.95% (Failover HA) e Integridad End-to-End (E2E).

## 2. Criterios de Aceptación
1. **`tests/ui/`:** Valida en navegador real Modo Claro/Oscuro persistente, contraste, controles accesibles (`aria-label`, teclado `Escape`), barra inferior móvil (`375px`) sin desborde horizontal y recorrido por las 5 secciones más el Asistente IA.
2. **`tests/security/`:** Valida dinámicamente cabeceras OWASP, rechazo `401` ante JWT inválido, bloqueo RBAC `403` ante escalada `POSTOR -> ADMIN`, mitigación de fuerza bruta `429 Too Many Requests` (`Retry-After`), rechazo de payloads masivos (`> 2 MiB`) y neutralización XSS/SQLi.
3. **`tests/stress/`:** Ejecuta con k6 un pico súbito (*Spike Stampede*) sobre lecturas y ofertas simultáneas con `0` errores `5xx` y `p(95) < 400ms`.
4. **`tests/resilience/`:** Detiene `livebid-api-2` en caliente verificando `0%` de peticiones fallidas gracias al failover automático de Nginx hacia `livebid-api-1`, y restaura el clúster completo al terminar.
5. **`tests/coverage/` y `tests/run-all-tests.sh`:** Centraliza los reportes de cobertura de Go (`-race`) y Jest (`100%`) y provee un ejecutor maestro documentado en `tests/README.md` e integrado en GitHub Actions.
