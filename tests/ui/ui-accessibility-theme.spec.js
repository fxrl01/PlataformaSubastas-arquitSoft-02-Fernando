// @ts-check
const path = require('path');
const { test, expect } = require(path.join(__dirname, '../e2e/node_modules/@playwright/test'));

test.describe('LiveBid — Suite Dedicada de UI, Accesibilidad (WCAG 2.2), Temas y Multi-Dispositivo', () => {
  test('Modo Claro / Oscuro persiste tras recarga y mantiene contraste legible en controles nativos', async ({ page }) => {
    await page.goto('/');
    await expect(page.locator('#scheduleGrid article.schedule-card').first()).toBeVisible();

    // Forzar Modo Claro desde Mi Perfil y verificar persistencia
    await page.locator('[data-nav-view="perfil"]').click();
    await expect(page.locator('#viewPerfil')).toBeVisible();
    await page.locator('input[name="themePreference"][value="light"]').check();

    const lightScheme = await page.evaluate(() => ({
      meta: document.querySelector('meta[name="color-scheme"]')?.getAttribute('content'),
      stored: localStorage.getItem('color-scheme'),
    }));
    expect(lightScheme.meta).toBe('light');
    expect(lightScheme.stored).toBe('light');

    // Cambiar a Modo Oscuro y verificar persistencia tras reload completo
    await page.locator('input[name="themePreference"][value="dark"]').check();
    await page.reload();

    const darkSchemeAfterReload = await page.evaluate(() => ({
      meta: document.querySelector('meta[name="color-scheme"]')?.getAttribute('content'),
      stored: localStorage.getItem('color-scheme'),
      selectColors: (() => {
        const sel = document.getElementById('auctionSelector');
        if (!sel) return { fg: '', bg: '' };
        const cs = getComputedStyle(sel);
        return { fg: cs.color, bg: cs.backgroundColor };
      })(),
    }));
    expect(darkSchemeAfterReload.meta).toBe('dark');
    expect(darkSchemeAfterReload.stored).toBe('dark');
    expect(darkSchemeAfterReload.selectColors.fg).not.toBe(darkSchemeAfterReload.selectColors.bg);
  });

  test('Accesibilidad WCAG: nombres accesibles, cierre por teclado Escape y barra fija inferior en móvil táctil', async ({ browser }) => {
    const mobileContext = await browser.newContext({
      viewport: { width: 375, height: 812 },
      hasTouch: true,
      isMobile: true,
    });
    const page = await mobileContext.newPage();

    try {
      await page.goto('/');
      await expect(page.locator('#scheduleGrid article.schedule-card').first()).toBeVisible();

      // 1. Verificar cero desborde lateral y barra de navegación inferior tipo app nativa en móvil
      const mobileLayout = await page.evaluate(() => {
        const nav = document.querySelector('.section-navbar');
        const navStyle = nav ? getComputedStyle(nav) : null;
        const tabRect = document.querySelector('.section-nav-link')?.getBoundingClientRect();
        return {
          overflow: document.documentElement.scrollWidth - document.documentElement.clientWidth,
          navPosition: navStyle?.position,
          tabHeight: tabRect ? Math.round(tabRect.height) : 0,
        };
      });
      expect(mobileLayout.overflow).toBeLessThanOrEqual(1);
      expect(mobileLayout.navPosition).toBe('fixed');
      expect(mobileLayout.tabHeight).toBeGreaterThanOrEqual(44);

      // 2. Verificar nombres accesibles en controles críticos y ausencia de API Docs público
      await expect(page.locator('header.top-nav #btnOpenSwaggerDocs')).toHaveCount(0);
      await expect(page.locator('#btnNotificationsBell')).toHaveAttribute('aria-label', /.+/);
      await expect(page.locator('#btnThemeToggle')).toHaveAttribute('aria-label', /.+/);
      await expect(page.locator('#btnOpenAiDrawer')).toHaveAttribute('aria-label', /.+/);

      // 3. Verificar navegación por teclado (Escape cierra diálogo nativo <dialog>)
      await page.locator('#btnOpenAuthModal').click();
      await expect(page.locator('#authDialog')).toBeVisible();
      await page.keyboard.press('Escape');
      await expect(page.locator('#authDialog')).toBeHidden();
    } finally {
      await mobileContext.close();
    }
  });

  test('Recorrido funcional de las 5 vistas SPA y consulta interactiva en el Asistente IA LiveBid', async ({ page }) => {
    await page.goto('/');

    // Vista 1: Cronograma y filtro dinámico de búsqueda por nombre de bien
    await expect(page.locator('#viewCronograma')).toBeVisible();
    const firstCard = page.locator('#scheduleGrid article.schedule-card').first();
    await expect(firstCard).toBeVisible();
    const rawHeading = ((await firstCard.locator('h3').textContent()) || '').trim();
    const sampleWord = rawHeading.split(/\s+/).slice(1).find((w) => w.length >= 2) || 'Subasta';
    await page.locator('#scheduleSearchInput').fill(sampleWord);
    await expect(page.locator('#scheduleGrid article.schedule-card').first()).toBeVisible();
    await page.locator('#scheduleSearchInput').fill('');

    // Vista 2: Sala en Vivo centrada en el usuario + chat público
    await page.locator('[data-nav-view="sala"]').click();
    await expect(page.locator('#viewSala')).toBeVisible();
    await expect(page.locator('#roomLiveChatList')).toBeVisible();

    // Vista 3: Catálogo con filtros por categoría
    await page.locator('[data-nav-view="catalogo"]').click();
    await expect(page.locator('#viewCatalogo')).toBeVisible();
    await expect(page.locator('#catalogGrid article.catalog-card').first()).toBeVisible();

    // Vista 4: Social Feed de la comunidad
    await page.locator('[data-nav-view="social"]').click();
    await expect(page.locator('#viewSocial')).toBeVisible();
    await expect(page.locator('#socialFeedList article.social-post-card').first()).toBeVisible();

    // Vista 5: Asistente IA LiveBid responde dinámicamente con guía contextual
    await page.locator('#btnOpenAiDrawer').click();
    await expect(page.locator('#aiAssistantDrawer')).toBeVisible();
    await page.locator('#aiChatInput').fill('¿Cómo funciona el depósito de garantía reembolsable?');
    await page.locator('#btnSendAiChat').click();
    await expect(page.locator('#aiChatMessages .ai-msg.assistant').last()).toContainText(/garantía|reembolsable|tarjeta/i);
  });
});
