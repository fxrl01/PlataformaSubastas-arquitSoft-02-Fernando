import { describe, expect, it, jest } from '@jest/globals';
import { ProcesarPujaCasoUso } from '../aplicacion/procesar-puja.caso-uso.ts';
import { RetenerGarantiaCasoUso } from '../aplicacion/retener-garantia.caso-uso.ts';
import { LiquidarSubastaCasoUso } from '../aplicacion/liquidar-subasta.caso-uso.ts';
import { Subasta } from '../dominio/modelos/subasta.modelo.ts';
import { EstadoSubasta } from '../dominio/modelos/estado-subasta.enum.ts';
import { CodigoErrorPuja } from '../dominio/modelos/errores-puja.ts';
import { Garantia, EstadoGarantia } from '../dominio/modelos/garantia.modelo.ts';
import { NivelRiesgo } from '../dominio/contratos/agente-riesgo.contrato.ts';
import { RepositorioSubastasMemoria } from '../infraestructura/repositorio-subastas-memoria.ts';
import { MotorPujasSimulado } from '../infraestructura/motor-pujas-simulado.ts';
import { ProcesadorPagosSimulado } from '../infraestructura/procesador-pagos-simulado.ts';
import { NotificadorSalaConsola } from '../infraestructura/notificador-sala-consola.ts';
import { RepositorioGarantiasMemoria } from '../infraestructura/repositorio-garantias-memoria.ts';
import { PasarelaBancariaSimulada } from '../infraestructura/pasarela-bancaria-simulada.ts';

