export const EstadoGarantia = {
  RETENIDA: 'RETENIDA',
  CAPTURADA_COBRO: 'CAPTURADA_COBRO',
  LIBERADA: 'LIBERADA'
} as const;

export type EstadoGarantia = typeof EstadoGarantia[keyof typeof EstadoGarantia];

export class Garantia {
  public readonly id: string;
  public readonly subastaId: string;
  public readonly postorId: string;
  public readonly montoRetenido: number;
  public readonly autorizacionBancariaId: string;
  public estado: EstadoGarantia;
  public readonly scoreConfianzaIA: number;
  public readonly fechaRetencion: Date;

  constructor(
    id: string,
    subastaId: string,
    postorId: string,
    montoRetenido: number,
    autorizacionBancariaId: string,
    estado: EstadoGarantia,
    scoreConfianzaIA: number,
    fechaRetencion: Date = new Date()
  ) {
    this.id = id;
    this.subastaId = subastaId;
    this.postorId = postorId;
    this.montoRetenido = montoRetenido;
    this.autorizacionBancariaId = autorizacionBancariaId;
    this.estado = estado;
    this.scoreConfianzaIA = scoreConfianzaIA;
    this.fechaRetencion = fechaRetencion;
  }

  public capturar(): void {
    if (this.estado !== EstadoGarantia.RETENIDA) {
      throw new Error(`No se puede capturar una garantía en estado ${this.estado}`);
    }
    this.estado = EstadoGarantia.CAPTURADA_COBRO;
  }

  public liberar(): void {
    if (this.estado !== EstadoGarantia.RETENIDA) {
      throw new Error(`No se puede liberar una garantía en estado ${this.estado}`);
    }
    this.estado = EstadoGarantia.LIBERADA;
  }

  public esActiva(): boolean {
    return this.estado === EstadoGarantia.RETENIDA;
  }
}
