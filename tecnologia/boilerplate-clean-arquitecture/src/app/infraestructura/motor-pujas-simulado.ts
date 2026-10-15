import type { MotorPujasContrato } from '../dominio/contratos/motor-pujas.contrato.ts';

export class MotorPujasSimulado implements MotorPujasContrato {
  private readonly locks: Set<string> = new Set();

  public async adquirirLock(subastaId: string): Promise<boolean> {
    if (this.locks.has(subastaId)) {
      return false;
    }
    this.locks.add(subastaId);
    return true;
  }

  public async liberarLock(subastaId: string): Promise<void> {
    this.locks.delete(subastaId);
  }
}
