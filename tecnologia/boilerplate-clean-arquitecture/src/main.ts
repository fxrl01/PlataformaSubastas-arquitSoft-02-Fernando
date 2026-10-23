import http, { IncomingMessage, ServerResponse } from 'node:http';
import { Subasta } from './app/dominio/modelos/subasta.modelo.ts';
import { EstadoSubasta } from './app/dominio/modelos/estado-subasta.enum.ts';
import { RepositorioSubastasMemoria } from './app/infraestructura/repositorio-subastas-memoria.ts';
import { MotorPujasSimulado } from './app/infraestructura/motor-pujas-simulado.ts';
import { ProcesadorPagosSimulado } from './app/infraestructura/procesador-pagos-simulado.ts';
import { NotificadorSalaConsola } from './app/infraestructura/notificador-sala-consola.ts';
import { GeminiRiesgoAdapter } from './app/infraestructura/gemini-riesgo.adapter.ts';
import { CircuitBreakerIA } from './app/infraestructura/circuit-breaker-ia.ts';
import { PasarelaBancariaSimulada } from './app/infraestructura/pasarela-bancaria-simulada.ts';
import { RepositorioGarantiasMemoria } from './app/infraestructura/repositorio-garantias-memoria.ts';
import { ProcesarPujaCasoUso } from './app/aplicacion/procesar-puja.caso-uso.ts';
import { RetenerGarantiaCasoUso } from './app/aplicacion/retener-garantia.caso-uso.ts';
import { LiquidarSubastaCasoUso } from './app/aplicacion/liquidar-subasta.caso-uso.ts';
import { PujasControlador } from './app/presentacion/pujas.controlador.ts';

// =========================================================================
// 1. BOOTSTRAPPING & INVERSIÓN DE DEPENDENCIAS (CLEAN ARCHITECTURE)
// =========================================================================

// Infraestructura y Persistencia
const repoSubastas = new RepositorioSubastasMemoria();
const repoGarantias = new RepositorioGarantiasMemoria();
const motorLocks = new MotorPujasSimulado();
const pasarelaBancaria = new PasarelaBancariaSimulada();
const notificadorSala = new NotificadorSalaConsola();

// Agente de IA con Circuit Breaker
const geminiAdapter = new GeminiRiesgoAdapter();
const circuitBreakerIA = new CircuitBreakerIA(geminiAdapter);

// Adaptador de compatibilidad para el motor de subasta (verifica si existe garantía activa)
const procesadorPagosAdapter = {
  validarGarantiaActiva: async (postorId: string, subastaId: string) => {
    const garantia = await repoGarantias.obtenerGarantia(postorId, subastaId);
    return garantia ? garantia.esActiva() : false;
  },
  retenerGarantia: async (postorId: string, subastaId: string, monto: number) => true,
  liberarGarantia: async (postorId: string, subastaId: string) => {}
};

// Casos de Uso
const ucProcesarPuja = new ProcesarPujaCasoUso(repoSubastas, motorLocks, procesadorPagosAdapter, notificadorSala);
const ucRetenerGarantia = new RetenerGarantiaCasoUso(circuitBreakerIA, pasarelaBancaria, repoGarantias);
const ucLiquidarSubasta = new LiquidarSubastaCasoUso(pasarelaBancaria, repoGarantias);

// Controladores
const controladorPujas = new PujasControlador(ucProcesarPuja);

// Semilla de datos inicial (Subasta activa en vivo)
const subastaInicial = new Subasta(
  'SUB-2026-001',
  'Vehículo Toyota Hilux 2022 4x4 (Lote de Liquidación)',
  45000,
  1000,
  45000,
  EstadoSubasta.ACTIVA,
  new Date(Date.now() + 7200000) // 2 horas de duración
);
await repoSubastas.guardar(subastaInicial);

// =========================================================================
// 2. ENRUTADOR HTTP NATIVO (REST API)
// =========================================================================

function leerJSON(req: IncomingMessage): Promise<any> {
  return new Promise((resolve, reject) => {
    let cuerpo = '';
    req.on('data', (chunk) => (cuerpo += chunk));
    req.on('end', () => {
      try {
        resolve(cuerpo ? JSON.parse(cuerpo) : {});
      } catch (err) {
        reject(err);
      }
    });
  });
}

function responderJSON(res: ServerResponse, status: number, data: any) {
  res.writeHead(status, {
    'Content-Type': 'application/json',
    'Access-Control-Allow-Origin': '*',
    'Access-Control-Allow-Methods': 'GET, POST, OPTIONS',
    'Access-Control-Allow-Headers': 'Content-Type, Authorization'
  });
  res.end(JSON.stringify(data, null, 2));
}

