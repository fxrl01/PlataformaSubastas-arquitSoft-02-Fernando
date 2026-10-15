import type { ProcesadorPagosCustodiaContrato } from '../dominio/contratos/procesador-pagos-custodia.contrato.ts';

export class ProcesadorPagosSimulado implements ProcesadorPagosCustodiaContrato {
  private readonly garantiasValidas: Set<string> = new Set();

  public habilitarGarantia(postorId: string, subastaId: string): void {
    this.garantiasValidas.add(`${postorId}:${subastaId}`);
  }

  public async validarGarantiaActiva(postorId: string, subastaId: string): Promise<boolean> {
    return this.garantiasValidas.has(`${postorId}:${subastaId}`);
  }

  public async retenerGarantia(postorId: string, subastaId: string, monto: number): Promise<boolean> {
    this.habilitarGarantia(postorId, subastaId);
    return true;
  }

  public async liberarGarantia(postorId: string, subastaId: string): Promise<void> {
    this.garantiasValidas.delete(`${postorId}:${subastaId}`);
  }
}
