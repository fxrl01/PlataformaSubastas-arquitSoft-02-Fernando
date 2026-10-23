import { ProcesarPujaCasoUso } from '../aplicacion/procesar-puja.caso-uso.ts';
import { CodigoErrorPuja } from '../dominio/modelos/errores-puja.ts';

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
      let status = 409;
      if (resultado.codigoError === CodigoErrorPuja.GARANTIA_REQUERIDA) status = 403;
      if (resultado.codigoError === CodigoErrorPuja.COLISION_CONCURRENTE) status = 503;
      if (resultado.codigoError === CodigoErrorPuja.POSTOR_YA_ES_LIDER) status = 400;
      if (resultado.codigoError === CodigoErrorPuja.SUBASTA_CERRADA) status = 400;

      return {
        status,
        codigoError: resultado.codigoError,
        error: resultado.mensaje
      };
    }

    return { status: 200, data: resultado };
  }
}
