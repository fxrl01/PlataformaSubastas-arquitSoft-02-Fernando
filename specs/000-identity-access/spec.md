# Especificación: Identidad y Acceso

> **Feature ID:** `000-identity-access`  
> **Estado:** Aprobada para el MVP tradicional

## Propósito

Proporcionar identidad autenticada y autorización básica antes de permitir
operaciones de subasta. El usuario se identifica mediante correo y contraseña;
la wallet, Web3 y KYC avanzado quedan fuera de esta primera versión.

## Roles

- `POSTOR`: consulta subastas y emite pujas.
- `VENDEDOR`: registra bienes y consulta liquidaciones.
- `MARTILLERO`: crea, activa y cierra subastas.
- `ADMIN`: administra usuarios y auditoría.

## Estados

`PENDIENTE`, `ACTIVO`, `SUSPENDIDO`, `BLOQUEADO`.

Solo un usuario `ACTIVO` puede iniciar sesión y operar según sus permisos.

## Criterios de aceptación

### Registro

```gherkin
Given un correo no registrado y una contraseña válida
When el cliente solicita el registro
Then el sistema crea un usuario con rol POSTOR y estado PENDIENTE
And almacena únicamente un hash de la contraseña
And nunca devuelve la contraseña ni el hash en la respuesta
```

### Inicio de sesión

```gherkin
Given un usuario ACTIVO con credenciales válidas
When solicita iniciar sesión
Then el sistema devuelve un access token de corta duración
And devuelve la identidad pública y el rol del usuario
```

### Rechazo de acceso

```gherkin
Given credenciales inválidas o un usuario no ACTIVO
When solicita iniciar sesión
Then el sistema rechaza la operación sin revelar si el correo existe
```

### Autorización de puja

```gherkin
Given una solicitud autenticada
When el sistema procesa una puja
Then obtiene el postor desde la identidad autenticada
And ignora cualquier postorId enviado por el cliente
```
