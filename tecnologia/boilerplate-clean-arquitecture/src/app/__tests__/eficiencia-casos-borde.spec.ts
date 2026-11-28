import { describe, expect, it, jest } from '@jest/globals';
import { Subasta } from '../dominio/modelos/subasta.modelo.ts';
import { EstadoSubasta } from '../dominio/modelos/estado-subasta.enum.ts';
import { CodigoErrorPuja } from '../dominio/modelos/errores-puja.ts';
import { EstadoGarantia } from '../dominio/modelos/garantia.modelo.ts';
import { ProcesarPujaCasoUso } from '../aplicacion/procesar-puja.caso-uso.ts';
import { LiquidarSubastaCasoUso } from '../aplicacion/liquidar-subasta.caso-uso.ts';
import { RetenerGarantiaCasoUso } from '../aplicacion/retener-garantia.caso-uso.ts';
import { RepositorioSubastasMemoria } from '../infraestructura/repositorio-subastas-memoria.ts';
import { RepositorioGarantiasMemoria } from '../infraestructura/repositorio-garantias-memoria.ts';
import { MotorPujasSimulado } from '../infraestructura/motor-pujas-simulado.ts';
import { NotificadorSalaConsola } from '../infraestructura/notificador-sala-consola.ts';
import { PasarelaBancariaSimulada } from '../infraestructura/pasarela-bancaria-simulada.ts';
import { ProcesadorPagosSimulado } from '../infraestructura/procesador-pagos-simulado.ts';
import { CircuitBreakerIA } from '../infraestructura/circuit-breaker-ia.ts';
import { GeminiRiesgoAdapter } from '../infraestructura/gemini-riesgo.adapter.ts';

describe('Eficiencia y Casos Borde de Alta Frecuencia — Clean Architecture', () => {
  it('procesa 100 pujas secuenciales alternadas manteniendo consistencia exacta y baja latencia en memoria', () => {
    const finOriginal = new Date(Date.now() + 300_000);
    const subasta = new Subasta(
      'SUB-HIGH-FREQ',
      'Colección Numismática Oro',
      10_000,
      500,
      10_000,
      EstadoSubasta.ACTIVA,
      finOriginal
    );

    const t0 = performance.now();
    let montoEsperado = 10_000;
    for (let i = 1; i <= 100; i++) {
      const postor = i % 2 === 0 ? 'POSTOR-PAR' : 'POSTOR-IMPAR';
      montoEsperado += 500;
      const res = subasta.validarYRegistrarPuja(postor, montoEsperado);
      expect(res.antiSnipingActivado).toBe(false);
    }
    const elapsedMs = performance.now() - t0;

    expect(subasta.pujas).toHaveLength(100);
    expect(subasta.precioActual).toBe(60_000);
    expect(subasta.postorLiderId).toBe('POSTOR-PAR');
    expect(elapsedMs).toBeLessThan(100);
  });

  it('verifica la frontera de Anti-Sniping y la liquidación justa entre 3 postores con garantía activa', async () => {
    jest.spyOn(console, 'log').mockImplementation(() => {});
    const subastasRepo = new RepositorioSubastasMemoria();
    const garantiasRepo = new RepositorioGarantiasMemoria();
    const motorLock = new MotorPujasSimulado();
    const notificador = new NotificadorSalaConsola();
    const pasarela = new PasarelaBancariaSimulada();
    const pagos = new ProcesadorPagosSimulado();
    const ia = new CircuitBreakerIA(new GeminiRiesgoAdapter(), 3, 2000, 150);

    const subasta = new Subasta(
      'SUB-MULTI-SETTLE',
      'Departamento Miraflores',
      100_000,
      2_000,
      100_000,
      EstadoSubasta.ACTIVA,
      new Date(Date.now() + 12_000)
    );
    await subastasRepo.guardar(subasta);

    const retenerUC = new RetenerGarantiaCasoUso(ia, pasarela, garantiasRepo);
    const pujarUC = new ProcesarPujaCasoUso(subastasRepo, motorLock, pagos, notificador);
    const liquidarUC = new LiquidarSubastaCasoUso(pasarela, garantiasRepo);

    // 3 postores constituyen su garantía reembolsable
    const rA = await retenerUC.ejecutar({ postorId: 'USR-A', subastaId: 'SUB-MULTI-SETTLE', precioBaseSubasta: 100_000 });
    const rB = await retenerUC.ejecutar({ postorId: 'USR-B', subastaId: 'SUB-MULTI-SETTLE', precioBaseSubasta: 100_000 });
    const rC = await retenerUC.ejecutar({ postorId: 'USR-C', subastaId: 'SUB-MULTI-SETTLE', precioBaseSubasta: 100_000 });

    expect(rA.exito && rB.exito && rC.exito).toBe(true);
    pagos.habilitarGarantia('USR-A', 'SUB-MULTI-SETTLE');
    pagos.habilitarGarantia('USR-B', 'SUB-MULTI-SETTLE');
    pagos.habilitarGarantia('USR-C', 'SUB-MULTI-SETTLE');

    // Intruso sin garantía es rechazado con GARANTIA_REQUERIDA
    const rechazoIntruso = await pujarUC.ejecutar({ subastaId: 'SUB-MULTI-SETTLE', postorId: 'USR-INTRUSO', monto: 105_000 });
    expect(rechazoIntruso.exito).toBe(false);
    expect(rechazoIntruso.codigoError).toBe(CodigoErrorPuja.GARANTIA_REQUERIDA);

    // USR-A oferta en ventana crítica (<15s) -> activa Anti-Sniping (+30s)
    const puja1 = await pujarUC.ejecutar({ subastaId: 'SUB-MULTI-SETTLE', postorId: 'USR-A', monto: 102_000 });
    expect(puja1.exito).toBe(true);
    expect(puja1.antiSnipingActivado).toBe(true);

    // USR-B supera la oferta
    const puja2 = await pujarUC.ejecutar({ subastaId: 'SUB-MULTI-SETTLE', postorId: 'USR-B', monto: 106_000 });
    expect(puja2.exito).toBe(true);
    expect(puja2.precioActual).toBe(106_000);

    // Al liquidar, SOLO se captura la garantía de USR-B y se liberan USR-A y USR-C (2 liberadas)
    const liquidacion = await liquidarUC.ejecutar({
      subastaId: 'SUB-MULTI-SETTLE',
      postorGanadorId: 'USR-B',
      montoFinalAdjudicado: 106_000,
    });
    expect(liquidacion.exito).toBe(true);
    expect(liquidacion.garantiaGanador?.estado).toBe(EstadoGarantia.CAPTURADA_COBRO);
    expect(liquidacion.garantiasLiberadasCount).toBe(2);
  });
});