export const servidor = http.createServer(async (req, res) => {
  const url = req.url || '';
  const metodo = req.method || 'GET';

  // Manejo de CORS Preflight
  if (metodo === 'OPTIONS') {
    responderJSON(res, 204, {});
    return;
  }

  try {
    // 1. Health Check
    if (metodo === 'GET' && url === '/health') {
      responderJSON(res, 200, {
        status: 'UP',
        sistema: 'LiveBid Realtime Engine',
        version: '1.0.0',
        timestamp: new Date().toISOString(),
        circuitBreakerIA: circuitBreakerIA.getEstado(),
        arquitectura: 'Clean Architecture + Event-Driven'
      });
      return;
    }

    // 2. GET /api/v1/subastas/:id (Consulta de estado)
    if (metodo === 'GET' && url.startsWith('/api/v1/subastas/')) {
      const id = url.split('/')[4];
      const subasta = await repoSubastas.obtenerPorId(id);
      if (!subasta) {
        responderJSON(res, 404, { error: `Subasta ${id} no encontrada` });
        return;
      }
      responderJSON(res, 200, {
        id: subasta.id,
        titulo: subasta.titulo,
        precioActual: subasta.precioActual,
        incrementoMinimo: subasta.incrementoMinimo,
        postorLiderId: subasta.postorLiderId || 'NINGUNO',
        estado: subasta.estado,
        finProgramado: subasta.finProgramado,
        totalPujas: subasta.pujas.length
      });
      return;
    }

    // 3. POST /api/v1/subastas/:id/garantia (Retener garantía con IA)
    if (metodo === 'POST' && url.match(/^\/api\/v1\/subastas\/[\w-]+\/garantia$/)) {
      const subastaId = url.split('/')[4];
      const body = await leerJSON(req);
      const subasta = await repoSubastas.obtenerPorId(subastaId);
      if (!subasta) {
        responderJSON(res, 404, { error: `Subasta ${subastaId} no encontrada` });
        return;
      }

      const resultado = await ucRetenerGarantia.ejecutar({
        subastaId,
        postorId: body.postorId,
        precioBaseSubasta: subasta.precioBase
      });

      responderJSON(res, resultado.exito ? 200 : 400, resultado);
      return;
    }

    // 4. POST /api/v1/subastas/:id/puja (Emitir puja en vivo)
    if (metodo === 'POST' && url.match(/^\/api\/v1\/subastas\/[\w-]+\/puja$/)) {
      const subastaId = url.split('/')[4];
      const body = await leerJSON(req);
      const respuesta = await controladorPujas.atenderEventoPuja({
        subastaId,
        postorId: body.postorId,
        monto: Number(body.monto)
      });

      responderJSON(res, respuesta.status, respuesta);
      return;
    }

    // 5. POST /api/v1/subastas/:id/liquidar (Adjudicación final)
    if (metodo === 'POST' && url.match(/^\/api\/v1\/subastas\/[\w-]+\/liquidar$/)) {
      const subastaId = url.split('/')[4];
      const body = await leerJSON(req);
      const resultado = await ucLiquidarSubasta.ejecutar({
        subastaId,
        postorGanadorId: body.postorGanadorId,
        montoFinalAdjudicado: Number(body.montoFinal)
      });

      responderJSON(res, resultado.exito ? 200 : 400, resultado);
      return;
    }

    // Ruta no encontrada
    responderJSON(res, 404, { error: 'Ruta no encontrada en LiveBid API' });
  } catch (error: any) {
    responderJSON(res, 500, { error: error?.message || 'Error interno del servidor' });
  }
});

// =========================================================================
// 3. ARRANQUE DEL SERVIDOR
// =========================================================================

const PUERTO = Number(process.env.PORT) || 3000;

if (process.env.NODE_ENV !== 'test') {
  servidor.listen(PUERTO, () => {
    console.log(`
=================================================================
  🚀 LIVEBID REALTIME ENGINE INICIADO CON ÉXITO
=================================================================
  📡 Puerto HTTP/REST:    http://localhost:${PUERTO}
  🩺 Health Check:        http://localhost:${PUERTO}/health
  🏛️ Arquitectura:        Clean Architecture + Event-Driven
  🛡️ Seguridad:           Zero Trust + Escrow + Circuit Breaker
  📦 Lote de Prueba:      SUB-2026-001 (Toyota Hilux S/. 45,000)
=================================================================
    `);
  });
}
