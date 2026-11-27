// @ts-check
const path = require('path');
const { test, expect } = require(path.join(__dirname, '../e2e/node_modules/@playwright/test'));

test.describe('LiveBid — Suite Dinámica de Seguridad OWASP (DAST: Cabeceras, Auth/RBAC, Rate Limit, DoS y XSS)', () => {
  test('OWASP A05: Cabeceras de endurecimiento HTTP presentes y vitrina pública sanitizada sin datos privados', async ({ request }) => {
    const res = await request.get('/api/v1/subastas');
    expect(res.ok()).toBeTruthy();

    const headers = res.headers();
    expect(headers['x-content-type-options']).toBe('nosniff');
    expect(headers['x-frame-options']).toBe('DENY');
    expect(headers['referrer-policy']).toBe('strict-origin-when-cross-origin');
    expect(headers['permissions-policy']).toContain('camera=()');

    const vitrina = await request.get('/api/v1/vendedores/usr_admin_root/vitrina');
    expect(vitrina.ok()).toBeTruthy();
    const raw = await vitrina.text();
    expect(raw).not.toContain('admin@livebid.pe');
    expect(raw).not.toContain('password');
    expect(raw).not.toContain('document_id');
  });

  test('OWASP A01 & A07: Bloqueo de JWT inválido (401) y prevención de escalada de privilegios POSTOR -> ADMIN (403)', async ({ request }) => {
    const forgedJwt = 'Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiJhdHRhY2tlciIsInJvbGUiOiJBRE1JTiJ9.invalidsig';

    const unauthAdmin = await request.get('/api/v1/admin/users', {
      headers: { Authorization: forgedJwt },
    });
    expect(unauthAdmin.status()).toBe(401);

    const unauthLike = await request.post('/api/v1/social/posts/soc_seed_01/like');
    expect(unauthLike.status()).toBe(401);

    // Registrar un usuario POSTOR, aprobarlo como ADMIN y comprobar que NO puede escalar a rutas ADMIN ni crear lotes
    const suffix = Date.now();
    const bidderEmail = `sec.postor.${suffix}@livebid.pe`;
    const bidderPass = 'SecPostor!2026';

    const regRes = await request.post('/api/v1/auth/register', {
      headers: { 'CF-Connecting-IP': `198.51.100.${(suffix % 200) + 1}` },
      data: {
        name: 'Postor Auditoria RBAC',
        email: bidderEmail,
        password: bidderPass,
        rol_solicitado: 'POSTOR',
        documento_identidad: 'DNI 44556677',
      },
    });
    expect(regRes.status()).toBe(201);
    const regBody = await regRes.json();
    const userId = regBody.id || regBody.user?.id;
    expect(userId).toBeTruthy();

    const adminLogin = await request.post('/api/v1/auth/login', {
      data: { email: 'admin@livebid.pe', password: 'AdminLiveBid2026!' },
    });
    expect(adminLogin.ok()).toBeTruthy();
    const { access_token: adminToken } = await adminLogin.json();

    const approveRes = await request.patch(`/api/v1/admin/users/${userId}/status`, {
      headers: { Authorization: `Bearer ${adminToken}` },
      data: { estado: 'ACTIVO', kyc_aprobado: true },
    });
    expect(approveRes.ok()).toBeTruthy();

    const bidderLogin = await request.post('/api/v1/auth/login', {
      data: { email: bidderEmail, password: bidderPass },
    });
    expect(bidderLogin.ok()).toBeTruthy();
    const { access_token: bidderToken } = await bidderLogin.json();

    // Intento de acceso horizontal/vertical prohibido para POSTOR -> debe responder 403 Forbidden
    const forbiddenAdmin = await request.get('/api/v1/admin/users', {
      headers: { Authorization: `Bearer ${bidderToken}` },
    });
    expect(forbiddenAdmin.status()).toBe(403);

    const forbiddenCreateAuction = await request.post('/api/v1/subastas', {
      headers: { Authorization: `Bearer ${bidderToken}` },
      data: { titulo: 'Subasta No Autorizada', precio_base: 1000, duracion_segundos: 300 },
    });
    expect(forbiddenCreateAuction.status()).toBe(403);
  });

  test('OWASP A04: Mitigación de Fuerza Bruta / Abuso de API (429 Too Many Requests) y límite de Payload DoS (>2 MiB)', async ({ request }) => {
    const attackerIP = `203.0.113.${Math.floor(Math.random() * 200) + 10}`;
    let rateLimited429 = false;
    let retryAfterHeader = '';

    // 2 réplicas x 60 req/min = máximo 120; enviamos hasta 140 desde una IP atacante aislada
    for (let i = 0; i < 140; i++) {
      const res = await request.post('/api/v1/ia/asistente', {
        headers: { 'CF-Connecting-IP': attackerIP },
        data: { mensaje: 'ping', subasta_id: 'SUB-2026-001' },
      });
      if (res.status() === 429) {
        rateLimited429 = true;
        retryAfterHeader = res.headers()['retry-after'] || '';
        break;
      }
    }
    expect(rateLimited429).toBe(true);
    expect(retryAfterHeader).toBe('60');

    // Verificar que un usuario legítimo desde otra IP NO fue bloqueado
    const legitRes = await request.post('/api/v1/ia/asistente', {
      headers: { 'CF-Connecting-IP': '192.0.2.99' },
      data: { mensaje: '¿Cómo usar el cronograma?', subasta_id: 'SUB-2026-001' },
    });
    expect(legitRes.ok()).toBeTruthy();

    // Payload gigante de ~2.4 MiB debe ser rechazado (413 en Nginx o 400/413 en MaxBytesReader) sin afectar /health
    const oversizedPayload = 'A'.repeat(2_400_000);
    const dosRes = await request.post('/api/v1/ia/asistente', {
      headers: { 'CF-Connecting-IP': '192.0.2.88' },
      data: { mensaje: oversizedPayload },
    });
    expect([400, 413]).toContain(dosRes.status());

    const healthAfterDoS = await request.get('/health');
    expect(healthAfterDoS.ok()).toBeTruthy();
  });

  test('OWASP A03: Neutralización de cargas XSS e Inyección SQL en buscador y renderizado DOM', async ({ page, request }) => {
    const sqliRes = await request.get(`/api/v1/subastas/${encodeURIComponent("' OR '1'='1")}`);
    expect([400, 404]).toContain(sqliRes.status());

    await page.goto('/');
    await page.locator('#scheduleSearchInput').fill('<img src=x onerror="window.__xssTriggered=true"><script>window.__xssTriggered=true</script>');
    const xssTriggered = await page.evaluate(() => Boolean(/** @type {any} */ (window).__xssTriggered));
    expect(xssTriggered).toBe(false);
  });
});
