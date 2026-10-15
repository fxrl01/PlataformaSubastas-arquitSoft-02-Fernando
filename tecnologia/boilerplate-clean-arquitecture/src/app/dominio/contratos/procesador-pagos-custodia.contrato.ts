export interface ProcesadorPagosCustodiaContrato {
  validarGarantiaActiva(postorId: string, subastaId: string): Promise<boolean>;
  retenerGarantia(postorId: string, subastaId: string, monto: number): Promise<boolean>;
  liberarGarantia(postorId: string, subastaId: string): Promise<void>;
}
