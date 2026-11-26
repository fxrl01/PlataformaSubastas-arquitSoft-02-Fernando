// @ts-check
const { test, expect } = require('@playwright/test');

test.describe('LiveBid — E2E Real sobre Cluster Nginx + 2 Réplicas Go + PgBouncer + Postgres + Redis', () => {
  test('Gateway Nginx balancea entre réplicas Go y aplica caché HTTP en estáticos', async ({ request }) => {
    const instances = new Set();
    for (let i = 0; i < 8; i++) {
      const res = await request.get('/health');
      expect(res.ok()).toBeTruthy();
      const body = await res.json();
      expect(body.status).toBe('UP');
      expect(body.cache).toBe('redis-aside');
      if (body.instance) instances.add(body.instance);
    }
    expect(instances.size).toBeGreaterThanOrEqual(2);

    const cssRes = await request.get('/styles.css');
    expect(cssRes.ok()).toBeTruthy();
    expect(cssRes.headers()['cache-control']).toContain('max-age=3600');
    expect(cssRes.headers()['x-static-layer']).toBe('nginx-frontend-cache');

    const swaggerRes = await request.get('/swagger');
    expect(swaggerRes.ok()).toBeTruthy();
    expect(swaggerRes.headers()['content-type']).toContain('text/html');
    expect(await swaggerRes.text()).toContain('SwaggerUIBundle');

    const openApiRes = await request.get('/api/v1/openapi.json');
    expect(openApiRes.ok()).toBeTruthy();
    const openApiSpec = await openApiRes.json();
    expect(openApiSpec.openapi).toBe('3.0.3');

    const aiAssistantRes = await request.post('/api/v1/ia/asistente', {
      data: {
        mensaje: '¿Cómo puedo subir y vender un bien de valor?',
        subasta_id: 'SUB-2026-001',
        rol_usuario: 'VISITANTE'
      }
    });
    expect(aiAssistantRes.ok()).toBeTruthy();
    const aiBody = await aiAssistantRes.json();
    expect(aiBody.respuesta).toContain('Vendedor');
  });

  test('Flujo E2E completo con 2 navegadores: Registro -> KYC Admin -> Nueva Subasta -> Stripe Hold -> Puja WS -> Cierre y Merkle', async ({ browser, request }) => {
    const uniqueSuffix = Date.now();
    const bidderEmail = `e2e.postor.${uniqueSuffix}@livebid.pe`;
    const auctionTitle = `Porsche 911 E2E ${uniqueSuffix}`;

    const adminContext = await browser.newContext();
    const bidderContext = await browser.newContext();
    const adminPage = await adminContext.newPage();
    const bidderPage = await bidderContext.newPage();

    try {
      // 1. Navegador Postor: Verificar vitrina PostgreSQL + Asistente IA lateral y registrarse con rol POSTOR
      await bidderPage.goto('/');
      await expect(bidderPage.locator('#catalogGrid article.catalog-card').first()).toBeVisible({ timeout: 10_000 });
      await expect(bidderPage.locator('#navUserName')).toHaveText('Visitante');
      await bidderPage.locator('#btnOpenAiDrawer').click();
      await expect(bidderPage.locator('#aiAssistantDrawer')).toBeVisible();
      await bidderPage.locator('#btnCloseAiDrawer').click();
      await expect(bidderPage.locator('#aiAssistantDrawer')).toBeHidden({ timeout: 5_000 });

      await bidderPage.locator('#btnOpenAuthModal').click();
      await bidderPage.locator('[data-auth-tab="register"]').click();
      await bidderPage.locator('#regName').fill('Postor E2E Playwright');
      await bidderPage.locator('#regEmail').fill(bidderEmail);
      await bidderPage.locator('#regPassword').fill('PasswordE2E!2026');
      await bidderPage.locator('#regRole').selectOption('POSTOR');
      await bidderPage.locator('#regDoc').fill('DNI 70112233');
      await bidderPage.locator('#regPhone').fill('+51 999 000 111');
      await bidderPage.locator('#registerForm button[type="submit"]').click();
      await expect(bidderPage.locator('#feedbackToast')).toContainText('registrado con solicitud de rol', { timeout: 10_000 });

      // 2. Navegador Admin: Iniciar sesión como ADMIN y aprobar KYC del Postor
      await adminPage.goto('/');
      await adminPage.locator('#btnOpenAuthModal').click();
      await adminPage.locator('#loginEmail').fill('admin@livebid.pe');
      await adminPage.locator('#loginPassword').fill('AdminLiveBid2026!');
      await adminPage.locator('#loginForm button[type="submit"]').click();
      await expect(adminPage.locator('#btnOpenAdminModal')).toBeVisible({ timeout: 10_000 });

      await adminPage.locator('#btnOpenAdminModal').click();
      const userRow = adminPage.locator('#adminUsersTableBody tr', { hasText: bidderEmail });
      await expect(userRow).toBeVisible({ timeout: 10_000 });
      await userRow.locator('[data-action="approve"]').click();
      await expect(adminPage.locator('#adminFeedbackMsg')).toContainText('activado con KYC aprobado');
      await adminPage.locator('#adminDialog .btn-close-dialog').click();

      // 3. Navegador Admin: Crear y activar nueva subasta desde el modal de Lote
      await adminPage.locator('#btnOpenLotModal').click();
      await adminPage.locator('[data-lot-tab="nueva"]').click();
      await adminPage.locator('#newSubTitulo').fill(auctionTitle);
      await adminPage.locator('#newSubPrecioBase').fill('20000');
      await adminPage.locator('#newSubDesc').fill('Vehículo verificado en prueba End-to-End automatizada');
      await adminPage.locator('#newAuctionForm button[type="submit"]').click();

      await expect(adminPage.locator('#lotTitle')).toContainText(auctionTitle, { timeout: 10_000 });
      const createdIdText = await adminPage.locator('#lotIdTag').textContent();
      const auctionId = (createdIdText || '').replace('LOTE OFICIAL #', '').trim();
      expect(auctionId).toMatch(/^sub_/);

      // 4. Navegador Postor: Iniciar sesión ya aprobado, elegir el lote, retener garantía Stripe y pujar
      await bidderPage.reload();
      await bidderPage.locator('#btnOpenAuthModal').click();
      await bidderPage.locator('#loginEmail').fill(bidderEmail);
      await bidderPage.locator('#loginPassword').fill('PasswordE2E!2026');
      await bidderPage.locator('#loginForm button[type="submit"]').click();
      await expect(bidderPage.locator('#navUserRoleBadge')).toContainText('POSTOR');

      await bidderPage.locator('#auctionSelector').selectOption(auctionId);
      await expect(bidderPage.locator('#lotTitle')).toContainText(auctionTitle);

      // Autorizar garantía preventiva Stripe (capture_method: manual)
      await bidderPage.locator('#btnOpenStripeModal').click();
      await expect(bidderPage.locator('#stripeSummaryIntentId')).toContainText('pi_stripe_hold_', { timeout: 10_000 });
      await bidderPage.locator('#btnConfirmStripeHold').click();
      await expect(bidderPage.locator('#escrowBadge')).toContainText('RETENIDA', { timeout: 10_000 });

      // Emitir puja en vivo
      const minBidVal = await bidderPage.locator('#bidInput').inputValue();
      await bidderPage.locator('#btnEmitirPuja').click();
      await expect(bidderPage.locator('#currentPriceValue')).not.toHaveText('20,000', { timeout: 10_000 });
      expect(Number(minBidVal)).toBeGreaterThan(20000);

      // 5. Verificar sincronización en tiempo real vía WebSocket en el navegador del Admin
      await expect(adminPage.locator('#totalBidsCount')).toContainText('1 oferta', { timeout: 10_000 });

      // 6. Admin inicia conteo de martillo y cierra la subasta
      await adminPage.locator('#btnMartilloConteo').click();
      await adminPage.locator('#btnAdjudicar').click();
      await expect(adminPage.locator('#auctionPhaseTag')).toContainText('ESTRADO CERRADO', { timeout: 10_000 });

      // 7. Verificar Auditoría Web3 + Integridad de cadena SHA-256
      const auditRes = await request.get(`/api/v1/subastas/${auctionId}/auditoria-web3?indice=0`);
      expect(auditRes.ok()).toBeTruthy();
      const audit = await auditRes.json();
      expect(audit.estado).toBe('CERRADA');
      expect(audit.cadena_integra).toBe(true);
      expect(audit.total_pujas_auditadas).toBe(1);
      expect(audit.prueba_merkle?.verificada).toBe(true);
    } finally {
      await adminContext.close();
      await bidderContext.close();
    }
  });
});