describe('Aplicación — Casos de Uso (ProcesarPuja, RetenerGarantia, LiquidarSubasta)', () => {
  it('cubre todas las rutas de ProcesarPujaCasoUso', async () => {
    const repo = new RepositorioSubastasMemoria();
    const motor = new MotorPujasSimulado();
    const pagos = new ProcesadorPagosSimulado();
    const notificador = new NotificadorSalaConsola();
    jest.spyOn(console, 'log').mockImplementation(() => {});

    const uc = new ProcesarPujaCasoUso(repo, motor, pagos, notificador);

    // 1. Sin garantía activa -> GARANTIA_REQUERIDA
    const sinGar = await uc.ejecutar({ subastaId: 'SUB-1', postorId: 'USR-1', monto: 2000 });
    expect(sinGar.exito).toBe(false);
    expect(sinGar.codigoError).toBe(CodigoErrorPuja.GARANTIA_REQUERIDA);

    // 2. Lock ocupado -> COLISION_CONCURRENTE
    pagos.habilitarGarantia('USR-1', 'SUB-1');
    await motor.adquirirLock('SUB-1');
    const colision = await uc.ejecutar({ subastaId: 'SUB-1', postorId: 'USR-1', monto: 2000 });
    expect(colision.exito).toBe(false);
    expect(colision.codigoError).toBe(CodigoErrorPuja.COLISION_CONCURRENTE);
    await motor.liberarLock('SUB-1');

    // 3. Subasta inexistente -> SUBASTA_CERRADA
    const noExiste = await uc.ejecutar({ subastaId: 'SUB-1', postorId: 'USR-1', monto: 2000 });
    expect(noExiste.exito).toBe(false);
    expect(noExiste.codigoError).toBe(CodigoErrorPuja.SUBASTA_CERRADA);

    // 4. Éxito normal sin Anti-Sniping
    const subasta = new Subasta(
      'SUB-1',
      'Lote 1',
      1000,
      100,
      1000,
      EstadoSubasta.ACTIVA,
      new Date(Date.now() + 90_000)
    );
    await repo.guardar(subasta);
    const okNormal = await uc.ejecutar({ subastaId: 'SUB-1', postorId: 'USR-1', monto: 1100 });
    expect(okNormal.exito).toBe(true);
    expect(okNormal.antiSnipingActivado).toBe(false);

    // 5. Éxito con Anti-Sniping (< 15s) -> llama difundirTiempoExtendido
    pagos.habilitarGarantia('USR-2', 'SUB-1');
    subasta.finProgramado = new Date(Date.now() + 5_000);
    const spyTiempo = jest.spyOn(notificador, 'difundirTiempoExtendido');
    const okSniping = await uc.ejecutar({ subastaId: 'SUB-1', postorId: 'USR-2', monto: 1200 });
    expect(okSniping.exito).toBe(true);
    expect(okSniping.antiSnipingActivado).toBe(true);
    expect(spyTiempo).toHaveBeenCalledTimes(1);

    // 6. Error de dominio (autopuja)
    const errDom = await uc.ejecutar({ subastaId: 'SUB-1', postorId: 'USR-2', monto: 1500 });
    expect(errDom.exito).toBe(false);
    expect(errDom.codigoError).toBe(CodigoErrorPuja.POSTOR_YA_ES_LIDER);

    // 7. Error inesperado no-ErrorDominioPuja (con mensaje y sin mensaje)
    const repoFalla = {
      obtenerPorId: async () => {
        throw new Error('DB caída');
      },
      guardar: async () => {},
    };
    const ucFallaMsg = new ProcesarPujaCasoUso(repoFalla, motor, pagos, notificador);
    const resFallaMsg = await ucFallaMsg.ejecutar({ subastaId: 'SUB-1', postorId: 'USR-1', monto: 2000 });
    expect(resFallaMsg.exito).toBe(false);
    expect(resFallaMsg.mensaje).toBe('DB caída');

    const repoFallaVacio = {
      obtenerPorId: async () => {
        throw {};
      },
      guardar: async () => {},
    };
    const ucFallaVacio = new ProcesarPujaCasoUso(repoFallaVacio, motor, pagos, notificador);
    const resFallaVacio = await ucFallaVacio.ejecutar({ subastaId: 'SUB-1', postorId: 'USR-1', monto: 2000 });
    expect(resFallaVacio.exito).toBe(false);
    expect(resFallaVacio.mensaje).toBe('Error al procesar la puja');
  });

  it('cubre todas las rutas de RetenerGarantiaCasoUso y LiquidarSubastaCasoUso', async () => {
    const repoGar = new RepositorioGarantiasMemoria();
    const pasarela = new PasarelaBancariaSimulada();
    const agenteOK = {
      evaluarPostor: async () => ({
        score: 90,
        nivelRiesgo: NivelRiesgo.BAJO,
        porcentajeGarantia: 0.1,
        justificacion: 'OK',
        modoFallbackActivado: false,
      }),
    };

    const ucRetener = new RetenerGarantiaCasoUso(agenteOK, pasarela, repoGar);

    // 1. Retención exitosa
    const rOk = await ucRetener.ejecutar({ postorId: 'USR-VIP', subastaId: 'SUB-2', precioBaseSubasta: 10000 });
    expect(rOk.exito).toBe(true);
    expect(rOk.garantia?.montoRetenido).toBe(1000);

    // 2. Doble retención rechazada
    const rDup = await ucRetener.ejecutar({ postorId: 'USR-VIP', subastaId: 'SUB-2', precioBaseSubasta: 10000 });
    expect(rDup.exito).toBe(false);
    expect(rDup.mensaje).toContain('ya cuenta con una garantía activa');

    // 3. Rechazo bancario (con mensaje y con fallback de mensaje vacío)
    const rSinFondos = await ucRetener.ejecutar({ postorId: 'USR-SIN_FONDOS', subastaId: 'SUB-2', precioBaseSubasta: 10000 });
    expect(rSinFondos.exito).toBe(false);
    expect(rSinFondos.mensaje).toContain('Fondos insuficientes');

    const pasarelaMsgVacio = {
      retener: async () => ({ exito: false, autorizacionId: '', mensaje: '' }),
      capturar: async () => false,
      liberar: async () => false,
    };
    const ucRetenerVacio = new RetenerGarantiaCasoUso(agenteOK, pasarelaMsgVacio, repoGar);
    const rMsgVacio = await ucRetenerVacio.ejecutar({ postorId: 'USR-OTRO', subastaId: 'SUB-2', precioBaseSubasta: 10000 });
    expect(rMsgVacio.exito).toBe(false);
    expect(rMsgVacio.mensaje).toBe('Error al procesar la retención bancaria');

    // 4. LiquidarSubastaCasoUso: sin garantías
    const ucLiquidar = new LiquidarSubastaCasoUso(pasarela, repoGar);
    const liqVacia = await ucLiquidar.ejecutar({ subastaId: 'SUB-VACIA', postorGanadorId: 'USR-1', montoFinalAdjudicado: 5000 });
    expect(liqVacia.exito).toBe(false);

    // 5. LiquidarSubastaCasoUso: ganador capturado + perdedor activo liberado + perdedor ya liberado omitido
    await ucRetener.ejecutar({ postorId: 'USR-LOSER-1', subastaId: 'SUB-2', precioBaseSubasta: 10000 });
    const gInactiva = new Garantia('GAR-INACT', 'SUB-2', 'USR-LOSER-INACT', 1000, 'AUTH-X', EstadoGarantia.LIBERADA, 80);
    await repoGar.guardar(gInactiva);

    const liqOk = await ucLiquidar.ejecutar({ subastaId: 'SUB-2', postorGanadorId: 'USR-VIP', montoFinalAdjudicado: 12000 });
    expect(liqOk.exito).toBe(true);
    expect(liqOk.garantiaGanador?.estado).toBe(EstadoGarantia.CAPTURADA_COBRO);
    expect(liqOk.garantiasLiberadasCount).toBe(1);
  });
});
