export interface ResultadoRetencionBancaria {
  exito: boolean;
  autorizacionId: string;
  mensaje?: string;
}

export interface PasarelaBancariaContrato {
  retener(postorId: string, monto: number): Promise<ResultadoRetencionBancaria>;
  capturar(autorizacionId: string): Promise<boolean>;
  liberar(autorizacionId: string): Promise<boolean>;
}
