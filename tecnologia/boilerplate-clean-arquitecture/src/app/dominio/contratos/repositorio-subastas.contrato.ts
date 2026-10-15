import type { Subasta } from '../modelos/subasta.modelo.ts';

export interface RepositorioSubastasContrato {
  obtenerPorId(id: string): Promise<Subasta | null>;
  guardar(subasta: Subasta): Promise<void>;
}
