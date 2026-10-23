import type { Puja } from '../modelos/puja.modelo.ts';
import type { Subasta } from '../modelos/subasta.modelo.ts';

export interface NotificadorSalaContrato {
  difundirNuevaPuja(puja: Puja, subasta: Subasta, antiSnipingActivado: boolean): Promise<void>;
  difundirTiempoExtendido(subastaId: string, nuevoFinProgramado: Date): Promise<void>;
  difundirFinSubasta(subastaId: string, ganadorId?: string, montoFinal?: number): Promise<void>;
}
