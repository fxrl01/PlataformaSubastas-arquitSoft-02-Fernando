import type { RepositorioSubastasContrato } from '../dominio/contratos/repositorio-subastas.contrato.ts';
import type { MotorPujasContrato } from '../dominio/contratos/motor-pujas.contrato.ts';
import type { ProcesadorPagosCustodiaContrato } from '../dominio/contratos/procesador-pagos-custodia.contrato.ts';
import type { NotificadorSalaContrato } from '../dominio/contratos/notificador-sala.contrato.ts';
import type { Puja } from '../dominio/modelos/puja.modelo.ts';
import { CodigoErrorPuja, ErrorDominioPuja } from '../dominio/modelos/errores-puja.ts';

export interface EntradaProcesarPuja {
  subastaId: string;
  postorId: string;
  monto: number;
}

export interface SalidaProcesarPuja {
  exito: boolean;
  puja?: Puja;
  codigoError?: CodigoErrorPuja;
  mensaje: string;
  precioActual?: number;
  postorLiderId?: string;
  antiSnipingActivado?: boolean;
  nuevoFinProgramado?: Date;
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
    // 1. Verificación financiera Zero Trust (Garantía en custodia previa)
    const tieneGarantia = await this.procesadorPagos.validarGarantiaActiva(
      entrada.postorId,
      entrada.subastaId
    );
    if (!tieneGarantia) {
      return {
        exito: false,
        codigoError: CodigoErrorPuja.GARANTIA_REQUERIDA,
        mensaje: 'El postor no cuenta con depósito de garantía activo para esta subasta'
      };
    }

    // 2. Control de concurrencia atómico distribuido (Anti-Race Condition)
    const lockAdquirido = await this.motorPujas.adquirirLock(entrada.subastaId);
    if (!lockAdquirido) {
      return {
        exito: false,
        codigoError: CodigoErrorPuja.COLISION_CONCURRENTE,
        mensaje: 'Colisión de pujas concurrentes detectada. Por favor, reintente en unos instantes'
      };
    }

    try {
      // 3. Obtener estado actual del agregado Subasta
      const subasta = await this.repositorioSubastas.obtenerPorId(entrada.subastaId);
      if (!subasta) {
        return {
          exito: false,
          codigoError: CodigoErrorPuja.SUBASTA_CERRADA,
          mensaje: `La subasta ${entrada.subastaId} no existe`
        };
      }

      // 4. Evaluar reglas de negocio en el dominio (Anti-Sniping, incremento, líder)
      const resultado = subasta.validarYRegistrarPuja(entrada.postorId, entrada.monto);

      // 5. Persistir estado del agregado
      await this.repositorioSubastas.guardar(subasta);

      // 6. Difundir evento de nueva puja líder a la sala en tiempo real
      await this.notificadorSala.difundirNuevaPuja(resultado.puja, subasta, resultado.antiSnipingActivado);

      if (resultado.antiSnipingActivado) {
        await this.notificadorSala.difundirTiempoExtendido(subasta.id, resultado.nuevoFinProgramado);
      }

      return {
        exito: true,
        puja: resultado.puja,
        precioActual: subasta.precioActual,
        postorLiderId: subasta.postorLiderId,
        antiSnipingActivado: resultado.antiSnipingActivado,
        nuevoFinProgramado: resultado.nuevoFinProgramado,
        mensaje: 'Puja registrada exitosamente'
      };
    } catch (error: any) {
      if (error instanceof ErrorDominioPuja) {
        return {
          exito: false,
          codigoError: error.codigo,
          mensaje: error.message
        };
      }
      return {
        exito: false,
        mensaje: error?.message || 'Error al procesar la puja'
      };
    } finally {
      // 7. Liberar lock atómico garantizado
      await this.motorPujas.liberarLock(entrada.subastaId);
    }
  }
}
