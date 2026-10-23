import type { Garantia } from '../modelos/garantia.modelo.ts';

export interface CustodiaGarantiasContrato {
  obtenerGarantia(postorId: string, subastaId: string, soloActiva?: boolean): Promise<Garantia | null>;
  obtenerPorSubasta(subastaId: string): Promise<Garantia[]>;
  guardar(garantia: Garantia): Promise<void>;
}
