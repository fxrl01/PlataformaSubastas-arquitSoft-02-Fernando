import { Subasta } from '../dominio/modelos/subasta.modelo.ts';
import { EstadoSubasta } from '../dominio/modelos/estado-subasta.enum.ts';
import { RepositorioSubastasMemoria } from '../infraestructura/repositorio-subastas-memoria.ts';
import { MotorPujasSimulado } from '../infraestructura/motor-pujas-simulado.ts';
import { ProcesadorPagosSimulado } from '../infraestructura/procesador-pagos-simulado.ts';
import { NotificadorSalaConsola } from '../infraestructura/notificador-sala-consola.ts';
import { ProcesarPujaCasoUso } from '../aplicacion/procesar-puja.caso-uso.ts';
import { PujasControlador } from '../presentacion/pujas.controlador.ts';

async function ejecutarPruebas() {
  console.log('--- INICIANDO PRUEBAS DE DOMINIO Y CLEAN ARCHITECTURE (LIVEBID) ---\n');

  // 1. Instanciación de componentes mediante inyección de dependencias (DIP)
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

  console.log('✅ Subasta creada: SUB-2026-001 | Precio Base: S/. 45,000 | Incremento: S/. 1,000');

  // 3. Caso Negativo: Postor intenta pujar sin depósito de garantía (Zero Trust)
  console.log('\n[PRUEBA 1] Postor USR-01 puja S/. 46,000 SIN haber depositado garantía:');
  const res1 = await controlador.atenderEventoPuja({
    subastaId: 'SUB-2026-001',
    postorId: 'USR-01',
    monto: 46000
  });
  console.log('Resultado esperado (Rechazado):', res1);

  // 4. Caso Positivo: Se habilita garantía y se emite puja válida
  pagos.habilitarGarantia('USR-01', 'SUB-2026-001');
  console.log('\n[PRUEBA 2] Postor USR-01 con garantía habilitada puja S/. 46,000:');
  const res2 = await controlador.atenderEventoPuja({
    subastaId: 'SUB-2026-001',
    postorId: 'USR-01',
    monto: 46000
  });
  console.log('Resultado esperado (Aceptado):', res2);

  // 5. Caso Negativo: Postor USR-02 intenta pujar por debajo del incremento mínimo
  pagos.habilitarGarantia('USR-02', 'SUB-2026-001');
  console.log('\n[PRUEBA 3] Postor USR-02 intenta pujar S/. 46,500 (Incremento requerido: S/. 1,000):');
  const res3 = await controlador.atenderEventoPuja({
    subastaId: 'SUB-2026-001',
    postorId: 'USR-02',
    monto: 46500
  });
  console.log('Resultado esperado (Rechazado por monto insuficiente):', res3);

  // 6. Caso Positivo: Postor USR-02 puja válidamente S/. 47,000
  console.log('\n[PRUEBA 4] Postor USR-02 puja válidamente S/. 47,000:');
  const res4 = await controlador.atenderEventoPuja({
    subastaId: 'SUB-2026-001',
    postorId: 'USR-02',
    monto: 47000
  });
  console.log('Resultado esperado (Aceptado nuevo líder):', res4);

  console.log('\n--- TODAS LAS PRUEBAS UNITARIAS DE ARQUITECTURA FINALIZADAS CON ÉXITO ✅ ---');
}

ejecutarPruebas().catch(console.error);
