import { EstadoSubasta } from './estado-subasta.enum.ts';
import type { Puja } from './puja.modelo.ts';

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

  public validarYRegistrarPuja(postorId: string, montoOfertado: number): Puja {
    if (this.estado !== EstadoSubasta.ACTIVA && this.estado !== EstadoSubasta.EN_CONTEO_MARTILLO) {
      throw new Error(`No se puede pujar en una subasta con estado ${this.estado}`);
    }

    const montoMinimoRequerido = this.precioActual + this.incrementoMinimo;
    if (montoOfertado < montoMinimoRequerido) {
      throw new Error(
        `El monto ofertado (${montoOfertado}) debe ser al menos el precio actual (${this.precioActual}) más el incremento mínimo (${this.incrementoMinimo})`
      );
    }

    const ahora = new Date();
    const tiempoRestanteMs = this.finProgramado.getTime() - ahora.getTime();
    if (tiempoRestanteMs > 0 && tiempoRestanteMs < 15000) {
      this.finProgramado = new Date(this.finProgramado.getTime() + 30000);
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

    return nuevaPuja;
  }
}
