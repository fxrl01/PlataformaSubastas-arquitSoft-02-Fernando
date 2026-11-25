import { describe, expect, it } from '@jest/globals';
import { Subasta } from '../dominio/modelos/subasta.modelo.ts';
import { EstadoSubasta } from '../dominio/modelos/estado-subasta.enum.ts';
import { CodigoErrorPuja, ErrorDominioPuja } from '../dominio/modelos/errores-puja.ts';
import { Garantia, EstadoGarantia } from '../dominio/modelos/garantia.modelo.ts';
import { NivelRiesgo } from '../dominio/contratos/agente-riesgo.contrato.ts';

describe('Dominio — Subasta, Garantia y Errores', () => {
  it('valida todas las reglas del ciclo de pujas y Anti-Sniping en Subasta', () => {
    // Constructor con valores por defecto y sin postorLiderId
    const subasta = new Subasta(
      'SUB-1',
      'Toyota Hilux',
      45000,
      1000,
      45000,
      EstadoSubasta.ACTIVA,
      new Date(Date.now() + 120_000)
    );
    expect(subasta.pujas).toEqual([]);
    expect(subasta.postorLiderId).toBeUndefined();

    // Puja válida normal (fuera de ventana Anti-Sniping > 15s)
    const r1 = subasta.validarYRegistrarPuja('USR-1', 46000);
    expect(r1.antiSnipingActivado).toBe(false);
    expect(subasta.precioActual).toBe(46000);
    expect(subasta.postorLiderId).toBe('USR-1');

    // Prohibición de autopuja
    expect(() => subasta.validarYRegistrarPuja('USR-1', 48000)).toThrow(ErrorDominioPuja);
    try {
      subasta.validarYRegistrarPuja('USR-1', 48000);
    } catch (e) {
      expect((e as ErrorDominioPuja).codigo).toBe(CodigoErrorPuja.POSTOR_YA_ES_LIDER);
    }

    // Monto insuficiente
    try {
      subasta.validarYRegistrarPuja('USR-2', 46500);
      throw new Error('Debió lanzar MONTO_INSUFICIENTE');
    } catch (e) {
      expect((e as ErrorDominioPuja).codigo).toBe(CodigoErrorPuja.MONTO_INSUFICIENTE);
    }

    // Acepta puja en EN_CONTEO_MARTILLO y activa Anti-Sniping (< 15s restantes)
    subasta.estado = EstadoSubasta.EN_CONTEO_MARTILLO;
    subasta.finProgramado = new Date(Date.now() + 8_000);
    const finAntes = subasta.finProgramado.getTime();
    const r2 = subasta.validarYRegistrarPuja('USR-2', 47000);
    expect(r2.antiSnipingActivado).toBe(true);
    expect(r2.nuevoFinProgramado.getTime()).toBe(finAntes + 30_000);
    expect(subasta.estado).toBe(EstadoSubasta.ACTIVA);

    // Rechaza pujas cuando la subasta no está activa ni en conteo
    const cerrada = new Subasta(
      'SUB-2',
      'Lote Cerrado',
      1000,
      100,
      1000,
      EstadoSubasta.ADJUDICADA,
      new Date(Date.now() + 60_000),
      [],
      'USR-X'
    );
    try {
      cerrada.validarYRegistrarPuja('USR-3', 1200);
      throw new Error('Debió lanzar SUBASTA_CERRADA');
    } catch (e) {
      expect((e as ErrorDominioPuja).codigo).toBe(CodigoErrorPuja.SUBASTA_CERRADA);
    }
  });

  it('valida las transiciones de estado y errores de Garantia', () => {
    const g1 = new Garantia('GAR-1', 'SUB-1', 'USR-1', 4500, 'AUTH-1', EstadoGarantia.RETENIDA, 92);
    expect(g1.esActiva()).toBe(true);
    expect(g1.fechaRetencion).toBeInstanceOf(Date);
    g1.capturar();
    expect(g1.estado).toBe(EstadoGarantia.CAPTURADA_COBRO);
    expect(g1.esActiva()).toBe(false);
    expect(() => g1.capturar()).toThrow('No se puede capturar una garantía en estado CAPTURADA_COBRO');

    const fechaFija = new Date('2026-11-23T12:00:00Z');
    const g2 = new Garantia('GAR-2', 'SUB-1', 'USR-2', 4500, 'AUTH-2', EstadoGarantia.RETENIDA, 80, fechaFija);
    expect(g2.fechaRetencion).toBe(fechaFija);
    g2.liberar();
    expect(g2.estado).toBe(EstadoGarantia.LIBERADA);
    expect(() => g2.liberar()).toThrow('No se puede liberar una garantía en estado LIBERADA');

    expect(NivelRiesgo.BAJO).toBe('BAJO');
    expect(NivelRiesgo.MEDIO).toBe('MEDIO');
    expect(NivelRiesgo.ALTO).toBe('ALTO');
  });
});
