import type { AgenteRiesgoContrato, EvaluacionRiesgoIA } from '../dominio/contratos/agente-riesgo.contrato.ts';
import { NivelRiesgo } from '../dominio/contratos/agente-riesgo.contrato.ts';

export class GeminiRiesgoAdapter implements AgenteRiesgoContrato {
  private estaServidorCaido: boolean = false;

  public simularCaidaServidor(caido: boolean): void {
    this.estaServidorCaido = caido;
  }

  public async evaluarPostor(postorId: string, subastaId: string): Promise<EvaluacionRiesgoIA> {
    if (this.estaServidorCaido) {
      throw new Error('Gemini API Connection Timeout / 503 Service Unavailable');
    }

    // Análisis inteligente de perfil de riesgo simulando el razonamiento de Gemini:
    if (postorId.includes('RIESGO_ALTO') || postorId === 'USR-BOT-99') {
      return {
        score: 38,
        nivelRiesgo: NivelRiesgo.ALTO,
        porcentajeGarantia: 0.20, // 20% de garantía por riesgo elevado de abandono
        justificacion: 'Patrón anómalo detectado: alta velocidad de conexión y antecedentes de pujas impagas',
        modoFallbackActivado: false
      };
    }

    return {
      score: 92,
      nivelRiesgo: NivelRiesgo.BAJO,
      porcentajeGarantia: 0.10, // 10% estándar para postores confiables
      justificacion: 'Perfil verificado con KYC aprobado y comportamiento de puja consistente',
      modoFallbackActivado: false
    };
  }
}
