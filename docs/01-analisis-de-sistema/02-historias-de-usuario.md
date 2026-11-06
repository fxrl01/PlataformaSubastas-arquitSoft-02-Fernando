# Historias de usuario — LiveBid

> **Formato:** *Como [actor], quiero [acción], para [beneficio].*

---

## 1. Postor / Inversionista VIP (`POSTOR`)

| ID | Historia de usuario |
|---|---|
| **HU01** | Como postor, quiero explorar las subastas activas y programadas junto con los datos del encargado oficial y la ficha pericial del lote, para evaluar la legitimidad del bien antes de participar. |
| **HU02** | Como postor, quiero conectarme a la sala en vivo por WebSocket y ver la transmisión del martillero sincronizada con el precio líder, para competir en tiempo real. |
| **HU03** | Como postor, quiero enviar una puja con clave de idempotencia (`Idempotency-Key`) y firma opcional **EIP-712**, para asegurar que mi oferta no se duplique ni pueda ser repudiada. |
| **HU04** | Como postor, quiero recibir actualizaciones instantáneas (< 50 ms) cuando entra una nueva puja o se activa la extensión *Anti-Sniping* (+30 s), para reaccionar a tiempo antes del martillazo. |
| **HU05** | Como postor, quiero delegar un **Agente Autónomo de IA** configurando mi presupuesto máximo (`max_cap`) y estrategia (`CONSERVADORA`, `EQUILIBRADA`, `SNIPER_ANTI_EXTENSION`), para que puje por mí automáticamente con latencia sub-milisegundo sin superar mi techo financiero. |
| **HU06** | Como postor, quiero consultar el certificado de auditoría **Web3 (Raíz de Merkle y Prueba de Inclusión)** de una subasta, para comprobar matemáticamente que ninguna puja fue alterada ni omitida. |
| **HU07** | Como usuario recién registrado, quiero solicitar mi activación KYC e indicar mis datos de contacto/identidad, para quedar habilitado en la plataforma según mi rol. |

---

## 2. Martillero / Encargado Oficial (`MARTILLERO`)

| ID | Historia de usuario |
|---|---|
| **HU08** | Como martillero encargado, quiero que mis credenciales oficiales (nombre, cargo, licencia/registro y contacto) figuren en la ficha de la subasta, para respaldar legalmente el acto de remate. |
| **HU09** | Como martillero, quiero activar una subasta programada (`PENDIENTE` $\rightarrow$ `ACTIVA`) e iniciar el conteo regresivo del martillo (`ACTIVA` $\rightarrow$ `EN_CONTEO`), para conducir el ritmo del evento en vivo. |
| **HU10** | Como martillero, quiero ejecutar el cierre oficial (`CERRADA`) o dejar que el worker automático cierre las subastas vencidas generando el recibo de liquidación on-chain, para adjudicar el lote de forma transparente. |

---

## 3. Vendedor / Consignante (`VENDEDOR`)

| ID | Historia de usuario |
|---|---|
| **HU11** | Como vendedor, quiero crear una subasta y subir los **detalles completos del lote** (categoría, ubicación, precio de reserva, imágenes, certificado de peritaje y especificaciones técnicas), para maximizar la confianza y el valor de las ofertas. |
| **HU12** | Como vendedor, quiero registrar o actualizar los **detalles del encargado/martillero** responsable de mi subasta, para dejar constancia de quién dirige y supervisa la operación. |
| **HU13** | Como vendedor, quiero que la garantía del ganador se liquide de forma atómica al cerrarse la subasta (vía pasarela o Smart Contract Escrow), para asegurar el cobro sin riesgo de impago. |

---

## 4. Administrador de la Plataforma (`ADMIN`)

| ID | Historia de usuario |
|---|---|
| **HU14** | Como administrador, quiero listar y filtrar los usuarios registrados por estado (`PENDIENTE`, `ACTIVO`, `SUSPENDIDO`, `BLOQUEADO`) y rol, para gestionar el padrón de participantes y aprobar su KYC. |
| **HU15** | Como administrador, quiero asignar o modificar el rol de cualquier usuario (`POSTOR`, `VENDEDOR`, `MARTILLERO`, `ADMIN`) y cambiar su estado operativo con invalidación inmediata de caché, para aplicar control de acceso basado en roles (RBAC) en tiempo real. |
| **HU16** | Como administrador, quiero consultar el reporte de inteligencia antifraude de IA (*Wash Trading*, *Shill Bidding*, *Bot Sniping*) y cambiar en caliente (`hot-swap`) el modelo activo del **Multi-Model AI Router**, para adaptar la defensa del sistema sin reiniciar el servidor. |
| **HU17** | Como administrador o auditor, quiero inspeccionar la cadena de hashes SHA-256 (`hash_auditoria`) en PostgreSQL y la `MerkleRoot` Web3, para resolver cualquier disputa legal con evidencia criptográfica inmutable. |
