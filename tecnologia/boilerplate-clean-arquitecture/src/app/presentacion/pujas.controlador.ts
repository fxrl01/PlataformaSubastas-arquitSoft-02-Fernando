import { ProcesarPujaCasoUso } from '../aplicacion/procesar-puja.caso-uso.ts';

export class PujasControlador {
  private readonly procesarPujaCasoUso: ProcesarPujaCasoUso;

  constructor(procesarPujaCasoUso: ProcesarPujaCasoUso) {
    this.procesarPujaCasoUso = procesarPujaCasoUso;
  }

  public async atenderEventoPuja(peticion: { subastaId: string; postorId: string; monto: number }) {
    if (!peticion.subastaId || !peticion.postorId || !peticion.monto) {
      return { status: 400, error: 'Parámetros incompletos para emitir puja' };
    }

    const resultado = await this.procesarPujaCasoUso.ejecutar({
      subastaId: peticion.subastaId,
      postorId: peticion.postorId,
      monto: peticion.monto
    });

    if (!resultado.exito) {
      return { status: 409, error: resultado.mensaje };
    }

    return { status: 200, data: resultado };
  }
}
