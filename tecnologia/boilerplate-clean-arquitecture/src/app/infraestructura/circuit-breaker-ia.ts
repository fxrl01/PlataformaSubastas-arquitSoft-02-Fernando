import type { AgenteRiesgoContrato, EvaluacionRiesgoIA } from '../dominio/contratos/agente-riesgo.contrato.ts';
import { NivelRiesgo } from '../dominio/contratos/agente-riesgo.contrato.ts';

export const EstadoCircuitBreaker = {
  CLOSED: 'CLOSED',       // Modo normal: llamadas fluyen a la IA
  OPEN: 'OPEN',           // Falla detectada: llamadas van al fallback local
  HALF_OPEN: 'HALF_OPEN'  // Prueba de recuperación
} as const;

export type EstadoCircuitBreaker = typeof EstadoCircuitBreaker[keyof typeof EstadoCircuitBreaker];

export class CircuitBreakerIA implements AgenteRiesgoContrato {
  private estado: EstadoCircuitBreaker = EstadoCircuitBreaker.CLOSED;
  private fallosConsecutivos: number = 0;
  private readonly umbralFallos: number = 2; // Tras 2 fallos consecutivos se abre
  private readonly tiempoEsperaReaperturaMs: number = 3000;
  private ultimoFalloTimestamp: number = 0;
  private readonly servicioIAReal: AgenteRiesgoContrato;

  constructor(servicioIAReal: AgenteRiesgoContrato) {
    this.servicioIAReal = servicioIAReal;
  }

  public getEstado(): EstadoCircuitBreaker {
    return this.estado;
  }

  public async evaluarPostor(postorId: string, subastaId: string): Promise<EvaluacionRiesgoIA> {
    const ahora = Date.now();

    // 1. Si está OPEN y ya pasó el tiempo de espera, pasa a HALF_OPEN para probar
    if (this.estado === EstadoCircuitBreaker.OPEN) {
      if (ahora - this.ultimoFalloTimestamp >= this.tiempoEsperaReaperturaMs) {
        this.estado = EstadoCircuitBreaker.HALF_OPEN;
        console.log('🛡️ [CIRCUIT BREAKER] Transición a HALF_OPEN: intentando llamada de prueba a Gemini...');
      } else {
        // Circuito abierto: aplicar fallback de inmediato sin llamar a la API
        console.log('🛡️ [CIRCUIT BREAKER OPEN] API de IA no disponible. Activando Fallback Heurístico Local...');
        return this.ejecutarFallbackLocal(postorId);
      }
    }

    try {
      // 2. Intentar llamar al servicio real de IA
      const resultado = await this.servicioIAReal.evaluarPostor(postorId, subastaId);
      
      // Si tuvo éxito, restaurar el circuito a CLOSED
      if (this.estado === EstadoCircuitBreaker.HALF_OPEN) {
        console.log('🛡️ [CIRCUIT BREAKER] Recuperación exitosa. Circuito vuelve a CLOSED.');
      }
      this.estado = EstadoCircuitBreaker.CLOSED;
      this.fallosConsecutivos = 0;
      return resultado;
    } catch (error: any) {
      this.fallosConsecutivos++;
      this.ultimoFalloTimestamp = Date.now();

      console.warn(
        `⚠️ [CIRCUIT BREAKER] Fallo detectado en API de IA (${this.fallosConsecutivos}/${this.umbralFallos}): ${error?.message}`
      );

      if (this.fallosConsecutivos >= this.umbralFallos) {
        this.estado = EstadoCircuitBreaker.OPEN;
        console.error('🚨 [CIRCUIT BREAKER] Umbral superado. ¡CIRCUITO ABIERTO! Modo contingencia activado.');
      }

      // En caso de fallo, nunca se detiene la subasta: devolvemos fallback
      return this.ejecutarFallbackLocal(postorId);
    }
  }

  private ejecutarFallbackLocal(postorId: string): EvaluacionRiesgoIA {
    return {
      score: 75,
      nivelRiesgo: NivelRiesgo.MEDIO,
      porcentajeGarantia: 0.10, // 10% estándar seguro por contingencia
      justificacion: 'Evaluación heurística local por contingencia (API de IA no disponible temporalmente)',
      modoFallbackActivado: true
    };
  }
}
