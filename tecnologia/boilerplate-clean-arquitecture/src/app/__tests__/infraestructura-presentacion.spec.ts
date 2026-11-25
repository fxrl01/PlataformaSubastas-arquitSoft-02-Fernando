import { describe, expect, it, jest } from '@jest/globals';
import { CircuitBreakerIA, EstadoCircuitBreaker } from '../infraestructura/circuit-breaker-ia.ts';
import { GeminiRiesgoAdapter } from '../infraestructura/gemini-riesgo.adapter.ts';
import { PasarelaBancariaSimulada } from '../infraestructura/pasarela-bancaria-simulada.ts';
import { RepositorioGarantiasMemoria } from '../infraestructura/repositorio-garantias-memoria.ts';
import { ProcesadorPagosSimulado } from '../infraestructura/procesador-pagos-simulado.ts';
import { NotificadorSalaConsola } from '../infraestructura/notificador-sala-consola.ts';
import { MotorPujasSimulado } from '../infraestructura/motor-pujas-simulado.ts';
import { RepositorioSubastasMemoria } from '../infraestructura/repositorio-subastas-memoria.ts';
import { PujasControlador } from '../presentacion/pujas.controlador.ts';
import { ProcesarPujaCasoUso } from '../aplicacion/procesar-puja.caso-uso.ts';
import { Subasta } from '../dominio/modelos/subasta.modelo.ts';
import { EstadoSubasta } from '../dominio/modelos/estado-subasta.enum.ts';
import { Garantia, EstadoGarantia } from '../dominio/modelos/garantia.modelo.ts';
import { NivelRiesgo } from '../dominio/contratos/agente-riesgo.contrato.ts';

