import type { RepositorioSubastasContrato } from '../dominio/contratos/repositorio-subastas.contrato.ts';
import type { MotorPujasContrato } from '../dominio/contratos/motor-pujas.contrato.ts';
import type { ProcesadorPagosCustodiaContrato } from '../dominio/contratos/procesador-pagos-custodia.contrato.ts';
import type { NotificadorSalaContrato } from '../dominio/contratos/notificador-sala.contrato.ts';
import type { Puja } from '../dominio/modelos/puja.modelo.ts';

export interface EntradaProcesarPuja {
  subastaId: string;
  postorId: string;
  monto: number;
}

export interface SalidaProcesarPuja {
  exito: boolean;
  puja?: Puja;
  mensaje: string;
  precioActual?: number;
}

export class ProcesarPujaCasoUso {
  private readonly repositorioSubastas: RepositorioSubastasContrato;
  private readonly motorPujas: MotorPujasContrato;
  private readonly procesadorPagos: ProcesadorPagosCustodiaContrato;
  private readonly notificadorSala: NotificadorSalaContrato;

  constructor(
    repositorioSubastas: RepositorioSubastasContrato,
    motorPujas: MotorPujasContrato,
    procesadorPagos: ProcesadorPagosCustodiaContrato,
    notificadorSala: NotificadorSalaContrato
  ) {
    this.repositorioSubastas = repositorioSubastas;
    this.motorPujas = motorPujas;
    this.procesadorPagos = procesadorPagos;
    this.notificadorSala = notificadorSala;
  }

  public async ejecutar(entrada: EntradaProcesarPuja): Promise<SalidaProcesarPuja> {
    const tieneGarantia = await this.procesadorPagos.validarGarantiaActiva(
      entrada.postorId,
      entrada.subastaId
    );
    if (!tieneGarantia) {
      return {
        exito: false,
        mensaje: 'El postor no cuenta con depósito de garantía activo para esta subasta'
      };
    }

    const lockAdquirido = await this.motorPujas.adquirirLock(entrada.subastaId);
    if (!lockAdquirido) {
      return {
        exito: false,
        mensaje: 'Colisión de pujas detectada. Por favor, reintente en unos instantes'
      };
    }

    try {
      const subasta = await this.repositorioSubastas.obtenerPorId(entrada.subastaId);
      if (!subasta) {
        return {
          exito: false,
          mensaje: `La subasta ${entrada.subastaId} no existe`
        };
      }

      const nuevaPuja = subasta.validarYRegistrarPuja(entrada.postorId, entrada.monto);
      await this.repositorioSubastas.guardar(subasta);
      await this.notificadorSala.difundirNuevaPuja(nuevaPuja);

      return {
        exito: true,
        puja: nuevaPuja,
        precioActual: subasta.precioActual,
        mensaje: 'Puja registrada exitosamente'
      };
    } catch (error: any) {
      return {
        exito: false,
        mensaje: error?.message || 'Error al procesar la puja'
      };
    } finally {
      await this.motorPujas.liberarLock(entrada.subastaId);
    }
  }
}
