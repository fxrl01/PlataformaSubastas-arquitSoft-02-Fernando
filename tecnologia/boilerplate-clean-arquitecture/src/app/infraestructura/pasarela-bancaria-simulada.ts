import type { PasarelaBancariaContrato, ResultadoRetencionBancaria } from '../dominio/contratos/pasarela-bancaria.contrato.ts';

export class PasarelaBancariaSimulada implements PasarelaBancariaContrato {
  private readonly retenciones: Map<string, { postorId: string; monto: number; activa: boolean }> = new Map();

  public async retener(postorId: string, monto: number): Promise<ResultadoRetencionBancaria> {
    if (postorId.includes('SIN_FONDOS')) {
      return {
        exito: false,
        autorizacionId: '',
        mensaje: 'Fondos insuficientes en la tarjeta para constituir la garantía'
      };
    }

    const autorizacionId = `AUTH-BANK-${Date.now()}-${Math.floor(Math.random() * 1000)}`;
    this.retenciones.set(autorizacionId, { postorId, monto, activa: true });
    
    return {
      exito: true,
      autorizacionId,
      mensaje: 'Preautorización bancaria completada exitosamente'
    };
  }

  public async capturar(autorizacionId: string): Promise<boolean> {
    const item = this.retenciones.get(autorizacionId);
    if (!item || !item.activa) return false;
    item.activa = false;
    return true;
  }

  public async liberar(autorizacionId: string): Promise<boolean> {
    const item = this.retenciones.get(autorizacionId);
    if (!item) return false;
    item.activa = false;
    return true;
  }
}
