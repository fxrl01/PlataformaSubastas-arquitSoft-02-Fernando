import type { AgenteRiesgoContrato, EvaluacionRiesgoIA } from '../dominio/contratos/agente-riesgo.contrato.ts';
import type { PasarelaBancariaContrato } from '../dominio/contratos/pasarela-bancaria.contrato.ts';
import type { CustodiaGarantiasContrato } from '../dominio/contratos/custodia-garantias.contrato.ts';
import { Garantia, EstadoGarantia } from '../dominio/modelos/garantia.modelo.ts';

export interface EntradaRetenerGarantia {
  postorId: string;
  subastaId: string;
  precioBaseSubasta: number;
}

export interface SalidaRetenerGarantia {
  exito: boolean;
  garantia?: Garantia;
  evaluacionIA?: EvaluacionRiesgoIA;
  mensaje: string;
}

export class RetenerGarantiaCasoUso {
  private readonly agenteRiesgo: AgenteRiesgoContrato;
  private readonly pasarelaBancaria: PasarelaBancariaContrato;
  private readonly custodiaRepo: CustodiaGarantiasContrato;

  constructor(
    agenteRiesgo: AgenteRiesgoContrato,
    pasarelaBancaria: PasarelaBancariaContrato,
    custodiaRepo: CustodiaGarantiasContrato
  ) {
    this.agenteRiesgo = agenteRiesgo;
    this.pasarelaBancaria = pasarelaBancaria;
    this.custodiaRepo = custodiaRepo;
  }

  public async ejecutar(entrada: EntradaRetenerGarantia): Promise<SalidaRetenerGarantia> {
    // 1. Evitar doble retención si ya tiene garantía activa
    const garantiaExistente = await this.custodiaRepo.obtenerGarantia(
      entrada.postorId,
      entrada.subastaId
    );
    if (garantiaExistente && garantiaExistente.esActiva()) {
      return {
        exito: false,
        mensaje: 'El postor ya cuenta con una garantía activa para esta subasta'
      };
    }

    // 2. Evaluación de riesgo mediante Agente de IA (con Circuit Breaker)
    const evaluacion = await this.agenteRiesgo.evaluarPostor(
      entrada.postorId,
      entrada.subastaId
    );

    // 3. Cálculo dinámico de la garantía según perfil de riesgo
    const montoGarantia = entrada.precioBaseSubasta * evaluacion.porcentajeGarantia;

    // 4. Preautorización bancaria de fondos en la pasarela
    const retencionBancaria = await this.pasarelaBancaria.retener(entrada.postorId, montoGarantia);
    if (!retencionBancaria.exito) {
      return {
        exito: false,
        evaluacionIA: evaluacion,
        mensaje: retencionBancaria.mensaje || 'Error al procesar la retención bancaria'
      };
    }

    // 5. Creación y persistencia de la entidad Garantia
    const nuevaGarantia = new Garantia(
      `GAR-${Date.now()}-${Math.floor(Math.random() * 1000)}`,
      entrada.subastaId,
      entrada.postorId,
      montoGarantia,
      retencionBancaria.autorizacionId,
      EstadoGarantia.RETENIDA,
      evaluacion.score
    );

    await this.custodiaRepo.guardar(nuevaGarantia);

    return {
      exito: true,
      garantia: nuevaGarantia,
      evaluacionIA: evaluacion,
      mensaje: `Garantía constituida exitosamente (Score IA: ${evaluacion.score}, Garantía: S/. ${montoGarantia})`
    };
  }
}
