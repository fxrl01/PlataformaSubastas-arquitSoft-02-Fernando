import type { CustodiaGarantiasContrato } from '../dominio/contratos/custodia-garantias.contrato.ts';
import type { Garantia } from '../dominio/modelos/garantia.modelo.ts';

export class RepositorioGarantiasMemoria implements CustodiaGarantiasContrato {
  private readonly garantias: Map<string, Garantia> = new Map();

  public async obtenerGarantia(
    postorId: string,
    subastaId: string,
    soloActiva: boolean = false
  ): Promise<Garantia | null> {
    for (const g of this.garantias.values()) {
      if (g.postorId === postorId && g.subastaId === subastaId) {
        if (!soloActiva || g.esActiva()) {
          return g;
        }
      }
    }
    return null;
  }

  public async obtenerPorSubasta(subastaId: string): Promise<Garantia[]> {
    const lista: Garantia[] = [];
    for (const g of this.garantias.values()) {
      if (g.subastaId === subastaId) {
        lista.push(g);
      }
    }
    return lista;
  }

  public async guardar(garantia: Garantia): Promise<void> {
    this.garantias.set(garantia.id, garantia);
  }
}