describe('Infraestructura y Presentación — Adaptadores y Controladores', () => {
  it('cubre CircuitBreakerIA (CLOSED -> OPEN -> HALF_OPEN -> CLOSED) y GeminiRiesgoAdapter', async () => {
    jest.spyOn(console, 'log').mockImplementation(() => {});
    jest.spyOn(console, 'warn').mockImplementation(() => {});
    jest.spyOn(console, 'error').mockImplementation(() => {});

    const gemini = new GeminiRiesgoAdapter();
    expect((await gemini.evaluarPostor('USR-VIP-01', 'SUB-1')).nivelRiesgo).toBe(NivelRiesgo.BAJO);
    expect((await gemini.evaluarPostor('USR-RIESGO_ALTO', 'SUB-1')).nivelRiesgo).toBe(NivelRiesgo.ALTO);
    expect((await gemini.evaluarPostor('USR-BOT-99', 'SUB-1')).nivelRiesgo).toBe(NivelRiesgo.ALTO);

    const cb = new CircuitBreakerIA(gemini);
    expect(cb.getEstado()).toBe(EstadoCircuitBreaker.CLOSED);

    // Camino exitoso normal con el circuito CLOSED (cubre rama false de HALF_OPEN)
    const okClosed = await cb.evaluarPostor('USR-VIP-01', 'SUB-1');
    expect(okClosed.modoFallbackActivado).toBe(false);

    // Simular caída de Gemini -> 2 fallos abren el circuito
    gemini.simularCaidaServidor(true);
    const f1 = await cb.evaluarPostor('USR-1', 'SUB-1');
    expect(f1.modoFallbackActivado).toBe(true);
    expect(cb.getEstado()).toBe(EstadoCircuitBreaker.CLOSED);

    const f2 = await cb.evaluarPostor('USR-2', 'SUB-1');
    expect(f2.modoFallbackActivado).toBe(true);
    expect(cb.getEstado()).toBe(EstadoCircuitBreaker.OPEN);

    // Mientras está OPEN antes del cooldown -> fallback inmediato
    const fOpen = await cb.evaluarPostor('USR-3', 'SUB-1');
    expect(fOpen.modoFallbackActivado).toBe(true);
    expect(cb.getEstado()).toBe(EstadoCircuitBreaker.OPEN);

    // Simular paso del tiempo > 3000ms para entrar en HALF_OPEN y recuperarse a CLOSED
    const realNow = Date.now;
    const nowSpy = jest.spyOn(Date, 'now').mockImplementation(() => realNow() + 4000);
    gemini.simularCaidaServidor(false);
    const recovered = await cb.evaluarPostor('USR-VIP-01', 'SUB-1');
    expect(recovered.modoFallbackActivado).toBe(false);
    expect(cb.getEstado()).toBe(EstadoCircuitBreaker.CLOSED);
    nowSpy.mockRestore();
  });

  it('cubre PasarelaBancariaSimulada, RepositorioGarantiasMemoria, ProcesadorPagosSimulado, Notificador y Motor', async () => {
    jest.spyOn(console, 'log').mockImplementation(() => {});

    const pasarela = new PasarelaBancariaSimulada();
    const errRet = await pasarela.retener('USR-SIN_FONDOS', 500);
    expect(errRet.exito).toBe(false);

    const okRet = await pasarela.retener('USR-1', 500);
    expect(okRet.exito).toBe(true);
    expect(await pasarela.capturar(okRet.autorizacionId)).toBe(true);
    expect(await pasarela.capturar(okRet.autorizacionId)).toBe(false);
    expect(await pasarela.capturar('INEXISTENTE')).toBe(false);

    const okRet2 = await pasarela.retener('USR-2', 500);
    expect(await pasarela.liberar(okRet2.autorizacionId)).toBe(true);
    expect(await pasarela.liberar('INEXISTENTE')).toBe(false);

    const repoGar = new RepositorioGarantiasMemoria();
    const gRet = new Garantia('G-1', 'SUB-1', 'USR-1', 500, 'A-1', EstadoGarantia.RETENIDA, 90);
    const gLib = new Garantia('G-2', 'SUB-1', 'USR-2', 500, 'A-2', EstadoGarantia.LIBERADA, 90);
    await repoGar.guardar(gRet);
    await repoGar.guardar(gLib);

    expect(await repoGar.obtenerGarantia('USR-1', 'SUB-1')).toBe(gRet);
    expect(await repoGar.obtenerGarantia('USR-2', 'SUB-1', true)).toBeNull();
    expect(await repoGar.obtenerGarantia('NO-EXISTE', 'SUB-1')).toBeNull();
    expect(await repoGar.obtenerPorSubasta('SUB-1')).toHaveLength(2);

    const procPagos = new ProcesadorPagosSimulado();
    expect(await procPagos.retenerGarantia('USR-9', 'SUB-9', 100)).toBe(true);
    expect(await procPagos.validarGarantiaActiva('USR-9', 'SUB-9')).toBe(true);
    await procPagos.liberarGarantia('USR-9', 'SUB-9');
    expect(await procPagos.validarGarantiaActiva('USR-9', 'SUB-9')).toBe(false);

    const motor = new MotorPujasSimulado();
    expect(await motor.adquirirLock('S-1')).toBe(true);
    expect(await motor.adquirirLock('S-1')).toBe(false);
    await motor.liberarLock('S-1');

    const repoSub = new RepositorioSubastasMemoria();
    expect(await repoSub.obtenerPorId('NO')).toBeNull();
    const sub = new Subasta('S-1', 'Lote', 1000, 100, 1000, EstadoSubasta.ACTIVA, new Date(Date.now() + 60000));
    await repoSub.guardar(sub);
    expect(await repoSub.obtenerPorId('S-1')).toBe(sub);

    const notif = new NotificadorSalaConsola();
    await notif.difundirNuevaPuja(
      { id: 'P1', subastaId: 'S-1', postorId: 'U1', monto: 1100, timestamp: new Date(), correlativo: 1 },
      sub,
      false
    );
    await notif.difundirNuevaPuja(
      { id: 'P2', subastaId: 'S-1', postorId: 'U2', monto: 1200, timestamp: new Date(), correlativo: 2 },
      sub,
      true
    );
    await notif.difundirTiempoExtendido('S-1', new Date());
    await notif.difundirFinSubasta('S-1', 'U2', 1200);
  });

  it('cubre todos los códigos HTTP de PujasControlador (400, 403, 409, 503, 200)', async () => {
    jest.spyOn(console, 'log').mockImplementation(() => {});
    const repo = new RepositorioSubastasMemoria();
    const motor = new MotorPujasSimulado();
    const pagos = new ProcesadorPagosSimulado();
    const notif = new NotificadorSalaConsola();
    const ctrl = new PujasControlador(new ProcesarPujaCasoUso(repo, motor, pagos, notif));

    // Parámetros incompletos -> 400
    expect((await ctrl.atenderEventoPuja({ subastaId: '', postorId: 'U1', monto: 100 })).status).toBe(400);

    // Sin garantía -> 403
    expect((await ctrl.atenderEventoPuja({ subastaId: 'S1', postorId: 'U1', monto: 100 })).status).toBe(403);

    // Lock ocupado -> 503
    pagos.habilitarGarantia('U1', 'S1');
    await motor.adquirirLock('S1');
    expect((await ctrl.atenderEventoPuja({ subastaId: 'S1', postorId: 'U1', monto: 100 })).status).toBe(503);
    await motor.liberarLock('S1');

    // Subasta inexistente / cerrada -> 400
    expect((await ctrl.atenderEventoPuja({ subastaId: 'S1', postorId: 'U1', monto: 100 })).status).toBe(400);

    // Subasta activa -> 200 éxito, 400 autopuja, 409 monto insuficiente
    await repo.guardar(new Subasta('S1', 'Item', 1000, 100, 1000, EstadoSubasta.ACTIVA, new Date(Date.now() + 60000)));
    expect((await ctrl.atenderEventoPuja({ subastaId: 'S1', postorId: 'U1', monto: 1100 })).status).toBe(200);
    expect((await ctrl.atenderEventoPuja({ subastaId: 'S1', postorId: 'U1', monto: 1200 })).status).toBe(400);
    pagos.habilitarGarantia('U2', 'S1');
    expect((await ctrl.atenderEventoPuja({ subastaId: 'S1', postorId: 'U2', monto: 1150 })).status).toBe(409);
  });
});
