import { EstadoGarantia } from '../dominio/modelos/garantia.modelo.ts';
import { NivelRiesgo } from '../dominio/contratos/agente-riesgo.contrato.ts';
import { GeminiRiesgoAdapter } from '../infraestructura/gemini-riesgo.adapter.ts';
import { CircuitBreakerIA, EstadoCircuitBreaker } from '../infraestructura/circuit-breaker-ia.ts';
import { PasarelaBancariaSimulada } from '../infraestructura/pasarela-bancaria-simulada.ts';
import { RepositorioGarantiasMemoria } from '../infraestructura/repositorio-garantias-memoria.ts';
import { RetenerGarantiaCasoUso } from '../aplicacion/retener-garantia.caso-uso.ts';
import { LiquidarSubastaCasoUso } from '../aplicacion/liquidar-subasta.caso-uso.ts';

async function ejecutarPruebasGarantiaEIA() {
  console.log('--- INICIANDO PRUEBAS DE CUSTODIA (ESCROW) Y AGENTE IA (SDD SPEC-002) ---\n');

  // 1. Instanciación de componentes mediante Clean Architecture
  const geminiAdapter = new GeminiRiesgoAdapter();
  const circuitBreaker = new CircuitBreakerIA(geminiAdapter);
  const pasarela = new PasarelaBancariaSimulada();
  const repo = new RepositorioGarantiasMemoria();

  const ucRetener = new RetenerGarantiaCasoUso(circuitBreaker, pasarela, repo);
  const ucLiquidar = new LiquidarSubastaCasoUso(pasarela, repo);

  const subastaId = 'SUB-LOTE-ORO-2026';
  const precioBase = 50000; // S/. 50,000

  // [TEST 1] Evaluación IA de Bajo Riesgo (10% de garantía)
  console.log('[TEST 1] Postor USR-VIP-01 solicita garantía. Gemini evalúa riesgo:');
  const res1 = await ucRetener.ejecutar({
    postorId: 'USR-VIP-01',
    subastaId,
    precioBaseSubasta: precioBase
  });
  console.log('-> Resultado:', {
    exito: res1.exito,
    scoreIA: res1.evaluacionIA?.score,
    nivelRiesgo: res1.evaluacionIA?.nivelRiesgo,
    montoGarantia: res1.garantia?.montoRetenido,
    estado: res1.garantia?.estado
  });

  if (
    !res1.exito ||
    res1.evaluacionIA?.nivelRiesgo !== NivelRiesgo.BAJO ||
    res1.garantia?.montoRetenido !== 5000 || // 10% de 50,000
    res1.garantia?.estado !== EstadoGarantia.RETENIDA
  ) {
    throw new Error('FALLÓ TEST 1: Debería retener S/. 5,000 con riesgo bajo');
  }
  console.log('✅ TEST 1 APROBADO: Perfil confiable con 10% de garantía retenida.\n');

  // [TEST 2] Evaluación IA de Alto Riesgo (Garantía aumentada al 20%)
  console.log('[TEST 2] Postor sospechoso USR-RIESGO_ALTO solicita garantía:');
  const res2 = await ucRetener.ejecutar({
    postorId: 'USR-RIESGO_ALTO',
    subastaId,
    precioBaseSubasta: precioBase
  });
  console.log('-> Resultado:', {
    exito: res2.exito,
    scoreIA: res2.evaluacionIA?.score,
    nivelRiesgo: res2.evaluacionIA?.nivelRiesgo,
    montoGarantia: res2.garantia?.montoRetenido,
    justificacion: res2.evaluacionIA?.justificacion
  });

  if (
    !res2.exito ||
    res2.evaluacionIA?.nivelRiesgo !== NivelRiesgo.ALTO ||
    res2.garantia?.montoRetenido !== 10000 // 20% de 50,000
  ) {
    throw new Error('FALLÓ TEST 2: Debería retener S/. 10,000 (20%) por riesgo alto');
  }
  console.log('✅ TEST 2 APROBADO: Detección inteligente de riesgo con garantía ajustada al 20%.\n');

  // [TEST 3] Resiliencia ante caída de Gemini (Circuit Breaker y Fallback Local)
  console.log('[TEST 3] Simulación de falla en Gemini (API caída / 503). Probando Circuit Breaker:');
  geminiAdapter.simularCaidaServidor(true); // Forzamos fallo de conexión

  // Primera llamada: detecta fallo y registra
  await ucRetener.ejecutar({
    postorId: 'USR-FALLO-1',
    subastaId,
    precioBaseSubasta: precioBase
  });

  // Segunda llamada: acumula 2 fallos y abre el circuito (OPEN)
  const res3 = await ucRetener.ejecutar({
    postorId: 'USR-FALLO-2',
    subastaId,
    precioBaseSubasta: precioBase
  });

  console.log('-> Estado Circuit Breaker:', circuitBreaker.getEstado());
  console.log('-> Respuesta con Fallback:', {
    exito: res3.exito,
    modoFallback: res3.evaluacionIA?.modoFallbackActivado,
    score: res3.evaluacionIA?.score,
    montoGarantia: res3.garantia?.montoRetenido
  });

  if (
    !res3.exito ||
    !res3.evaluacionIA?.modoFallbackActivado ||
    circuitBreaker.getEstado() !== EstadoCircuitBreaker.OPEN
  ) {
    throw new Error('FALLÓ TEST 3: El Circuit Breaker no activó el modo contingencia (OPEN)');
  }
  console.log('✅ TEST 3 APROBADO: Circuit Breaker protegió el sistema con Fallback Local sin detener la subasta.\n');

  // [TEST 4] Liquidación Atómica al Martillazo Final (Capture + Mass Release)
  console.log('[TEST 4] Adjudicación de la subasta: ganador USR-VIP-01 por S/. 62,000:');
  const resLiquidar = await ucLiquidar.ejecutar({
    subastaId,
    postorGanadorId: 'USR-VIP-01',
    montoFinalAdjudicado: 62000
  });

  console.log('-> Resultado Liquidación:', resLiquidar);

  const garantiaGanador = await repo.obtenerGarantia('USR-VIP-01', subastaId, false);
  const garantiasSubasta = await repo.obtenerPorSubasta(subastaId);

  const perdedoresLiberados = garantiasSubasta.filter(
    (g) => g.postorId !== 'USR-VIP-01' && g.estado === EstadoGarantia.LIBERADA
  );

  console.log(`-> Garantía Ganador Estado: ${garantiaGanador?.estado}`);
  console.log(`-> Garantías Perdedores Liberadas: ${perdedoresLiberados.length}`);

  if (
    !resLiquidar.exito ||
    garantiaGanador?.estado !== EstadoGarantia.CAPTURADA_COBRO ||
    perdedoresLiberados.length < 2
  ) {
    throw new Error('FALLÓ TEST 4: Error en liquidación o liberación de garantías');
  }
  console.log('✅ TEST 4 APROBADO: Fondos del ganador capturados y perdedores liberados de inmediato.\n');

  console.log('🎉 --- TODOS LOS TESTS DE LA ESPECIFICACIÓN 002 APROBADOS AL 100% ✅ ---');
}

ejecutarPruebasGarantiaEIA().catch((err) => {
  console.error('\n❌ ERROR EN LA SUITE DE PRUEBAS:', err);
  process.exit(1);
});
