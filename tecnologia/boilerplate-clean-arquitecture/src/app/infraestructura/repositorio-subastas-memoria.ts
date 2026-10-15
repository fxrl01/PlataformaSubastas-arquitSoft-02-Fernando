import type { RepositorioSubastasContrato } from '../dominio/contratos/repositorio-subastas.contrato.ts';
import type { Subasta } from '../dominio/modelos/subasta.modelo.ts';

export class RepositorioSubastasMemoria implements RepositorioSubastasContrato {
  private readonly subastas: Map<string, Subasta> = new Map();

  public async obtenerPorId(id: string): Promise<Subasta | null> {
    const subasta = this.subastas.get(id);
    return subasta || null;
  }

  public async guardar(subasta: Subasta): Promise<void> {
    this.subastas.set(subasta.id, subasta);
  }
}
