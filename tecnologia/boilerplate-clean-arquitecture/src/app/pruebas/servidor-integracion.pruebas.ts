import http from 'node:http';
import { servidor } from '../../main.ts';

function peticionHTTP(opciones: http.RequestOptions, body?: any): Promise<{ status: number; data: any }> {
  return new Promise((resolve, reject) => {
    const req = http.request(opciones, (res) => {
      let respuestaStr = '';
      res.on('data', (chunk) => (respuestaStr += chunk));
      res.on('end', () => {
        try {
          const data = respuestaStr ? JSON.parse(respuestaStr) : {};
          resolve({ status: res.statusCode || 500, data });
        } catch (err) {
          reject(err);
        }
      });
    });

    req.on('error', reject);

    if (body) {
      req.write(JSON.stringify(body));
    }
    req.end();
  });
}

async function ejecutarPruebasIntegracionServidor() {
  console.log('--- INICIANDO PRUEBAS DE INTEGRACIÓN DEL SERVIDOR HTTP REST (SPEC-003) ---\n');

  const PUERTO_TEST = 3099;

  // 1. Levantar servidor en puerto de prueba
  await new Promise<void>((resolve) => {
    servidor.listen(PUERTO_TEST, () => {
      console.log(`🌐 Servidor de prueba escuchando en http://localhost:${PUERTO_TEST}`);
      resolve();
    });
  });

  try {
    // [TEST 1] GET /health
    console.log('\n[TEST 1] Comprobando Health Check GET /health:');
    const resHealth = await peticionHTTP({
      hostname: 'localhost',
      port: PUERTO_TEST,
      path: '/health',
      method: 'GET'
    });
    console.log('-> Status:', resHealth.status, '| Payload:', resHealth.data);
    if (resHealth.status !== 200 || resHealth.data.status !== 'UP') {
      throw new Error('FALLÓ TEST 1: El endpoint /health no respondió UP');
    }
    console.log('✅ TEST 1 APROBADO: Servidor saludable y operativo.');

    // [TEST 2] GET /api/v1/subastas/SUB-2026-001
    console.log('\n[TEST 2] Consultando estado inicial del lote GET /api/v1/subastas/SUB-2026-001:');
    const resSubasta = await peticionHTTP({
      hostname: 'localhost',
      port: PUERTO_TEST,
      path: '/api/v1/subastas/SUB-2026-001',
      method: 'GET'
    });
    console.log('-> Status:', resSubasta.status, '| Datos:', resSubasta.data);
    if (resSubasta.status !== 200 || resSubasta.data.precioActual !== 45000) {
      throw new Error('FALLÓ TEST 2: No se encontró la subasta inicial');
    }
    console.log('✅ TEST 2 APROBADO: Catálogo de subastas consultable.');

    // [TEST 3] POST /puja sin garantía (Zero Trust)
    console.log('\n[TEST 3] Postor USR-HTTP-01 emite puja SIN garantía:');
    const resPujaSinGarantia = await peticionHTTP(
      {
        hostname: 'localhost',
        port: PUERTO_TEST,
        path: '/api/v1/subastas/SUB-2026-001/puja',
        method: 'POST',
        headers: { 'Content-Type': 'application/json' }
      },
      { postorId: 'USR-HTTP-01', monto: 46000 }
    );
    console.log('-> Status:', resPujaSinGarantia.status, '| Error:', resPujaSinGarantia.data.error);
    if (resPujaSinGarantia.status !== 403) {
      throw new Error('FALLÓ TEST 3: Debería rechazar con status 403');
    }
    console.log('✅ TEST 3 APROBADO: Rechazo Zero Trust en la capa HTTP REST.');

    // [TEST 4] POST /garantia para USR-HTTP-01 (Con IA)
    console.log('\n[TEST 4] Postor USR-HTTP-01 solicita garantía vía POST /garantia:');
    const resGarantia = await peticionHTTP(
      {
        hostname: 'localhost',
        port: PUERTO_TEST,
        path: '/api/v1/subastas/SUB-2026-001/garantia',
        method: 'POST',
        headers: { 'Content-Type': 'application/json' }
      },
      { postorId: 'USR-HTTP-01' }
    );
    console.log('-> Status:', resGarantia.status, '| Garantía constituida:', {
      exito: resGarantia.data.exito,
      monto: resGarantia.data.garantia?.montoRetenido,
      scoreIA: resGarantia.data.evaluacionIA?.score
    });
    if (resGarantia.status !== 200 || !resGarantia.data.exito) {
      throw new Error('FALLÓ TEST 4: No se pudo constituir la garantía');
    }
    console.log('✅ TEST 4 APROBADO: Garantía registrada y evaluada con IA vía HTTP.');

    // [TEST 5] POST /puja válida con garantía activa
    console.log('\n[TEST 5] Postor USR-HTTP-01 emite puja válida de S/. 46,000 con garantía:');
    const resPujaValida = await peticionHTTP(
      {
        hostname: 'localhost',
        port: PUERTO_TEST,
        path: '/api/v1/subastas/SUB-2026-001/puja',
        method: 'POST',
        headers: { 'Content-Type': 'application/json' }
      },
      { postorId: 'USR-HTTP-01', monto: 46000 }
    );
    console.log('-> Status:', resPujaValida.status, '| Respuesta:', resPujaValida.data.data?.mensaje);
    if (resPujaValida.status !== 200 || !resPujaValida.data.data?.exito) {
      throw new Error('FALLÓ TEST 5: La puja con garantía debió ser aceptada');
    }
    console.log('✅ TEST 5 APROBADO: Oferta procesada y registrada en vivo.');

    // [TEST 6] POST /liquidar subasta
    console.log('\n[TEST 6] Martillero liquida la subasta adjudicando a USR-HTTP-01:');
    const resLiquidar = await peticionHTTP(
      {
        hostname: 'localhost',
        port: PUERTO_TEST,
        path: '/api/v1/subastas/SUB-2026-001/liquidar',
        method: 'POST',
        headers: { 'Content-Type': 'application/json' }
      },
      { postorGanadorId: 'USR-HTTP-01', montoFinal: 46000 }
    );
    console.log('-> Status:', resLiquidar.status, '| Mensaje:', resLiquidar.data.mensaje);
    if (resLiquidar.status !== 200 || !resLiquidar.data.exito) {
      throw new Error('FALLÓ TEST 6: Error al liquidar la subasta');
    }
    console.log('✅ TEST 6 APROBADO: Liquidación de subasta y captura de garantía completada.');

    console.log('\n🎉 --- TODAS LAS PRUEBAS DE INTEGRACIÓN HTTP REST FINALIZADAS CON ÉXITO ✅ ---');
  } finally {
    // Cerrar servidor
    servidor.close();
  }
}

ejecutarPruebasIntegracionServidor().catch((err) => {
  console.error('\n❌ ERROR EN PRUEBAS DE INTEGRACIÓN:', err);
  servidor.close();
  process.exit(1);
});
