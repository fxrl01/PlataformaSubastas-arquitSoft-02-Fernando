import type { PasarelaBancariaContrato } from '../dominio/contratos/pasarela-bancaria.contrato.ts';
import type { CustodiaGarantiasContrato } from '../dominio/contratos/custodia-garantias.contrato.ts';
import { Garantia } from '../dominio/modelos/garantia.modelo.ts';

export interface EntradaLiquidarSubasta {
  subastaId: string;
  postorGanadorId: string;
  montoFinalAdjudicado: number;
}

export interface SalidaLiquidarSubasta {
  exito: boolean;
  garantiaGanador?: Garantia;
  garantiasLiberadasCount: number;
  mensaje: string;
}

export class LiquidarSubastaCasoUso {
  private readonly pasarelaBancaria: PasarelaBancariaContrato;
  private readonly custodiaRepo: CustodiaGarantiasContrato;

  constructor(
    pasarelaBancaria: PasarelaBancariaContrato,
    custodiaRepo: CustodiaGarantiasContrato
  ) {
    this.pasarelaBancaria = pasarelaBancaria;
    this.custodiaRepo = custodiaRepo;
  }

  public async ejecutar(entrada: EntradaLiquidarSubasta): Promise<SalidaLiquidarSubasta> {
    const garantias = await this.custodiaRepo.obtenerPorSubasta(entrada.subastaId);
    if (garantias.length === 0) {
      return {
        exito: false,
        garantiasLiberadasCount: 0,
        mensaje: 'No se encontraron garantías registradas para liquidar esta subasta'
      };
    }

    let garantiaGanador: Garantia | undefined;
    let garantiasLiberadasCount = 0;

    for (const garantia of garantias) {
      if (garantia.postorId === entrada.postorGanadorId) {
        // 1. Capturar garantía del ganador como abono/liquidación
        garantia.capturar();
        await this.pasarelaBancaria.capturar(garantia.autorizacionBancariaId);
        await this.custodiaRepo.guardar(garantia);
        garantiaGanador = garantia;
      } else if (garantia.esActiva()) {
        // 2. Liberación en masa inmediata a todos los postores perdedores
        garantia.liberar();
        await this.pasarelaBancaria.liberar(garantia.autorizacionBancariaId);
        await this.custodiaRepo.guardar(garantia);
        garantiasLiberadasCount++;
      }
    }

    return {
      exito: true,
      garantiaGanador,
      garantiasLiberadasCount,
      mensaje: `Subasta liquidada con éxito: garantía del ganador capturada y ${garantiasLiberadasCount} garantías liberadas a postores perdedores`
    };
  }
}
