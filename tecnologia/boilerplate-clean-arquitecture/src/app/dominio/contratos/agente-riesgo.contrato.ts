export const NivelRiesgo = {
  BAJO: 'BAJO',
  MEDIO: 'MEDIO',
  ALTO: 'ALTO'
} as const;

export type NivelRiesgo = typeof NivelRiesgo[keyof typeof NivelRiesgo];

export interface EvaluacionRiesgoIA {
  score: number;             // 0 a 100
  nivelRiesgo: NivelRiesgo;
  porcentajeGarantia: number; // 0.10 ó 0.20
  justificacion: string;
  modoFallbackActivado: boolean;
}

export interface AgenteRiesgoContrato {
  evaluarPostor(postorId: string, subastaId: string): Promise<EvaluacionRiesgoIA>;
}
