export const EstadoSubasta = {
  PROGRAMADA: 'PROGRAMADA',
  ACTIVA: 'ACTIVA',
  EN_CONTEO_MARTILLO: 'EN_CONTEO_MARTILLO',
  ADJUDICADA: 'ADJUDICADA',
  DESIERTA: 'DESIERTA',
  CANCELADA: 'CANCELADA'
} as const;

export type EstadoSubasta = typeof EstadoSubasta[keyof typeof EstadoSubasta];
