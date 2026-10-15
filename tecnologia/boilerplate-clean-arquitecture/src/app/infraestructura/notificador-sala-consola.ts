import type { NotificadorSalaContrato } from '../dominio/contratos/notificador-sala.contrato.ts';
import type { Puja } from '../dominio/modelos/puja.modelo.ts';

export class NotificadorSalaConsola implements NotificadorSalaContrato {
  public async difundirNuevaPuja(puja: Puja): Promise<void> {
    console.log(
      `[BROADCAST SALA ${puja.subastaId}] 📢 ¡Nueva puja! S/. ${puja.monto} por el Postor ${puja.postorId} (#${puja.correlativo})`
    );
  }

  public async difundirFinSubasta(subastaId: string, ganadorId?: string, montoFinal?: number): Promise<void> {
    console.log(
      `[BROADCAST SALA ${subastaId}] 🔨 ¡MARTILLAZO FINAL! Adjudicada al postor ${ganadorId} por S/. ${montoFinal}`
    );
  }
}
