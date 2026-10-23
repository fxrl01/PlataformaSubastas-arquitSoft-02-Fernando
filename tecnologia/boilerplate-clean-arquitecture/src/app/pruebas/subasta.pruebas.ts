import { Subasta } from '../dominio/modelos/subasta.modelo.ts';
import { EstadoSubasta } from '../dominio/modelos/estado-subasta.enum.ts';
import { CodigoErrorPuja } from '../dominio/modelos/errores-puja.ts';
import { RepositorioSubastasMemoria } from '../infraestructura/repositorio-subastas-memoria.ts';
import { MotorPujasSimulado } from '../infraestructura/motor-pujas-simulado.ts';
import { ProcesadorPagosSimulado } from '../infraestructura/procesador-pagos-simulado.ts';
import { NotificadorSalaConsola } from '../infraestructura/notificador-sala-consola.ts';
import { ProcesarPujaCasoUso } from '../aplicacion/procesar-puja.caso-uso.ts';
import { PujasControlador } from '../presentacion/pujas.controlador.ts';

async function ejecutarPruebas() {
  console.log('--- INICIANDO SUITE DE PRUEBAS AUTOMATIZADAS (SDD SPEC-001) ---\n');

  // 1. Instanciación e Inversión de Dependencias (DIP)
  const repo = new RepositorioSubastasMemoria();
  const motor = new MotorPujasSimulado();
  const pagos = new ProcesadorPagosSimulado();
  const notificador = new NotificadorSalaConsola();
  const casoUso = new ProcesarPujaCasoUso(repo, motor, pagos, notificador);
  const controlador = new PujasControlador(casoUso);

  // 2. Preparar subasta inicial
  const subastaInicial = new Subasta(
    'SUB-2026-001',
    'Vehículo Toyota Hilux 2022 4x4',
    45000, // Precio base S/. 45,000
    1000,  // Incremento mínimo S/. 1,000
    45000, // Precio actual inicial
    EstadoSubasta.ACTIVA,
    new Date(Date.now() + 60000) // Cierra en 60 segundos
  );
  await repo.guardar(subastaInicial);

  console.log('📦 Subasta creada: SUB-2026-001 | Precio Base: S/. 45,000 | Incremento: S/. 1,000\n');

  // [TEST 1] Criterio de Aceptación 4: Rechazo Zero Trust sin depósito en garantía
  console.log('[TEST 1] Postor USR-01 puja S/. 46,000 SIN haber depositado garantía (Zero Trust):');
  const res1 = await controlador.atenderEventoPuja({
    subastaId: 'SUB-2026-001',
    postorId: 'USR-01',
    monto: 46000
  });
  console.log('-> Resultado:', res1);
  if (res1.status !== 403 || res1.codigoError !== CodigoErrorPuja.GARANTIA_REQUERIDA) {
    throw new Error('FALLÓ TEST 1: Debería retornar status 403 y GARANTIA_REQUERIDA');
  }
  console.log('✅ TEST 1 APROBADO: Rechazo en borde por falta de garantía.\n');

  // [TEST 2] Criterio de Aceptación 1: Puja válida con garantía activa
  pagos.habilitarGarantia('USR-01', 'SUB-2026-001');
  console.log('[TEST 2] Postor USR-01 con garantía habilitada puja válidamente S/. 46,000:');
  const res2 = await controlador.atenderEventoPuja({
    subastaId: 'SUB-2026-001',
    postorId: 'USR-01',
    monto: 46000
  });
  console.log('-> Resultado:', res2);
  if (res2.status !== 200 || !res2.data?.exito) {
    throw new Error('FALLÓ TEST 2: La puja debería ser aceptada');
  }
  console.log('✅ TEST 2 APROBADO: Puja líder aceptada y difundida.\n');

  // [TEST 3] Criterio de Aceptación 5: Prohibición de autopuja del líder vigente
  console.log('[TEST 3] Postor USR-01 (ya es líder) intenta volver a pujar S/. 48,000 sobre sí mismo:');
  const res3 = await controlador.atenderEventoPuja({
    subastaId: 'SUB-2026-001',
    postorId: 'USR-01',
    monto: 48000
  });
  console.log('-> Resultado:', res3);
  if (res3.status !== 400 || res3.codigoError !== CodigoErrorPuja.POSTOR_YA_ES_LIDER) {
    throw new Error('FALLÓ TEST 3: Debería rechazar autopuja con POSTOR_YA_ES_LIDER');
  }
  console.log('✅ TEST 3 APROBADO: Autopuja prevenida correctamente.\n');

  // [TEST 4] Regla de Negocio: Rechazo de monto insuficiente
  pagos.habilitarGarantia('USR-02', 'SUB-2026-001');
  console.log('[TEST 4] Postor USR-02 intenta pujar S/. 46,500 (Incremento requerido: S/. 1,000):');
  const res4 = await controlador.atenderEventoPuja({
    subastaId: 'SUB-2026-001',
    postorId: 'USR-02',
    monto: 46500
  });
  console.log('-> Resultado:', res4);
  if (res4.status !== 409 || res4.codigoError !== CodigoErrorPuja.MONTO_INSUFICIENTE) {
    throw new Error('FALLÓ TEST 4: Debería rechazar por MONTO_INSUFICIENTE');
  }
  console.log('✅ TEST 4 APROBADO: Regla de incremento mínimo respetada.\n');

  // [TEST 5] Criterio de Aceptación 2: Activación de la Regla Anti-Sniping
  console.log('[TEST 5] Subasta próxima a vencer en 10 segundos. Postor USR-02 puja S/. 47,000:');
  const subasta = await repo.obtenerPorId('SUB-2026-001');
  if (subasta) {
    // Forzamos el reloj a faltar solo 10 segundos
    subasta.finProgramado = new Date(Date.now() + 10000);
    await repo.guardar(subasta);
  }

  const res5 = await controlador.atenderEventoPuja({
    subastaId: 'SUB-2026-001',
    postorId: 'USR-02',
    monto: 47000
  });
  console.log('-> Resultado:', res5);
  if (!res5.data?.antiSnipingActivado) {
    throw new Error('FALLÓ TEST 5: La regla Anti-Sniping debió haberse activado');
  }
  console.log('✅ TEST 5 APROBADO: Regla Anti-Sniping extendió automáticamente el cronómetro (+30s).\n');

  // [TEST 6] Criterio de Aceptación 3: Resolución de Colisión Concurrente (Mutex Lock)
  pagos.habilitarGarantia('USR-03', 'SUB-2026-001');
  console.log('[TEST 6] Simulación de colisión concurrente: lock en Redis ocupado por otro proceso:');
  await motor.adquirirLock('SUB-2026-001'); // Bloqueamos el lock manualmente
  const res6 = await controlador.atenderEventoPuja({
    subastaId: 'SUB-2026-001',
    postorId: 'USR-03',
    monto: 49000
  });
  console.log('-> Resultado:', res6);
  await motor.liberarLock('SUB-2026-001'); // Liberamos
  if (res6.status !== 503 || res6.codigoError !== CodigoErrorPuja.COLISION_CONCURRENTE) {
    throw new Error('FALLÓ TEST 6: Debería retornar 503 y COLISION_CONCURRENTE');
  }
  console.log('✅ TEST 6 APROBADO: Colisión concurrente detectada sin alterar el estado.\n');

  console.log('🎉 --- TODOS LOS TESTS DE LA ESPECIFICACIÓN 001 APROBADOS AL 100% ✅ ---');
}

ejecutarPruebas().catch((err) => {
  console.error('\n❌ ERROR EN LA SUITE DE PRUEBAS:', err);
  process.exit(1);
});
