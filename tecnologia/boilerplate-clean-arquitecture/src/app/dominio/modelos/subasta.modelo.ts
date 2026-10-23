import { EstadoSubasta } from './estado-subasta.enum.ts';
import type { Puja } from './puja.modelo.ts';
import { CodigoErrorPuja, ErrorDominioPuja } from './errores-puja.ts';

export interface ResultadoRegistroPuja {
  puja: Puja;
  antiSnipingActivado: boolean;
  nuevoFinProgramado: Date;
}

export class Subasta {
  public readonly id: string;
  public readonly titulo: string;
  public readonly precioBase: number;
  public readonly incrementoMinimo: number;
  public precioActual: number;
  public estado: EstadoSubasta;
  public finProgramado: Date;
  public pujas: Puja[];
  public postorLiderId?: string;

  constructor(
    id: string,
    titulo: string,
    precioBase: number,
    incrementoMinimo: number,
    precioActual: number,
    estado: EstadoSubasta,
    finProgramado: Date,
    pujas: Puja[] = [],
    postorLiderId?: string
  ) {
    this.id = id;
    this.titulo = titulo;
    this.precioBase = precioBase;
    this.incrementoMinimo = incrementoMinimo;
    this.precioActual = precioActual;
    this.estado = estado;
    this.finProgramado = finProgramado;
    this.pujas = pujas;
    this.postorLiderId = postorLiderId;
  }

  public validarYRegistrarPuja(postorId: string, montoOfertado: number): ResultadoRegistroPuja {
    // 1. Validar estado de la subasta
    if (this.estado !== EstadoSubasta.ACTIVA && this.estado !== EstadoSubasta.EN_CONTEO_MARTILLO) {
      throw new ErrorDominioPuja(
        CodigoErrorPuja.SUBASTA_CERRADA,
        `No se puede pujar en una subasta con estado ${this.estado}`
      );
    }

    // 2. Prohibición de autopuja (el postor líder no puede pujar sobre su propia oferta)
    if (this.postorLiderId === postorId) {
      throw new ErrorDominioPuja(
        CodigoErrorPuja.POSTOR_YA_ES_LIDER,
        `El postor ${postorId} ya es el líder actual de la subasta`
      );
    }

    // 3. Validar incremento mínimo requerido
    const montoMinimoRequerido = this.precioActual + this.incrementoMinimo;
    if (montoOfertado < montoMinimoRequerido) {
      throw new ErrorDominioPuja(
        CodigoErrorPuja.MONTO_INSUFICIENTE,
        `El monto ofertado (S/. ${montoOfertado}) debe ser al menos S/. ${montoMinimoRequerido} (precio actual S/. ${this.precioActual} + incremento mínimo S/. ${this.incrementoMinimo})`
      );
    }

    // 4. Regla Anti-Sniping: si entra una puja a falta de menos de 15 segundos, se extiende 30 segundos
    const ahora = new Date();
    const tiempoRestanteMs = this.finProgramado.getTime() - ahora.getTime();
    let antiSnipingActivado = false;

    if (tiempoRestanteMs > 0 && tiempoRestanteMs < 15000) {
      this.finProgramado = new Date(this.finProgramado.getTime() + 30000);
      antiSnipingActivado = true;
    }

    const nuevaPuja: Puja = {
      id: `PUJ-${Date.now()}-${this.pujas.length + 1}`,
      subastaId: this.id,
      postorId,
      monto: montoOfertado,
      timestamp: ahora,
      correlativo: this.pujas.length + 1
    };

    this.precioActual = montoOfertado;
    this.postorLiderId = postorId;
    this.pujas.push(nuevaPuja);
    this.estado = EstadoSubasta.ACTIVA;

    return {
      puja: nuevaPuja,
      antiSnipingActivado,
      nuevoFinProgramado: this.finProgramado
    };
  }
}
