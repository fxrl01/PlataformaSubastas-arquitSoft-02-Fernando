import type { NotificadorSalaContrato } from '../dominio/contratos/notificador-sala.contrato.ts';
import type { Puja } from '../dominio/modelos/puja.modelo.ts';
import type { Subasta } from '../dominio/modelos/subasta.modelo.ts';

export class NotificadorSalaConsola implements NotificadorSalaContrato {
  public async difundirNuevaPuja(puja: Puja, subasta: Subasta, antiSnipingActivado: boolean): Promise<void> {
    const alertaSniping = antiSnipingActivado ? ' ⏳ [ANTI-SNIPING ACTIVADO: +30s]' : '';
    console.log(
      `[BROADCAST SALA ${puja.subastaId}] 📢 ¡Nueva puja líder! S/. ${puja.monto} por Postor ${puja.postorId} (#${puja.correlativo})${alertaSniping}`
    );
  }

  public async difundirTiempoExtendido(subastaId: string, nuevoFinProgramado: Date): Promise<void> {
    console.log(
      `[BROADCAST SALA ${subastaId}] ⏱️ Cronómetro extendido hasta: ${nuevoFinProgramado.toLocaleTimeString()}`
    );
  }

  public async difundirFinSubasta(subastaId: string, ganadorId?: string, montoFinal?: number): Promise<void> {
    console.log(
      `[BROADCAST SALA ${subastaId}] 🔨 ¡MARTILLAZO FINAL! Adjudicada al postor ${ganadorId} por S/. ${montoFinal}`
    );
  }
}
