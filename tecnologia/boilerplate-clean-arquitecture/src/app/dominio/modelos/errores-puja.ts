export const CodigoErrorPuja = {
  MONTO_INSUFICIENTE: 'MONTO_INSUFICIENTE',
  SUBASTA_CERRADA: 'SUBASTA_CERRADA',
  GARANTIA_REQUERIDA: 'GARANTIA_REQUERIDA',
  POSTOR_YA_ES_LIDER: 'POSTOR_YA_ES_LIDER',
  COLISION_CONCURRENTE: 'COLISION_CONCURRENTE'
} as const;

export type CodigoErrorPuja = typeof CodigoErrorPuja[keyof typeof CodigoErrorPuja];

export class ErrorDominioPuja extends Error {
  public readonly codigo: CodigoErrorPuja;

  constructor(codigo: CodigoErrorPuja, mensaje: string) {
    super(mensaje);
    this.name = 'ErrorDominioPuja';
    this.codigo = codigo;
  }
}
