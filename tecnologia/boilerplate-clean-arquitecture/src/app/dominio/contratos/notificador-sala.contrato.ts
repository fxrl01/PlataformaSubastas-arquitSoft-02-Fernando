import type { Puja } from '../modelos/puja.modelo.ts';

export interface NotificadorSalaContrato {
  difundirNuevaPuja(puja: Puja): Promise<void>;
  difundirFinSubasta(subastaId: string, ganadorId?: string, montoFinal?: number): Promise<void>;
}
