export interface MotorPujasContrato {
  adquirirLock(subastaId: string): Promise<boolean>;
  liberarLock(subastaId: string): Promise<void>;
}
