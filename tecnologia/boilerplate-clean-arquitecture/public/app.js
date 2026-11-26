/**
 * =============================================================================
 * AUREUM & CO. — CONTROLADOR MULTI-VISTA, MODO CLARO/OSCURO, SOCIAL FEED Y PERFIL
 *  1. Cronograma y Buscador de Subastas (vista principal limpia para el usuario)
 *  2. Sala en Vivo centrada en el usuario (paletas anónimas, sin datos de otros postores)
 *  3. Catálogo dedicado de Bienes y Consignación con fotos reales
 *  4. Social Feed interactivo respaldado en PostgreSQL (/api/v1/social/posts)
 *  5. Mi Perfil, Watchlist y Ajustes de Modo Claro / Oscuro (/api/v1/auth/me)
 *  6. Concierge flotante dinámico con Google Gemini (/api/v1/ia/asistente)
 * =============================================================================
 */

document.addEventListener('DOMContentLoaded', async () => {
  const isLocalhost = ['localhost', '127.0.0.1', '::1'].includes(window.location.hostname);
  document.querySelectorAll('.localhost-dev-only').forEach((el) => {
    el.classList.toggle('hidden', !isLocalhost);
  });

  if (!('closedBy' in HTMLDialogElement.prototype)) {
    const outside = (e, dialog) => {
      const r = dialog.getBoundingClientRect();
      return e.clientX < r.left || e.clientX > r.right || e.clientY < r.top || e.clientY > r.bottom;
    };
    for (const dialog of document.querySelectorAll('dialog[closedby="any"]')) {
      let pressedOutside = false;
      dialog.addEventListener('pointerdown', (e) => { pressedOutside = outside(e, dialog); });
      dialog.addEventListener('pointerup', (e) => { if (pressedOutside && outside(e, dialog)) dialog.close(); });
    }
  }

  inicializarValidacionAccesible();

  // ---------------------------------------------------------------------------
  // 1. ESTADO DE LA APLICACIÓN
  // ---------------------------------------------------------------------------
  const savedUser = (() => {
    try { return JSON.parse(localStorage.getItem('livebid_user') || 'null'); } catch (_) { return null; }
  })();
  const savedWatchlist = (() => {
    try { return JSON.parse(localStorage.getItem('livebid_watchlist') || '[]'); } catch (_) { return []; }
  })();

  const state = {
    vistaActiva: 'cronograma',
    subastaId: 'SUB-2026-001',
    subastas: [],
    filtroCategoria: 'TODOS',
    scheduleQuery: '',
    scheduleStatus: 'TODOS',
    watchlist: new Set(Array.isArray(savedWatchlist) ? savedWatchlist : []),
    socialPosts: [],
    subastaActual: null,
    precioActual: 45000,
    precioBase: 45000,
    incrementoMinimo: 1000,
    postorLiderId: 'NINGUNO',
    authToken: localStorage.getItem('livebid_token') || '',
    usuarioActual: savedUser,
    imagenReferencialDataUrl: '',
    nuevaSubastaImagenDataUrl: '',
    stripeIntentActual: null,
    garantiaActiva: null,
    tiempoSegundos: 120,
    activa: true,
    totalPujas: 0,
    timerInterval: null,
    ws: null,
    wsSubastaId: '',
    seenBidIds: new Set(),
    aiHistory: []
  };

  if (!state.authToken) {
    state.usuarioActual = null;
  }

  // ---------------------------------------------------------------------------
  // 2. SISTEMA DE MODO CLARO / MODO OSCURO (dark-mode + dark-mode-toggle)
  // ---------------------------------------------------------------------------
  inicializarSistemaDeTemas();

  function inicializarSistemaDeTemas() {
    const metaScheme = document.querySelector('meta[name="color-scheme"]');
    const quickToggleBtn = document.getElementById('btnThemeToggle');
    const radios = document.querySelectorAll('input[name="themePreference"]');
    const systemDarkQuery = window.matchMedia('(prefers-color-scheme: dark)');

    function obtenerEsquemaRenderizadoActual() {
      const stored = localStorage.getItem('color-scheme');
      if (stored === 'light' || stored === 'dark') return stored;
      return systemDarkQuery.matches ? 'dark' : 'light';
    }

    function aplicarPreferenciaTema(valor) {
      if (!metaScheme) return;
      if (valor === 'light' || valor === 'dark') {
        metaScheme.content = valor;
        try { localStorage.setItem('color-scheme', valor); } catch (_) {}
      } else {
        metaScheme.content = 'light dark';
        try { localStorage.removeItem('color-scheme'); } catch (_) {}
      }
      sincronizarControlesTema();
    }

    function sincronizarControlesTema() {
      const stored = localStorage.getItem('color-scheme') || 'light dark';
      radios.forEach((r) => { r.checked = r.value === stored; });
      const label = document.getElementById('themeToggleLabel');
      if (label) {
        label.textContent = obtenerEsquemaRenderizadoActual() === 'dark' ? 'Modo Claro' : 'Modo Oscuro';
      }
    }

    // Regla dark-mode-toggle: el botón de 2 estados invierte lo que se ve actualmente
    // y elimina localStorage si el objetivo coincide con la preferencia del sistema.
    quickToggleBtn?.addEventListener('click', () => {
      const actual = obtenerEsquemaRenderizadoActual();
      const objetivo = actual === 'dark' ? 'light' : 'dark';
      const sistema = systemDarkQuery.matches ? 'dark' : 'light';
      if (objetivo === sistema) {
        aplicarPreferenciaTema('light dark');
      } else {
        aplicarPreferenciaTema(objetivo);
      }
    });

    radios.forEach((r) => {
      r.addEventListener('change', () => {
        if (r.checked) aplicarPreferenciaTema(r.value);
      });
    });

    // MANDATORY (dark-mode): escuchar cambios dinámicos del sistema y de otras pestañas
    systemDarkQuery.addEventListener('change', sincronizarControlesTema);
    window.addEventListener('storage', (ev) => {
      if (ev.key === 'color-scheme' && metaScheme) {
        metaScheme.content = ev.newValue ?? 'light dark';
        sincronizarControlesTema();
      }
    });

    sincronizarControlesTema();
  }

  // ---------------------------------------------------------------------------
  // 3. NAVEGACIÓN ABIERTA ENTRE LAS 5 SECCIONES DE LA APP
  // ---------------------------------------------------------------------------
  const vistasMap = {
    cronograma: document.getElementById('viewCronograma'),
    sala: document.getElementById('viewSala'),
    catalogo: document.getElementById('viewCatalogo'),
    social: document.getElementById('viewSocial'),
    perfil: document.getElementById('viewPerfil')
  };

  function cambiarVistaActiva(nombreVista) {
    if (!vistasMap[nombreVista]) return;
    state.vistaActiva = nombreVista;

    Object.entries(vistasMap).forEach(([key, el]) => {
      if (el) el.classList.toggle('hidden', key !== nombreVista);
    });

    document.querySelectorAll('[data-nav-view]').forEach((btn) => {
      if (btn.getAttribute('data-nav-view') === nombreVista) {
        btn.setAttribute('aria-current', 'page');
      } else {
        btn.removeAttribute('aria-current');
      }
    });

    if (nombreVista === 'social') {
      cargarSocialFeed();
    } else if (nombreVista === 'perfil') {
      renderizarVistaPerfil();
    }
    window.scrollTo({ top: 0, behavior: 'smooth' });
  }

  document.querySelectorAll('[data-nav-view]').forEach((btn) => {
    btn.addEventListener('click', () => cambiarVistaActiva(btn.getAttribute('data-nav-view')));
  });

  document.querySelectorAll('[data-go-view]').forEach((el) => {
    el.addEventListener('click', (ev) => {
      ev.preventDefault();
      cambiarVistaActiva(el.getAttribute('data-go-view'));
    });
  });

  document.getElementById('btnGoLiveFromHero')?.addEventListener('click', () => {
    cambiarVistaActiva('sala');
  });

  // ---------------------------------------------------------------------------
  // 4. REFERENCIAS DOM Y GESTIÓN DE USUARIO / PERFIL
  // ---------------------------------------------------------------------------
  const elAuctionSelector = document.getElementById('auctionSelector');
  const elTopAuctionStateLabel = document.getElementById('topAuctionStateLabel');
  const elNavUserName = document.getElementById('navUserName');
  const elNavUserRoleBadge = document.getElementById('navUserRoleBadge');
  const elUserInitials = document.getElementById('userInitials');
  const elAuthBtnLabel = document.getElementById('authBtnLabel');
  const elEscrowBadge = document.getElementById('escrowBadge');
  const elBtnLogoutUser = document.getElementById('btnLogoutUser');
  const elFeedbackToast = document.getElementById('feedbackToast');

  const authDialog = document.getElementById('authDialog');
  const lotManagerDialog = document.getElementById('lotManagerDialog');
  const adminDialog = document.getElementById('adminDialog');
  const stripeDialog = document.getElementById('stripeDialog');

  const assistantDrawerApi = inicializarDrawerAsistente();

  actualizarCabeceraUsuario();
  await cargarCatalogoSubastas();
  await sincronizarEstadoInicial();
  await cargarSocialFeed();

  function obtenerRolUsuario() {
    if (!state.authToken || !state.usuarioActual) return 'VISITANTE';
    return state.usuarioActual.role || state.usuarioActual.rol || 'POSTOR';
  }

  function obtenerPaletaAnonima(idRaw) {
    if (!idRaw || idRaw === 'NINGUNO') return '';
    let hash = 0;
    const s = String(idRaw);
    for (let i = 0; i < s.length; i++) hash = (hash * 31 + s.charCodeAt(i)) % 900;
    return `Paleta #${100 + Math.abs(hash)}`;
  }

  function textoLiderCentradoEnUsuario() {
    if (!state.postorLiderId || state.postorLiderId === 'NINGUNO') {
      return 'Aún sin ofertas — puedes abrir la subasta';
    }
    if (state.usuarioActual?.id && state.usuarioActual.id === state.postorLiderId) {
      return '✓ ¡Tú tienes la oferta más alta en este momento!';
    }
    return `Oferta líder registrada (${obtenerPaletaAnonima(state.postorLiderId)})`;
  }

  function aplicarPermisosVisualesPorRol() {
    const autenticado = Boolean(state.authToken && state.usuarioActual);
    const rol = obtenerRolUsuario();

    const esAdmin = autenticado && rol === 'ADMIN';
    const puedeGestionarLote = autenticado && ['VENDEDOR', 'MARTILLERO', 'ADMIN'].includes(rol);
    const puedeUsarMartillo = autenticado && ['MARTILLERO', 'ADMIN'].includes(rol);

    document.getElementById('btnOpenAdminModal')?.classList.toggle('hidden', !esAdmin);
    document.getElementById('btnOpenLotModal')?.classList.toggle('hidden', !puedeGestionarLote);
    document.getElementById('btnQuickUploadImage')?.classList.toggle('hidden', !puedeGestionarLote);
    document.getElementById('btnEditEncargadoQuick')?.classList.toggle('hidden', !puedeGestionarLote);
    document.getElementById('gavelConsoleSection')?.classList.toggle('hidden', !puedeUsarMartillo);
    elBtnLogoutUser?.classList.toggle('hidden', !autenticado);

    const socialHint = document.getElementById('socialComposerAuthHint');
    if (socialHint) {
      socialHint.textContent = autenticado
        ? `Publicando como ${state.usuarioActual.name} (${rol})`
        : 'Inicia sesión para publicar novedades y comentar.';
    }
  }

  function actualizarGuiaPersonalEnSala() {
    const autenticado = Boolean(state.authToken && state.usuarioActual);
    const tieneGarantia = Boolean(state.garantiaActiva && state.garantiaActiva.estado === 'RETENIDA');
    const reqDep = Math.round((state.precioBase || 45000) * 0.1);
    const sigMin = (state.precioActual || 45000) + state.incrementoMinimo;

    const stepAuth = document.getElementById('stepAuthItem');
    const stepGar = document.getElementById('stepGuaranteeItem');
    const stepGarTxt = document.getElementById('stepGuaranteeText');
    const stepNextMin = document.getElementById('stepNextMinText');

    if (stepAuth) {
      stepAuth.classList.toggle('is-done', autenticado);
      stepAuth.querySelector('.step-bullet').textContent = autenticado ? '✓' : '1';
    }
    if (stepGar) {
      stepGar.classList.toggle('is-done', tieneGarantia);
      stepGar.querySelector('.step-bullet').textContent = tieneGarantia ? '✓' : '2';
    }
    if (stepGarTxt) {
      stepGarTxt.textContent = tieneGarantia
        ? `Garantía activa y reservada por S/. ${formatearNumero(state.garantiaActiva.monto)}.`
        : `Reserva temporalmente S/. ${formatearNumero(reqDep)} (10% reembolsable si no ganas).`;
    }
    if (stepNextMin) {
      stepNextMin.textContent = `S/. ${formatearNumero(sigMin)}`;
    }

    const watchBtn = document.getElementById('btnToggleWatchCurrent');
    if (watchBtn) {
      const siguiendo = state.watchlist.has(state.subastaId);
      watchBtn.textContent = siguiendo ? '⭐ Guardada en Mi Perfil' : '⭐ Guardar';
    }
  }

  function actualizarCabeceraUsuario() {
    const autenticado = Boolean(state.authToken && state.usuarioActual);
    if (autenticado) {
      const u = state.usuarioActual;
      const nombre = u.name || u.nombre || 'Usuario';
      const rol = obtenerRolUsuario();

      elNavUserName.textContent = nombre;
      elNavUserRoleBadge.textContent = rol;
      elUserInitials.textContent = inicialesDeNombre(nombre);
      elAuthBtnLabel.textContent = 'Cambiar Cuenta';

      if (state.garantiaActiva && state.garantiaActiva.estado === 'RETENIDA') {
        elEscrowBadge.textContent = `💳 Garantía: S/. ${formatearNumero(state.garantiaActiva.monto)} (RETENIDA)`;
      } else {
        elEscrowBadge.textContent = 'Ir a Mi Perfil y Ajustes';
      }
    } else {
      elNavUserName.textContent = 'Visitante';
      elNavUserRoleBadge.textContent = 'EXPLORANDO';
      elUserInitials.textContent = 'VS';
      elAuthBtnLabel.textContent = 'Ingresar / Crear Cuenta';
      elEscrowBadge.textContent = 'Ver mi perfil y actividad';
    }

    aplicarPermisosVisualesPorRol();
    actualizarGuiaPersonalEnSala();
    actualizarContextoAsistente();
  }

  document.getElementById('userProfileBadge')?.addEventListener('click', () => {
    cambiarVistaActiva('perfil');
  });

  elBtnLogoutUser?.addEventListener('click', () => {
    state.authToken = '';
    state.usuarioActual = null;
    state.garantiaActiva = null;
    localStorage.removeItem('livebid_token');
    localStorage.removeItem('livebid_user');
    if (state.ws) {
      try { state.ws.close(); } catch (_) {}
      state.ws = null;
    }
    actualizarCabeceraUsuario();
    renderizarVistaPerfil();
    mostrarToast('Has cerrado tu sesión.', 'success');
  });

  function guardarSesion(token, user) {
    state.authToken = token;
    state.usuarioActual = user;
    localStorage.setItem('livebid_token', token);
    localStorage.setItem('livebid_user', JSON.stringify(user));
    actualizarCabeceraUsuario();
    sincronizarGarantiaActivaBackend();
    conectarWebSocketSala();
    renderizarVistaPerfil();
  }

  // ---------------------------------------------------------------------------
  // 5. WATCHLIST (GUARDAR RECORDATORIOS EN PERFIL)
  // ---------------------------------------------------------------------------
  function alternarSeguimientoSubasta(id) {
    if (!id) return;
    if (state.watchlist.has(id)) {
      state.watchlist.delete(id);
      mostrarToast('Subasta retirada de tus guardados.', 'success');
    } else {
      state.watchlist.add(id);
      mostrarToast('⭐ Subasta guardada en Mi Perfil y Ajustes.', 'success');
    }
    try {
      localStorage.setItem('livebid_watchlist', JSON.stringify([...state.watchlist]));
    } catch (_) {}
    renderizarCronograma();
    renderizarVitrinaCatalogo();
    actualizarGuiaPersonalEnSala();
    if (state.vistaActiva === 'perfil') renderizarVistaPerfil();
  }

  document.getElementById('btnToggleWatchCurrent')?.addEventListener('click', () => {
    alternarSeguimientoSubasta(state.subastaId);
  });

  // ---------------------------------------------------------------------------
  // 6. CRONOGRAMA Y CATÁLOGO DE BIENES (POSTGRESQL)
  // ---------------------------------------------------------------------------
  function iconoPorCategoria(catRaw) {
    const c = String(catRaw || '').toUpperCase();
    if (c.includes('VEHIC') || c.includes('AUTO') || c.includes('4X4')) return '🚘';
    if (c.includes('RELOJ') || c.includes('JOYA') || c.includes('ORO')) return '⌚';
    if (c.includes('ARTE') || c.includes('ANTIG') || c.includes('PINT')) return '🖼️';
    if (c.includes('INMUEB') || c.includes('CASA') || c.includes('PROP')) return '🏛️';
    return '💎';
  }

  function formatearCategoriaHumana(catRaw) {
    if (!catRaw) return 'Lote Especial';
    return String(catRaw)
      .replace(/_/g, ' ')
      .toLowerCase()
      .replace(/\b\w/g, (l) => l.toUpperCase());
  }

  function formatearFechaCorta(iso) {
    if (!iso) return 'Horario continuo';
    const d = new Date(iso);
    if (Number.isNaN(d.getTime())) return 'Horario continuo';
    return d.toLocaleString('es-PE', { day: '2-digit', month: 'short', hour: '2-digit', minute: '2-digit' });
  }

  async function irASalaDeBien(id) {
    state.subastaId = id;
    if (elAuctionSelector) elAuctionSelector.value = id;
    cambiarVistaActiva('sala');
    await sincronizarEstadoInicial();
  }

  function renderizarCronograma() {
    const grid = document.getElementById('scheduleGrid');
    if (!grid) return;
    grid.innerHTML = '';

    const q = state.scheduleQuery.trim().toLowerCase();
    const filtradas = state.subastas.filter((s) => {
      const estado = s.estado || 'ACTIVA';
      if (state.scheduleStatus === 'EN_VIVO' && estado === 'CERRADA') return false;
      if (state.scheduleStatus === 'CERRADAS' && estado !== 'CERRADA') return false;
      if (!q) return true;
      const texto = `${s.titulo || ''} ${s.descripcion || ''} ${s.detalles_lote?.categoria || ''} ${s.detalles_lote?.ubicacion || ''}`.toLowerCase();
      return texto.includes(q);
    });

    if (filtradas.length === 0) {
      const empty = document.createElement('div');
      empty.className = 'card';
      empty.textContent = 'No se encontraron subastas con ese criterio en el cronograma.';
      grid.appendChild(empty);
      return;
    }

    filtradas.forEach((s) => {
      const det = s.detalles_lote || {};
      const imgUrl = Array.isArray(det.imagenes_urls) && det.imagenes_urls.length > 0 ? det.imagenes_urls[0] : '';
      const estado = s.estado || 'ACTIVA';
      const precio = s.precio_actual ?? s.precio_base ?? 0;
      const siguiendo = state.watchlist.has(s.id);

      const card = document.createElement('article');
      card.className = 'schedule-card';

      const timeBanner = document.createElement('div');
      timeBanner.className = 'schedule-time-banner';
      const estadoSpan = document.createElement('span');
      estadoSpan.className = `card-state-pill ${estado === 'CERRADA' ? 'closed' : 'live'}`;
      estadoSpan.textContent = estado === 'CERRADA' ? '🏁 Finalizada' : '🟢 En Vivo';
      const cierreSpan = document.createElement('span');
      cierreSpan.textContent = `Cierre: ${formatearFechaCorta(s.cierre)}`;
      timeBanner.append(estadoSpan, cierreSpan);
      card.appendChild(timeBanner);

      if (imgUrl) {
        const img = document.createElement('img');
        img.src = imgUrl;
        img.alt = s.titulo || 'Bien programado';
        img.loading = 'lazy';
        card.appendChild(img);
      }

      const body = document.createElement('div');
      body.className = 'schedule-meta-list';
      const h3 = document.createElement('h3');
      h3.textContent = `${iconoPorCategoria(det.categoria || s.titulo)} ${s.titulo}`;
      const loc = document.createElement('span');
      loc.textContent = `${formatearCategoriaHumana(det.categoria)} • ${det.ubicacion || 'Exhibición Lima'}`;
      const pr = document.createElement('strong');
      pr.style.color = 'var(--accent-gold)';
      pr.textContent = `${estado === 'CERRADA' ? 'Adjudicado en' : 'Oferta actual'}: S/. ${formatearNumero(precio)}`;
      body.append(h3, loc, pr);

      const footer = document.createElement('div');
      footer.className = 'schedule-card-footer';

      const watchBtn = document.createElement('button');
      watchBtn.type = 'button';
      watchBtn.className = `btn-watch-item${siguiendo ? ' is-watched' : ''}`;
      watchBtn.textContent = siguiendo ? '⭐ Guardado' : '☆ Recordar';
      watchBtn.addEventListener('click', () => alternarSeguimientoSubasta(s.id));

      const enterBtn = document.createElement('button');
      enterBtn.type = 'button';
      enterBtn.className = 'btn-enter-lot';
      enterBtn.textContent = estado === 'CERRADA' ? 'Ver Resultado' : 'Entrar a Sala';
      enterBtn.addEventListener('click', () => irASalaDeBien(s.id));

      footer.append(watchBtn, enterBtn);
      card.append(body, footer);
      grid.appendChild(card);
    });
  }

  document.getElementById('scheduleSearchInput')?.addEventListener('input', (e) => {
    state.scheduleQuery = e.target.value;
    renderizarCronograma();
  });

  document.getElementById('scheduleStatusSelect')?.addEventListener('change', (e) => {
    state.scheduleStatus = e.target.value;
    renderizarCronograma();
  });

  function coincideFiltroCategoria(subasta, filtro) {
    if (!filtro || filtro === 'TODOS') return true;
    const texto = `${subasta?.detalles_lote?.categoria || ''} ${subasta?.titulo || ''}`.toUpperCase();
    switch (filtro) {
      case 'VEHICULOS': return /VEHIC|AUTO|4X4|HILUX|PORSCHE/.test(texto);
      case 'JOYAS': return /RELOJ|JOYA|ROLEX|PATEK|ORO/.test(texto);
      case 'ARTE': return /ARTE|COLECC|ANTIG|VALOR/.test(texto);
      case 'INMUEBLES': return /INMUEB|PENTHOUSE|CASA|TERRENO/.test(texto);
      default: return true;
    }
  }

  function renderizarVitrinaCatalogo() {
    const grid = document.getElementById('catalogGrid');
    const countBadge = document.getElementById('catalogCountBadge');
    if (countBadge) countBadge.textContent = String(state.subastas.length);
    if (!grid) return;

    const filtradas = state.subastas.filter((s) => coincideFiltroCategoria(s, state.filtroCategoria));
    grid.innerHTML = '';

    if (filtradas.length === 0) {
      const empty = document.createElement('article');
      empty.className = 'catalog-card';
      const h = document.createElement('header');
      const h3 = document.createElement('h3');
      h3.textContent = 'Sin bienes en esta categoría';
      h.appendChild(h3);
      empty.appendChild(h);
      grid.appendChild(empty);
      return;
    }

    filtradas.forEach((s) => {
      const det = s.detalles_lote || {};
      const imgUrl = Array.isArray(det.imagenes_urls) && det.imagenes_urls.length > 0 ? det.imagenes_urls[0] : '';
      const estado = s.estado || 'ACTIVA';
      const precio = s.precio_actual ?? s.precio_base ?? 0;

      const card = document.createElement('article');
      card.className = `catalog-card${s.id === state.subastaId ? ' is-selected' : ''}`;

      if (imgUrl) {
        const img = document.createElement('img');
        img.className = 'catalog-card-img';
        img.src = imgUrl;
        img.alt = s.titulo || 'Bien en catálogo';
        img.loading = 'lazy';
        card.appendChild(img);
      } else {
        const media = document.createElement('div');
        media.className = 'media';
        media.setAttribute('aria-hidden', 'true');
        const em = document.createElement('span');
        em.className = 'media-emoji';
        em.textContent = iconoPorCategoria(det.categoria || s.titulo);
        const cap = document.createElement('span');
        cap.className = 'media-caption';
        cap.textContent = det.ubicacion || 'Sala Oficial';
        media.append(em, cap);
        card.appendChild(media);
      }

      const header = document.createElement('header');
      const badges = document.createElement('div');
      badges.className = 'card-badges-row';
      const cat = document.createElement('span');
      cat.className = 'card-cat-tag';
      cat.textContent = formatearCategoriaHumana(det.categoria);
      const pill = document.createElement('span');
      pill.className = `card-state-pill ${estado === 'CERRADA' ? 'closed' : 'live'}`;
      pill.textContent = estado === 'CERRADA' ? 'Finalizada' : 'En vivo';
      badges.append(cat, pill);
      const h3 = document.createElement('h3');
      h3.textContent = s.titulo;
      header.append(badges, h3);

      const details = document.createElement('div');
      details.className = 'catalog-card-details';
      const desc = document.createElement('p');
      desc.textContent = (s.descripcion || 'Bien verificado para subasta.').slice(0, 95);
      const priceRow = document.createElement('div');
      priceRow.className = 'catalog-card-price-row';
      const lbl = document.createElement('span');
      lbl.textContent = 'Precio actual';
      const val = document.createElement('strong');
      val.className = 'catalog-card-price-val';
      val.textContent = `S/. ${formatearNumero(precio)}`;
      priceRow.append(lbl, val);
      details.append(desc, priceRow);

      const footer = document.createElement('footer');
      const btn = document.createElement('button');
      btn.type = 'button';
      btn.className = 'btn-enter-lot';
      btn.textContent = 'Participar en Sala';
      btn.addEventListener('click', () => irASalaDeBien(s.id));
      footer.appendChild(btn);

      card.append(header, details, footer);
      grid.appendChild(card);
    });
  }

  document.querySelectorAll('[data-cat-filter]').forEach((chip) => {
    chip.addEventListener('click', () => {
      document.querySelectorAll('[data-cat-filter]').forEach((c) => c.classList.remove('active'));
      chip.classList.add('active');
      state.filtroCategoria = chip.getAttribute('data-cat-filter') || 'TODOS';
      renderizarVitrinaCatalogo();
    });
  });

  async function cargarCatalogoSubastas() {
    try {
      const res = await fetch('/api/v1/subastas');
      if (!res.ok) return;
      const lista = await res.json();
      if (!Array.isArray(lista) || lista.length === 0) return;

      state.subastas = lista;
      if (!lista.some((s) => s.id === state.subastaId)) {
        state.subastaId = lista[0].id;
      }

      elAuctionSelector.innerHTML = '';
      const socialAuctionSelect = document.getElementById('socialPostAuctionSelect');
      if (socialAuctionSelect) {
        socialAuctionSelect.innerHTML = '<option value="">Sin vínculo a lote</option>';
      }

      lista.forEach((s) => {
        const opt = document.createElement('option');
        opt.value = s.id;
        opt.textContent = `${s.titulo.slice(0, 34)} (${s.estado === 'CERRADA' ? 'Cerrada' : 'En vivo'})`;
        if (s.id === state.subastaId) opt.selected = true;
        elAuctionSelector.appendChild(opt);

        if (socialAuctionSelect) {
          const sOpt = document.createElement('option');
          sOpt.value = s.id;
          sOpt.textContent = s.titulo.slice(0, 40);
          socialAuctionSelect.appendChild(sOpt);
        }
      });

      renderizarCronograma();
      renderizarVitrinaCatalogo();
    } catch (_) {}
  }

  elAuctionSelector.addEventListener('change', async (e) => {
    await irASalaDeBien(e.target.value);
  });

  // ---------------------------------------------------------------------------
  // 7. SALA EN VIVO, WEBSOCKET Y OFERTAS CON PALETAS ANÓNIMAS
  // ---------------------------------------------------------------------------
  function conectarWebSocketSala() {
    if (!state.authToken || !state.subastaId) return;
    if (state.ws && state.wsSubastaId === state.subastaId && state.ws.readyState <= WebSocket.OPEN) return;
    if (state.ws) {
      try { state.ws.close(); } catch (_) {}
      state.ws = null;
    }
    const wsProto = window.location.protocol === 'https:' ? 'wss:' : 'ws:';
    const wsUrl = `${wsProto}//${window.location.host}/ws/subastas/${encodeURIComponent(state.subastaId)}?token=${encodeURIComponent(state.authToken)}`;
    try {
      const ws = new WebSocket(wsUrl);
      state.ws = ws;
      state.wsSubastaId = state.subastaId;
      ws.addEventListener('message', (msgEvent) => {
        try {
          const ev = JSON.parse(msgEvent.data);
          if (ev.type === 'bid.accepted' && ev.auction_id === state.subastaId) {
            if (ev.bid_id && state.seenBidIds.has(ev.bid_id)) return;
            if (ev.bid_id) state.seenBidIds.add(ev.bid_id);
            state.precioActual = Number(ev.amount);
            state.postorLiderId = ev.bidder_id;
            state.totalPujas++;
            const etiqueta = state.usuarioActual?.id === ev.bidder_id ? 'Tu oferta' : obtenerPaletaAnonima(ev.bidder_id);
            actualizarPreciosUI();
            if (ev.extended) dispararReglaAntiSniping();
            agregarPujaAlTimeline(Number(ev.amount), etiqueta);
            renderizarCronograma();
            renderizarVitrinaCatalogo();
          } else if (ev.type === 'auction.closed' && ev.auction_id === state.subastaId) {
            if (ev.winner_id) {
              state.postorLiderId = ev.winner_id;
              state.precioActual = Number(ev.amount || state.precioActual);
            }
            finalizarSubastaPorTiempo();
          }
        } catch (_) {}
      });
    } catch (_) {}
  }

  async function sincronizarEstadoInicial() {
    state.seenBidIds.clear();
    try {
      const res = await fetch(`/api/v1/subastas/${state.subastaId}`);
      if (res.ok) {
        const data = await res.json();
        state.subastaActual = data;
        renderizarSubastaCompleta(data);
      }
      const histRes = await fetch(`/api/v1/subastas/${state.subastaId}/pujas?page=1&per_page=40`);
      if (histRes.ok) {
        const hist = await histRes.json();
        const items = Array.isArray(hist.items) ? hist.items : [];
        state.totalPujas = hist.total ?? items.length;
        renderizarHistorialAnonimo(items, state.precioBase);
        actualizarPreciosUI();
      }
    } catch (_) {
      actualizarPreciosUI();
    }
    await sincronizarGarantiaActivaBackend();
    conectarWebSocketSala();
    actualizarGuiaPersonalEnSala();
    actualizarContextoAsistente();
  }

  function renderizarHistorialAnonimo(items, precioBase) {
    const timeline = document.getElementById('bidsTimeline');
    if (!timeline) return;
    timeline.innerHTML = '';

    if (!Array.isArray(items) || items.length === 0) {
      timeline.innerHTML = `
        <div class="bid-item winner-bid">
          <div class="bid-correlative">#0</div>
          <div class="bid-details">
            <div class="bid-row-top">
              <span class="bid-amount">S/. ${formatearNumero(precioBase)}</span>
              <span class="bid-time">Apertura</span>
            </div>
            <div class="bid-row-bottom">
              <span>Precio Base de Salida</span>
              <span class="audit-tag">Abierto</span>
            </div>
          </div>
        </div>
      `;
      return;
    }

    items.forEach((b, idx) => {
      if (b.id) state.seenBidIds.add(b.id);
      const esMio = state.usuarioActual?.id && b.postor_id === state.usuarioActual.id;
      const etiqueta = esMio ? 'Tu oferta' : obtenerPaletaAnonima(b.postor_id || idx);
      const correlativo = Math.max(1, (state.totalPujas || items.length) - idx);
      const row = document.createElement('div');
      row.className = `bid-item${idx === 0 ? ' winner-bid' : ''}`;
      row.innerHTML = `
        <div class="bid-correlative">#${correlativo}</div>
        <div class="bid-details">
          <div class="bid-row-top">
            <span class="bid-amount">S/. ${formatearNumero(b.monto)}</span>
            <span class="bid-time">${b.recibida_en ? new Date(b.recibida_en).toLocaleTimeString('es-PE', { hour: '2-digit', minute: '2-digit' }) : 'En sala'}</span>
          </div>
          <div class="bid-row-bottom">
            <span>${escaparHTML(etiqueta)}</span>
            <span class="audit-tag">✓ Registrada</span>
          </div>
        </div>
      `;
      timeline.appendChild(row);
    });
  }

  function renderizarSubastaCompleta(data) {
    state.precioBase = data.precio_base ?? 45000;
    state.precioActual = data.precio_actual ?? state.precioBase;
    state.postorLiderId = data.postor_lider_id || 'NINGUNO';

    const estado = data.estado || 'ACTIVA';
    const phaseTag = document.getElementById('auctionPhaseTag');
    if (estado === 'CERRADA') {
      state.activa = false;
      state.tiempoSegundos = 0;
      elTopAuctionStateLabel.textContent = 'SALA CERRADA';
      if (phaseTag) phaseTag.textContent = 'ESTRADO CERRADO • LOTE ADJUDICADO';
    } else if (estado === 'EN_CONTEO') {
      state.activa = true;
      if (state.tiempoSegundos > 15 || state.tiempoSegundos <= 0) state.tiempoSegundos = 15;
      elTopAuctionStateLabel.textContent = 'ÚLTIMOS LLAMADOS';
      if (phaseTag) phaseTag.textContent = '🔔 ¡A LA UNA, A LAS DOS!';
    } else {
      state.activa = true;
      if (data.cierre) {
        const diffSec = Math.floor((new Date(data.cierre).getTime() - Date.now()) / 1000);
        state.tiempoSegundos = diffSec > 0 ? Math.min(diffSec, 5999) : 120;
      } else {
        state.tiempoSegundos = 120;
      }
      elTopAuctionStateLabel.textContent = 'SALA EN VIVO';
      if (phaseTag) phaseTag.textContent = 'ESTRADO ABIERTO • PUJAS EN VIVO';
    }

    document.getElementById('lotIdTag').textContent = `LOTE OFICIAL #${data.id}`;
    document.getElementById('lotTitle').textContent = data.titulo || 'Bien en Subasta';
    document.getElementById('lotDescription').textContent = data.descripcion || 'Bien verificado para subasta en tiempo real.';

    const det = data.detalles_lote || {};
    document.getElementById('lotCategoryBadge').textContent = formatearCategoriaHumana(det.categoria || 'Catálogo').toUpperCase();
    document.getElementById('lotLocationBadge').textContent = det.ubicacion || 'LIMA - PERÚ';
    document.getElementById('lotInspectorBadge').textContent = det.inspeccionado_por ? `★ ${det.inspeccionado_por}` : '★ BIEN VERIFICADO';
    document.getElementById('defaultLotEmoji').textContent = iconoPorCategoria(det.categoria || data.titulo);

    const mainImg = document.getElementById('mainLotImage');
    const iconStage = document.getElementById('defaultLotIconStage');
    const imgUrl = Array.isArray(det.imagenes_urls) && det.imagenes_urls.length > 0 ? det.imagenes_urls[0] : '';
    if (imgUrl) {
      mainImg.src = imgUrl;
      mainImg.classList.remove('hidden');
      iconStage.classList.add('hidden');
    } else {
      mainImg.classList.add('hidden');
      iconStage.classList.remove('hidden');
    }

    const specsGrid = document.getElementById('lotSpecsGrid');
    if (specsGrid) {
      specsGrid.innerHTML = `
        <div class="spec-box">
          <span class="spec-label">PRECIO DE APERTURA</span>
          <span class="spec-value highlight">S/. ${formatearNumero(state.precioBase)}</span>
        </div>
        <div class="spec-box">
          <span class="spec-label">VALOR DE RESERVA</span>
          <span class="spec-value">S/. ${formatearNumero(det.precio_reserva || state.precioBase)}</span>
        </div>
        <div class="spec-box">
          <span class="spec-label">EXHIBICIÓN</span>
          <span class="spec-value">${escaparHTML(det.ubicacion || 'Sala Lima')}</span>
        </div>
        <div class="spec-box">
          <span class="spec-label">VERIFICACIÓN</span>
          <span class="spec-value">${escaparHTML(det.inspeccionado_por || 'Especialista Aureum')}</span>
        </div>
      `;
    }

    const enc = data.encargado || {};
    document.getElementById('encargadoNombre').textContent = enc.nombre || 'Dr. Carlos Mendoza Ríos';
    document.getElementById('encargadoCargo').textContent = enc.cargo || 'Martillero Público Oficial';
    document.getElementById('encargadoLicencia').textContent = enc.licencia ? `Lic. ${enc.licencia}` : 'Acreditado';
    document.getElementById('encargadoContacto').textContent = enc.contacto ? `📧 ${enc.contacto}` : '';
    document.getElementById('encargadoExperiencia').textContent = enc.experiencia || '';

    document.getElementById('detCategoria').value = det.categoria || 'VEHICULOS_COLECCION';
    document.getElementById('detUbicacion').value = det.ubicacion || 'San Isidro, Lima';
    document.getElementById('detReserva').value = det.precio_reserva || state.precioBase;
    document.getElementById('detInspector').value = det.inspeccionado_por || 'Ing. Roberto Salazar';
    document.getElementById('detCertUrl').value = det.certificado_peritaje_url || '';

    document.getElementById('encNombre').value = enc.nombre || 'Dr. Carlos Mendoza Ríos';
    document.getElementById('encMartilleroId').value = enc.martillero_id || 'MART-01';
    document.getElementById('encCargo').value = enc.cargo || 'Martillero Público Oficial';
    document.getElementById('encLicencia').value = enc.licencia || 'REG-SUNARP-2026-0412';
    document.getElementById('encContacto').value = enc.contacto || 'cmendoza@livebid.pe';
    document.getElementById('encExperiencia').value = enc.experiencia || '';

    renderizarRelojInstantaneo();
    actualizarPreciosUI();
  }

  function renderizarRelojInstantaneo() {
    const m = Math.floor(Math.max(0, state.tiempoSegundos) / 60);
    const s = Math.max(0, state.tiempoSegundos) % 60;
    document.getElementById('minutes').textContent = String(m).padStart(2, '0');
    document.getElementById('seconds').textContent = String(s).padStart(2, '0');
  }

  state.timerInterval = setInterval(() => {
    if (!state.activa) return;
    if (state.tiempoSegundos > 0) {
      state.tiempoSegundos--;
      renderizarRelojInstantaneo();
    } else {
      finalizarSubastaPorTiempo();
    }
  }, 1000);

  function dispararReglaAntiSniping() {
    if (state.tiempoSegundos <= 15) {
      state.tiempoSegundos += 30;
      renderizarRelojInstantaneo();
      const banner = document.getElementById('antiSnipingBanner');
      banner?.classList.remove('hidden');
      setTimeout(() => banner?.classList.add('hidden'), 5000);
    }
  }

  function finalizarSubastaPorTiempo() {
    state.activa = false;
    document.getElementById('minutes').textContent = '00';
    document.getElementById('seconds').textContent = '00';
    document.getElementById('auctionPhaseTag').textContent = 'ESTRADO CERRADO • LOTE ADJUDICADO';
    elTopAuctionStateLabel.textContent = 'SALA CERRADA';
  }

  function actualizarPreciosUI() {
    document.getElementById('currentPriceValue').textContent = formatearNumero(state.precioActual);
    document.getElementById('currentLeaderName').textContent = textoLiderCentradoEnUsuario();
    document.getElementById('totalBidsCount').textContent = `${state.totalPujas} oferta${state.totalPujas === 1 ? '' : 's'}`;

    const bidInput = document.getElementById('bidInput');
    const sigMin = state.precioActual + state.incrementoMinimo;
    if (bidInput) {
      bidInput.min = sigMin;
      bidInput.value = sigMin;
    }
    actualizarGuiaPersonalEnSala();
    actualizarContextoAsistente();
  }

  document.querySelectorAll('.btn-quick').forEach((btn) => {
    btn.addEventListener('click', () => {
      const inc = Number(btn.getAttribute('data-add'));
      const bidInput = document.getElementById('bidInput');
      if (bidInput) {
        bidInput.value = state.precioActual + inc;
        bidInput.focus();
      }
    });
  });

  document.getElementById('btnEmitirPuja')?.addEventListener('click', async () => {
    if (!state.authToken || !state.usuarioActual) {
      mostrarToast('Inicia sesión para confirmar tu oferta en sala.', 'error');
      cambiarTabAuth('login', false);
      authDialog.showModal();
      return;
    }

    const bidInput = document.getElementById('bidInput');
    const monto = Number(bidInput.value);
    if (monto < state.precioActual + state.incrementoMinimo) {
      mostrarToast(`Tu oferta mínima actual es S/. ${formatearNumero(state.precioActual + state.incrementoMinimo)}.`, 'error');
      return;
    }

    if (state.usuarioActual.id && state.usuarioActual.id === state.postorLiderId) {
      mostrarToast('Ya tienes la oferta más alta en este momento.', 'error');
      return;
    }

    const btn = document.getElementById('btnEmitirPuja');
    btn.disabled = true;
    try {
      const res = await fetch(`/api/v1/subastas/${state.subastaId}/pujas`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          'Idempotency-Key': `idem-${state.subastaId}-${state.usuarioActual.id}-${monto}-${Date.now()}`,
          Authorization: `Bearer ${state.authToken}`
        },
        body: JSON.stringify({ monto })
      });
      const data = await res.json().catch(() => ({}));
      if (res.ok) {
        if (data.puja_id) state.seenBidIds.add(data.puja_id);
        state.precioActual = monto;
        state.postorLiderId = state.usuarioActual.id;
        state.totalPujas++;
        actualizarPreciosUI();
        dispararReglaAntiSniping();
        agregarPujaAlTimeline(monto, 'Tu oferta');
        mostrarToast(`✓ ¡Tu oferta de S/. ${formatearNumero(monto)} lidera la sala!`, 'success');
      } else if (res.status === 409 && String(data.error || '').toLowerCase().includes('guarantee')) {
        mostrarToast('Primero activa tu garantía reembolsable para ofertar en este bien.', 'error');
        abrirModalStripe();
      } else {
        mostrarToast(`No se pudo registrar la oferta: ${data.error || res.statusText}`, 'error');
      }
    } catch (_) {
      mostrarToast('Error de conexión al enviar tu oferta.', 'error');
    } finally {
      btn.disabled = false;
    }
  });

  function agregarPujaAlTimeline(monto, etiqueta) {
    const timeline = document.getElementById('bidsTimeline');
    if (!timeline) return;
    const item = document.createElement('div');
    item.className = 'bid-item winner-bid';
    item.innerHTML = `
      <div class="bid-correlative">#${state.totalPujas}</div>
      <div class="bid-details">
        <div class="bid-row-top">
          <span class="bid-amount">S/. ${formatearNumero(monto)}</span>
          <span class="bid-time">Ahora</span>
        </div>
        <div class="bid-row-bottom">
          <span>${escaparHTML(etiqueta)}</span>
          <span class="audit-tag">✓ Registrada</span>
        </div>
      </div>
    `;
    timeline.querySelector('.winner-bid')?.classList.remove('winner-bid');
    timeline.insertBefore(item, timeline.firstChild);
  }

  document.getElementById('btnActivarSubasta')?.addEventListener('click', async () => {
    await fetch(`/api/v1/subastas/${state.subastaId}/activar`, {
      method: 'POST',
      headers: { Authorization: `Bearer ${state.authToken}` }
    });
    await cargarCatalogoSubastas();
    await sincronizarEstadoInicial();
  });

  document.getElementById('btnMartilloConteo')?.addEventListener('click', async () => {
    await fetch(`/api/v1/subastas/${state.subastaId}/conteo`, {
      method: 'POST',
      headers: { Authorization: `Bearer ${state.authToken}` }
    });
    if (state.tiempoSegundos > 15) state.tiempoSegundos = 15;
    await cargarCatalogoSubastas();
    await sincronizarEstadoInicial();
  });

  document.getElementById('btnAdjudicar')?.addEventListener('click', async () => {
    await fetch(`/api/v1/subastas/${state.subastaId}/cerrar`, {
      method: 'POST',
      headers: { Authorization: `Bearer ${state.authToken}` }
    });
    finalizarSubastaPorTiempo();
    await cargarCatalogoSubastas();
    await sincronizarEstadoInicial();
  });

  // ---------------------------------------------------------------------------
  // 8. SOCIAL FEED EN POSTGRESQL (/api/v1/social/posts)
  // ---------------------------------------------------------------------------
  async function cargarSocialFeed() {
    try {
      const res = await fetch('/api/v1/social/posts');
      if (!res.ok) return;
      const posts = await res.json();
      if (Array.isArray(posts)) {
        state.socialPosts = posts;
        renderizarSocialFeed();
      }
    } catch (_) {}
  }

  function renderizarSocialFeed() {
    const list = document.getElementById('socialFeedList');
    if (!list) return;
    list.innerHTML = '';

    if (state.socialPosts.length === 0) {
      const empty = document.createElement('div');
      empty.className = 'card';
      empty.textContent = 'Sé el primero en compartir una publicación con la comunidad.';
      list.appendChild(empty);
      return;
    }

    state.socialPosts.forEach((p) => {
      const card = document.createElement('article');
      card.className = 'social-post-card';

      const header = document.createElement('div');
      header.className = 'social-post-header';

      const authorWrap = document.createElement('div');
      authorWrap.className = 'social-author-wrap';
      const avatar = document.createElement('div');
      avatar.className = 'user-avatar';
      avatar.textContent = inicialesDeNombre(p.autor_nombre || 'AU');
      const authorMeta = document.createElement('div');
      const nameEl = document.createElement('strong');
      nameEl.textContent = p.autor_nombre || 'Participante';
      const subEl = document.createElement('div');
      subEl.style.fontSize = '0.72rem';
      subEl.style.color = 'var(--text-muted)';
      subEl.textContent = `${p.autor_rol || 'COMUNIDAD'} • ${formatearFechaCorta(p.created_at)}`;
      authorMeta.append(nameEl, subEl);
      authorWrap.append(avatar, authorMeta);

      const tagPill = document.createElement('span');
      tagPill.className = 'tier-pill';
      tagPill.textContent = p.etiqueta || 'COMUNIDAD';
      header.append(authorWrap, tagPill);

      const content = document.createElement('p');
      content.className = 'social-post-content';
      content.textContent = p.contenido;

      card.append(header, content);

      if (p.imagen_url) {
        const img = document.createElement('img');
        img.className = 'social-post-img';
        img.src = p.imagen_url;
        img.alt = 'Imagen compartida en la comunidad';
        img.loading = 'lazy';
        card.appendChild(img);
      }

      if (p.subasta_id) {
        const lotBtn = document.createElement('button');
        lotBtn.type = 'button';
        lotBtn.className = 'social-linked-lot';
        lotBtn.textContent = `🔨 Ver bien vinculado en Sala (${p.subasta_id})`;
        lotBtn.addEventListener('click', () => irASalaDeBien(p.subasta_id));
        card.appendChild(lotBtn);
      }

      const actions = document.createElement('div');
      actions.className = 'social-post-actions';
      const likeBtn = document.createElement('button');
      likeBtn.type = 'button';
      likeBtn.className = 'btn-social-like';
      likeBtn.textContent = `❤️ Me interesa (${p.likes_count || 0})`;
      likeBtn.addEventListener('click', async () => {
        const r = await fetch(`/api/v1/social/posts/${encodeURIComponent(p.id)}/like`, { method: 'POST' });
        if (r.ok) {
          const d = await r.json();
          p.likes_count = d.likes_count;
          likeBtn.textContent = `❤️ Me interesa (${p.likes_count})`;
        }
      });
      actions.appendChild(likeBtn);
      card.appendChild(actions);

      const commentsBox = document.createElement('div');
      commentsBox.className = 'social-comments-box';
      const comments = Array.isArray(p.comentarios) ? p.comentarios : [];
      comments.forEach((c) => {
        const cItem = document.createElement('div');
        cItem.className = 'social-comment-item';
        const cAuthor = document.createElement('strong');
        cAuthor.textContent = `${c.autor_nombre}: `;
        cItem.append(cAuthor, document.createTextNode(c.comentario));
        commentsBox.appendChild(cItem);
      });

      const cForm = document.createElement('form');
      cForm.className = 'social-comment-form';
      const cInput = document.createElement('input');
      cInput.type = 'text';
      cInput.placeholder = 'Escribe un comentario...';
      cInput.maxLength = 400;
      cInput.required = true;
      const cBtn = document.createElement('button');
      cBtn.type = 'submit';
      cBtn.className = 'btn-pill-quick';
      cBtn.textContent = 'Comentar';
      cForm.append(cInput, cBtn);

      cForm.addEventListener('submit', async (ev) => {
        ev.preventDefault();
        if (!state.authToken) {
          cambiarTabAuth('login', false);
          authDialog.showModal();
          return;
        }
        const txt = cInput.value.trim();
        if (!txt) return;
        const r = await fetch(`/api/v1/social/posts/${encodeURIComponent(p.id)}/comentarios`, {
          method: 'POST',
          headers: {
            'Content-Type': 'application/json',
            Authorization: `Bearer ${state.authToken}`
          },
          body: JSON.stringify({ comentario: txt })
        });
        if (r.ok) {
          cInput.value = '';
          await cargarSocialFeed();
        }
      });

      commentsBox.appendChild(cForm);
      card.appendChild(commentsBox);
      list.appendChild(card);
    });
  }

  document.getElementById('socialPostForm')?.addEventListener('submit', async (e) => {
    e.preventDefault();
    if (!state.authToken) {
      mostrarToast('Inicia sesión para publicar en el Social Feed.', 'error');
      cambiarTabAuth('login', false);
      authDialog.showModal();
      return;
    }

    const input = document.getElementById('socialPostInput');
    const tag = document.getElementById('socialPostTag')?.value || 'COMUNIDAD';
    const subastaId = document.getElementById('socialPostAuctionSelect')?.value || '';
    const imgUrl = document.getElementById('socialPostImageUrl')?.value.trim() || '';

    const res = await fetch('/api/v1/social/posts', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${state.authToken}`
      },
      body: JSON.stringify({
        contenido: input.value.trim(),
        etiqueta: tag,
        subasta_id: subastaId,
        imagen_url: imgUrl
      })
    });

    if (res.ok) {
      e.target.reset();
      await cargarSocialFeed();
      mostrarToast('✨ Tu publicación ya está visible en el Social Feed.', 'success');
    } else {
      mostrarToast('No se pudo publicar en este momento.', 'error');
    }
  });

  // ---------------------------------------------------------------------------
  // 9. SECCIÓN MI PERFIL Y AJUSTES (PATCH /api/v1/auth/me)
  // ---------------------------------------------------------------------------
  async function renderizarVistaPerfil() {
    const autenticado = Boolean(state.authToken && state.usuarioActual);
    if (autenticado) {
      try {
        const r = await fetch('/api/v1/auth/me', {
          headers: { Authorization: `Bearer ${state.authToken}` }
        });
        if (r.ok) {
          const fresh = await r.json();
          state.usuarioActual = fresh;
          localStorage.setItem('livebid_user', JSON.stringify(fresh));
          actualizarCabeceraUsuario();
        }
      } catch (_) {}
    }

    const u = state.usuarioActual;
    const bigInitials = document.getElementById('profileBigInitials');
    const heading = document.getElementById('profileHeading');
    const rolePill = document.getElementById('profileRolePill');
    const emailTxt = document.getElementById('profileEmailText');
    const bioPreview = document.getElementById('profileBioPreview');
    const authActionBtn = document.getElementById('btnProfileAuthAction');

    if (autenticado && u) {
      bigInitials.textContent = inicialesDeNombre(u.name);
      heading.textContent = u.name;
      rolePill.textContent = u.role || 'POSTOR';
      emailTxt.textContent = `${u.email} • Estado: ${u.status || 'ACTIVO'}`;
      bioPreview.textContent = u.bio || 'Completa tu biografía e intereses de colección en el formulario.';
      if (authActionBtn) authActionBtn.textContent = 'Cambiar de Cuenta';

      document.getElementById('profName').value = u.name || '';
      document.getElementById('profPhone').value = u.telefono || '';
      document.getElementById('profDoc').value = u.documento_identidad || '';
      document.getElementById('profWallet').value = u.wallet_address || '';
      document.getElementById('profBio').value = u.bio || '';
    } else {
      bigInitials.textContent = 'VS';
      heading.textContent = 'Visitante';
      rolePill.textContent = 'EXPLORANDO';
      emailTxt.textContent = 'Sin sesión iniciada';
      bioPreview.textContent = 'Inicia sesión o crea tu cuenta para editar tu perfil y sincronizar tu actividad.';
      if (authActionBtn) authActionBtn.textContent = 'Iniciar Sesión / Crear Cuenta';
    }

    const watchGrid = document.getElementById('profileWatchlistGrid');
    if (watchGrid) {
      watchGrid.innerHTML = '';
      const guardadas = state.subastas.filter((s) => state.watchlist.has(s.id));
      if (guardadas.length === 0) {
        const p = document.createElement('p');
        p.style.fontSize = '0.84rem';
        p.style.color = 'var(--text-secondary)';
        p.textContent = 'Aún no has marcado subastas con ⭐ en el Cronograma.';
        watchGrid.appendChild(p);
      } else {
        guardadas.forEach((s) => {
          const item = document.createElement('div');
          item.className = 'spec-box';
          const title = document.createElement('strong');
          title.textContent = s.titulo;
          const btn = document.createElement('button');
          btn.type = 'button';
          btn.className = 'btn-pill-quick';
          btn.style.marginTop = '8px';
          btn.textContent = 'Ir a la Subasta';
          btn.addEventListener('click', () => irASalaDeBien(s.id));
          item.append(title, btn);
          watchGrid.appendChild(item);
        });
      }
    }
  }

  document.getElementById('btnProfileAuthAction')?.addEventListener('click', () => {
    cambiarTabAuth('login', false);
    authDialog.showModal();
  });

  document.getElementById('btnProfilePublishAction')?.addEventListener('click', solicitarPublicarBien);

  document.getElementById('profileEditForm')?.addEventListener('submit', async (e) => {
    e.preventDefault();
    if (!state.authToken) {
      mostrarFeedbackDialog('profileSaveFeedback', 'Primero inicia sesión para guardar los datos de tu perfil.', 'error');
      return;
    }

    const res = await fetch('/api/v1/auth/me', {
      method: 'PATCH',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${state.authToken}`
      },
      body: JSON.stringify({
        name: document.getElementById('profName').value.trim(),
        telefono: document.getElementById('profPhone').value.trim(),
        documento_identidad: document.getElementById('profDoc').value.trim(),
        wallet_address: document.getElementById('profWallet').value.trim(),
        bio: document.getElementById('profBio').value.trim()
      })
    });

    if (res.ok) {
      const updated = await res.json();
      state.usuarioActual = updated;
      localStorage.setItem('livebid_user', JSON.stringify(updated));
      actualizarCabeceraUsuario();
      await renderizarVistaPerfil();
      mostrarFeedbackDialog('profileSaveFeedback', '✓ Tus datos y biografía se actualizaron correctamente.', 'success');
    } else {
      mostrarFeedbackDialog('profileSaveFeedback', 'No se pudo actualizar el perfil.', 'error');
    }
  });

  // ---------------------------------------------------------------------------
  // 10. MODALES DE AUTENTICACIÓN, LOTES, ADMIN Y GARANTÍA REEMBOLSABLE
  // ---------------------------------------------------------------------------
  function cambiarTabAuth(tab, modoVendedor = false) {
    document.querySelectorAll('[data-auth-tab]').forEach((b) => {
      b.classList.toggle('active', b.getAttribute('data-auth-tab') === tab);
    });
    document.getElementById('loginForm').classList.toggle('hidden', tab !== 'login');
    document.getElementById('registerForm').classList.toggle('hidden', tab !== 'register');
    document.getElementById('sellerOnboardingBanner')?.classList.toggle('hidden', !modoVendedor);
    if (modoVendedor) {
      document.getElementById('regRole').value = 'VENDEDOR';
    }
  }

  function solicitarPublicarBien() {
    const rol = obtenerRolUsuario();
    if (state.authToken && ['VENDEDOR', 'MARTILLERO', 'ADMIN'].includes(rol)) {
      abrirModalLote('nueva');
      return;
    }
    cambiarTabAuth('register', true);
    authDialog.showModal();
  }

  document.getElementById('btnPublishItemNav')?.addEventListener('click', solicitarPublicarBien);
  document.getElementById('btnPublishItemCatalog')?.addEventListener('click', solicitarPublicarBien);
  document.getElementById('btnOpenAuthModal')?.addEventListener('click', () => {
    cambiarTabAuth('login', false);
    authDialog.showModal();
  });

  document.querySelectorAll('[data-auth-tab]').forEach((btn) => {
    btn.addEventListener('click', () => cambiarTabAuth(btn.getAttribute('data-auth-tab'), false));
  });

  document.getElementById('btnFillAdminCreds')?.addEventListener('click', () => {
    document.getElementById('loginEmail').value = 'admin@livebid.pe';
    document.getElementById('loginPassword').value = 'AdminLiveBid2026!';
  });

  function mostrarFeedbackDialog(elId, msg, tipo = 'success') {
    const el = document.getElementById(elId);
    if (!el) return;
    el.textContent = msg;
    el.className = `dialog-feedback ${tipo}`;
    el.classList.remove('hidden');
  }

  document.getElementById('loginForm').addEventListener('submit', async (e) => {
    e.preventDefault();
    const res = await fetch('/api/v1/auth/login', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        email: document.getElementById('loginEmail').value.trim(),
        password: document.getElementById('loginPassword').value
      })
    });
    const data = await res.json().catch(() => ({}));
    if (!res.ok) {
      mostrarFeedbackDialog('authFeedbackMsg', 'Correo o contraseña incorrectos, o cuenta pendiente de verificación.', 'error');
      return;
    }
    guardarSesion(data.access_token, data.user);
    mostrarFeedbackDialog('authFeedbackMsg', `¡Bienvenido, ${data.user.name}!`, 'success');
    mostrarToast(`Sesión iniciada como ${data.user.name}.`, 'success');
    setTimeout(() => authDialog.close(), 400);
  });

  document.getElementById('registerForm').addEventListener('submit', async (e) => {
    e.preventDefault();
    const payload = {
      name: document.getElementById('regName').value.trim(),
      email: document.getElementById('regEmail').value.trim(),
      password: document.getElementById('regPassword').value,
      rol_solicitado: document.getElementById('regRole').value,
      documento_identidad: document.getElementById('regDoc').value.trim(),
      telefono: document.getElementById('regPhone').value.trim(),
      wallet_address: document.getElementById('regWallet').value.trim()
    };

    const res = await fetch('/api/v1/auth/register', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(payload)
    });
    const data = await res.json().catch(() => ({}));
    if (!res.ok) {
      mostrarFeedbackDialog('authFeedbackMsg', `No se pudo crear la cuenta: ${data.error || 'verifica tus datos'}`, 'error');
      return;
    }

    mostrarToast(
      `✅ ¡Cuenta creada! El correo ${data.email} quedó registrado con solicitud de rol ${data.requested_role || payload.rol_solicitado}. Pendiente de verificación.`,
      'success'
    );
    authDialog.close();
    e.target.reset();
  });

  function abrirModalLote(tab = 'detalles') {
    document.querySelectorAll('[data-lot-tab]').forEach((b) => {
      b.classList.toggle('active', b.getAttribute('data-lot-tab') === tab);
    });
    document.getElementById('lotDetailsForm').classList.toggle('hidden', tab !== 'detalles');
    document.getElementById('encargadoForm').classList.toggle('hidden', tab !== 'encargado');
    document.getElementById('newAuctionForm').classList.toggle('hidden', tab !== 'nueva');
    lotManagerDialog.showModal();
  }

  document.getElementById('btnOpenLotModal')?.addEventListener('click', () => abrirModalLote('detalles'));
  document.getElementById('btnQuickUploadImage')?.addEventListener('click', () => abrirModalLote('detalles'));
  document.getElementById('btnEditEncargadoQuick')?.addEventListener('click', () => abrirModalLote('encargado'));
  document.querySelectorAll('[data-lot-tab]').forEach((btn) => {
    btn.addEventListener('click', () => abrirModalLote(btn.getAttribute('data-lot-tab')));
  });

  function comprimirImagenADataURL(file) {
    return new Promise((resolve, reject) => {
      const reader = new FileReader();
      reader.onload = (ev) => {
        const img = new Image();
        img.onload = () => {
          const maxW = 900;
          const scale = img.width > maxW ? maxW / img.width : 1;
          const canvas = document.createElement('canvas');
          canvas.width = Math.round(img.width * scale);
          canvas.height = Math.round(img.height * scale);
          canvas.getContext('2d').drawImage(img, 0, 0, canvas.width, canvas.height);
          resolve(canvas.toDataURL('image/jpeg', 0.82));
        };
        img.onerror = reject;
        img.src = ev.target.result;
      };
      reader.onerror = reject;
      reader.readAsDataURL(file);
    });
  }

  document.getElementById('lotImageFileInput')?.addEventListener('change', async (e) => {
    const file = e.target.files?.[0];
    if (!file) return;
    try {
      state.imagenReferencialDataUrl = await comprimirImagenADataURL(file);
      const prev = document.getElementById('lotUploadPreviewImg');
      prev.src = state.imagenReferencialDataUrl;
      prev.classList.remove('hidden');
      document.getElementById('lotUploadPlaceholder').classList.add('hidden');
    } catch (_) {}
  });

  document.getElementById('newSubImageFile')?.addEventListener('change', async (e) => {
    const file = e.target.files?.[0];
    if (!file) return;
    try {
      state.nuevaSubastaImagenDataUrl = await comprimirImagenADataURL(file);
      const prev = document.getElementById('newSubPreviewImg');
      prev.src = state.nuevaSubastaImagenDataUrl;
      prev.classList.remove('hidden');
      document.getElementById('newSubPreviewPlaceholder').classList.add('hidden');
    } catch (_) {}
  });

  document.getElementById('lotDetailsForm')?.addEventListener('submit', async (e) => {
    e.preventDefault();
    const imagenes = [];
    if (state.imagenReferencialDataUrl) imagenes.push(state.imagenReferencialDataUrl);
    const urlManual = document.getElementById('lotImageUrlInput').value.trim();
    if (urlManual) imagenes.push(urlManual);
    if (imagenes.length === 0 && state.subastaActual?.detalles_lote?.imagenes_urls) {
      imagenes.push(...state.subastaActual.detalles_lote.imagenes_urls);
    }

    const res = await fetch(`/api/v1/subastas/${state.subastaId}/detalles-lote`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${state.authToken}`
      },
      body: JSON.stringify({
        categoria: document.getElementById('detCategoria').value.trim(),
        ubicacion: document.getElementById('detUbicacion').value.trim(),
        precio_reserva: Number(document.getElementById('detReserva').value) || state.precioBase,
        inspeccionado_por: document.getElementById('detInspector').value.trim(),
        certificado_peritaje_url: document.getElementById('detCertUrl').value.trim(),
        imagenes_urls: imagenes
      })
    });
    if (res.ok) {
      await cargarCatalogoSubastas();
      await sincronizarEstadoInicial();
      lotManagerDialog.close();
      mostrarToast('Ficha y fotografía del bien actualizadas.', 'success');
    }
  });

  document.getElementById('encargadoForm')?.addEventListener('submit', async (e) => {
    e.preventDefault();
    const res = await fetch(`/api/v1/subastas/${state.subastaId}/encargado`, {
      method: 'PUT',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${state.authToken}`
      },
      body: JSON.stringify({
        martillero_id: document.getElementById('encMartilleroId').value.trim(),
        nombre: document.getElementById('encNombre').value.trim(),
        cargo: document.getElementById('encCargo').value.trim(),
        licencia: document.getElementById('encLicencia').value.trim(),
        contacto: document.getElementById('encContacto').value.trim(),
        experiencia: document.getElementById('encExperiencia').value.trim()
      })
    });
    if (res.ok) {
      await sincronizarEstadoInicial();
      lotManagerDialog.close();
      mostrarToast('Martillero responsable actualizado.', 'success');
    }
  });

  document.getElementById('newAuctionForm')?.addEventListener('submit', async (e) => {
    e.preventDefault();
    const durMin = Number(document.getElementById('newSubDuracionMin').value) || 60;
    const inicio = new Date();
    const cierre = new Date(inicio.getTime() + durMin * 60 * 1000);
    const precioBase = Number(document.getElementById('newSubPrecioBase').value) || 35000;
    const reserva = Number(document.getElementById('newSubReserva').value) || precioBase;
    const urlImg = document.getElementById('newSubImageUrl').value.trim();
    const imgs = [];
    if (state.nuevaSubastaImagenDataUrl) imgs.push(state.nuevaSubastaImagenDataUrl);
    if (urlImg) imgs.push(urlImg);

    const res = await fetch('/api/v1/subastas', {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${state.authToken}`
      },
      body: JSON.stringify({
        titulo: document.getElementById('newSubTitulo').value.trim(),
        descripcion: document.getElementById('newSubDesc').value.trim(),
        precio_base: precioBase,
        inicio: inicio.toISOString(),
        cierre: cierre.toISOString(),
        encargado: {
          martillero_id: 'MART-01',
          nombre: document.getElementById('encNombre').value.trim() || 'Dr. Carlos Mendoza Ríos',
          cargo: 'Martillero Público Oficial',
          licencia: 'REG-SUNARP-2026-0412',
          contacto: 'cmendoza@livebid.pe'
        },
        detalles_lote: {
          categoria: document.getElementById('newSubCategoria').value,
          ubicacion: document.getElementById('newSubUbicacion').value.trim() || 'San Isidro, Lima',
          precio_reserva: reserva >= precioBase ? reserva : precioBase,
          inspeccionado_por: `Consignado por ${state.usuarioActual?.name || 'Vendedor Verificado'}`,
          imagenes_urls: imgs
        }
      })
    });

    const created = await res.json().catch(() => ({}));
    if (!res.ok) {
      mostrarToast(`No se pudo publicar el bien: ${created.error || res.statusText}`, 'error');
      return;
    }

    await fetch(`/api/v1/subastas/${created.id}/activar`, {
      method: 'POST',
      headers: { Authorization: `Bearer ${state.authToken}` }
    });

    state.nuevaSubastaImagenDataUrl = '';
    e.target.reset();
    lotManagerDialog.close();
    await cargarCatalogoSubastas();
    await irASalaDeBien(created.id);
    mostrarToast(`✨ "${created.titulo}" publicado en el Cronograma, Catálogo y Sala en Vivo.`, 'success');
  });

  // Panel Admin
  async function abrirPanelAdmin() {
    adminDialog.showModal();
    await cargarUsuariosAdmin();
  }

  document.getElementById('btnOpenAdminModal')?.addEventListener('click', abrirPanelAdmin);
  document.getElementById('btnRefreshUsers')?.addEventListener('click', cargarUsuariosAdmin);
  document.getElementById('adminFilterRole')?.addEventListener('change', cargarUsuariosAdmin);
  document.getElementById('adminFilterStatus')?.addEventListener('change', cargarUsuariosAdmin);

  async function cargarUsuariosAdmin() {
    const tbody = document.getElementById('adminUsersTableBody');
    const params = new URLSearchParams();
    const rf = document.getElementById('adminFilterRole').value;
    const sf = document.getElementById('adminFilterStatus').value;
    if (rf) params.set('rol', rf);
    if (sf) params.set('estado', sf);

    const res = await fetch(`/api/v1/admin/users?${params.toString()}`, {
      headers: { Authorization: `Bearer ${state.authToken}` }
    });
    if (!res.ok) return;
    const users = await res.json();
    tbody.innerHTML = '';

    users.forEach((u) => {
      const reqRole = u.requested_role || u.role;
      const tr = document.createElement('tr');
      tr.innerHTML = `
        <th scope="row">
          <div>${escaparHTML(u.name)}</div>
          <div style="font-size:0.72rem; color:var(--text-muted); font-weight:400;">${escaparHTML(u.email)}</div>
        </th>
        <td>${escaparHTML(u.documento_identidad || '-')} • ${escaparHTML(u.telefono || '-')}</td>
        <td><span class="tier-pill">${escaparHTML(reqRole)}</span></td>
        <td>
          <select class="admin-role-select" data-user-id="${escaparHTML(u.id)}">
            <option value="POSTOR" ${u.role === 'POSTOR' ? 'selected' : ''}>POSTOR</option>
            <option value="VENDEDOR" ${u.role === 'VENDEDOR' ? 'selected' : ''}>VENDEDOR</option>
            <option value="MARTILLERO" ${u.role === 'MARTILLERO' ? 'selected' : ''}>MARTILLERO</option>
            <option value="ADMIN" ${u.role === 'ADMIN' ? 'selected' : ''}>ADMIN</option>
          </select>
        </td>
        <td><strong>${escaparHTML(u.status)}</strong></td>
        <td>
          <div class="admin-action-group">
            <button type="button" class="btn-table-act approve" data-action="approve" data-id="${escaparHTML(u.id)}" data-req-role="${escaparHTML(reqRole)}">✅ Verificar y Activar</button>
            <button type="button" class="btn-table-act suspend" data-action="suspend" data-id="${escaparHTML(u.id)}">⏸️ Suspender</button>
            <button type="button" class="btn-table-act block" data-action="block" data-id="${escaparHTML(u.id)}">🚫 Bloquear</button>
          </div>
        </td>
      `;
      tbody.appendChild(tr);
    });

    tbody.querySelectorAll('.admin-role-select').forEach((sel) => {
      sel.addEventListener('change', async (ev) => {
        const uid = ev.target.getAttribute('data-user-id');
        await fetch(`/api/v1/admin/users/${uid}/role`, {
          method: 'PATCH',
          headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${state.authToken}` },
          body: JSON.stringify({ rol: ev.target.value })
        });
        await cargarUsuariosAdmin();
      });
    });

    tbody.querySelectorAll('[data-action]').forEach((btn) => {
      btn.addEventListener('click', async () => {
        const action = btn.getAttribute('data-action');
        const uid = btn.getAttribute('data-id');
        const reqRole = btn.getAttribute('data-req-role');

        if (action === 'approve') {
          if (reqRole && reqRole !== 'ADMIN') {
            await fetch(`/api/v1/admin/users/${uid}/role`, {
              method: 'PATCH',
              headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${state.authToken}` },
              body: JSON.stringify({ rol: reqRole })
            });
          }
          await fetch(`/api/v1/admin/users/${uid}/status`, {
            method: 'PATCH',
            headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${state.authToken}` },
            body: JSON.stringify({ estado: 'ACTIVO', kyc_aprobado: true })
          });
          mostrarFeedbackDialog('adminFeedbackMsg', `✓ Usuario ${uid} activado con KYC aprobado (${reqRole}).`, 'success');
        } else if (action === 'suspend') {
          await fetch(`/api/v1/admin/users/${uid}/status`, {
            method: 'PATCH',
            headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${state.authToken}` },
            body: JSON.stringify({ estado: 'SUSPENDIDO', kyc_aprobado: false })
          });
        } else if (action === 'block') {
          await fetch(`/api/v1/admin/users/${uid}/status`, {
            method: 'PATCH',
            headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${state.authToken}` },
            body: JSON.stringify({ estado: 'BLOQUEADO', kyc_aprobado: false })
          });
        }
        await cargarUsuariosAdmin();
      });
    });
  }

  // Garantía Reembolsable
  async function sincronizarGarantiaActivaBackend() {
    if (!state.authToken) {
      state.garantiaActiva = null;
      actualizarGuiaPersonalEnSala();
      return;
    }
    try {
      const res = await fetch(`/api/v1/subastas/${state.subastaId}/garantias/me`, {
        headers: { Authorization: `Bearer ${state.authToken}` }
      });
      if (res.ok) {
        const g = await res.json();
        state.garantiaActiva = g;
        elEscrowBadge.textContent = `💳 Garantía: S/. ${formatearNumero(g.monto)} (${g.estado})`;
        document.getElementById('guaranteeStatusNote').innerHTML =
          `✓ <strong>Garantía Activa:</strong> Tienes S/. ${formatearNumero(g.monto)} reservados de forma reembolsable para ofertar en este bien.`;
      } else {
        state.garantiaActiva = null;
        actualizarCabeceraUsuario();
      }
    } catch (_) {}
    actualizarGuiaPersonalEnSala();
  }

  async function prepararStripePaymentIntent() {
    const reqDefault = Math.round((state.precioBase || 45000) * 0.1);
    document.getElementById('stripeSummaryAuction').textContent = state.subastaActual?.titulo || state.subastaId;
    document.getElementById('stripeSummaryAmount').textContent = `S/. ${formatearNumero(reqDefault)}`;
    document.getElementById('stripeHoldAmountInput').value = reqDefault;
    document.getElementById('stripeCardHolder').value = state.usuarioActual?.name || '';

    if (!state.authToken) {
      document.getElementById('stripeSummaryIntentId').textContent = 'Inicia sesión para continuar';
      return;
    }

    const res = await fetch(`/api/v1/subastas/${state.subastaId}/garantias/stripe-intent`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${state.authToken}` },
      body: JSON.stringify({ monto: reqDefault })
    });
    const data = await res.json().catch(() => ({}));
    if (res.ok) {
      state.stripeIntentActual = data;
      document.getElementById('stripeSummaryAmount').textContent = `S/. ${formatearNumero(data.monto_requerido)}`;
      document.getElementById('stripeSummaryIntentId').textContent = data.payment_intent_id;
      document.getElementById('stripeHoldAmountInput').value = data.monto_retencion || data.monto_requerido;
    }
  }

  async function abrirModalStripe() {
    stripeDialog.showModal();
    await prepararStripePaymentIntent();
  }

  document.getElementById('btnOpenStripeModal')?.addEventListener('click', abrirModalStripe);
  document.getElementById('btnOpenStripeNav')?.addEventListener('click', abrirModalStripe);
  document.getElementById('btnRecalcStripeIntent')?.addEventListener('click', prepararStripePaymentIntent);

  document.getElementById('stripeGuaranteeForm')?.addEventListener('submit', async (e) => {
    e.preventDefault();
    const monto = Number(document.getElementById('stripeHoldAmountInput').value) || Math.round(state.precioBase * 0.1);
    const res = await fetch(`/api/v1/subastas/${state.subastaId}/garantias`, {
      method: 'POST',
      headers: { 'Content-Type': 'application/json', Authorization: `Bearer ${state.authToken}` },
      body: JSON.stringify({
        monto,
        payment_intent_id: state.stripeIntentActual?.payment_intent_id || ''
      })
    });
    const data = await res.json().catch(() => ({}));
    if (res.ok) {
      state.garantiaActiva = data;
      actualizarCabeceraUsuario();
      await sincronizarGarantiaActivaBackend();
      stripeDialog.close();
      mostrarToast(`💳 Garantía reembolsable RETENIDA (S/. ${formatearNumero(data.monto)}).`, 'success');
    } else {
      mostrarFeedbackDialog('stripeFeedbackMsg', `No se pudo confirmar: ${data.error || res.statusText}`, 'error');
    }
  });

  // ---------------------------------------------------------------------------
  // 11. ASISTENTE GEMINI FLOTANTE DINÁMICO Y DRAWER LATERAL
  // ---------------------------------------------------------------------------
  document.getElementById('btnDismissAiBubble')?.addEventListener('click', () => {
    document.getElementById('aiGreetingBubble')?.classList.add('hidden');
  });

  document.querySelectorAll('[data-ai-quick]').forEach((btn) => {
    btn.addEventListener('click', async () => {
      document.getElementById('aiGreetingBubble')?.classList.add('hidden');
      await assistantDrawerApi.open();
      enviarConsultaAsistente(btn.getAttribute('data-ai-quick'));
    });
  });

  document.querySelectorAll('[data-assistant-goto]').forEach((btn) => {
    btn.addEventListener('click', () => {
      const target = btn.getAttribute('data-assistant-goto');
      assistantDrawerApi.close();
      cambiarVistaActiva(target);
    });
  });

  function inicializarDrawerAsistente() {
    const drawer = document.getElementById('aiAssistantDrawer');
    const scroller = drawer?.querySelector('.Drawer-scroller');
    const sheet = drawer?.querySelector('.Drawer-sheet');
    const closeBtn = document.getElementById('btnCloseAiDrawer');
    const openTriggers = [
      document.getElementById('btnOpenAiDrawer'),
      document.getElementById('btnFloatingAiAssistant'),
      document.getElementById('btnOpenAiFromCard'),
      document.getElementById('btnOpenAiFromSocial')
    ].filter(Boolean);

    if (!drawer || !scroller || !sheet) {
      return { open: async () => {}, close: () => {} };
    }

    const setExpanded = (val) => {
      openTriggers.forEach((b) => {
        if (b.hasAttribute('aria-expanded')) b.setAttribute('aria-expanded', String(val));
      });
    };

    async function openDrawer() {
      document.getElementById('aiGreetingBubble')?.classList.add('hidden');
      if (!drawer.open) drawer.showModal();
      actualizarContextoAsistente();

      if (!CSS.supports('scroll-initial-target', 'nearest')) {
        scroller.scrollTo({ left: 0, behavior: 'instant' });
        await new Promise((r) => requestAnimationFrame(() => requestAnimationFrame(r)));
      }
      scroller.scrollTo({ left: scroller.scrollWidth, behavior: 'auto' });
    }

    function closeDrawer() {
      if (!drawer.open) return;
      scroller.scrollTo({ left: 0, behavior: 'auto' });
    }

    if (!CSS.supports('animation-timeline: scroll()')) {
      scroller.addEventListener('scroll', () => {
        const w = sheet.offsetWidth || 1;
        drawer.style.setProperty('--drawer-backdrop', String(Math.min(1, Math.max(0, scroller.scrollLeft / w))));
      });
    }

    const visibleThreshold = 1 / Math.max(window.innerWidth, 320);
    const observer = new IntersectionObserver(
      (entries) => {
        const entry = entries.at(-1);
        if (!entry) return;
        if (entry.intersectionRatio < visibleThreshold && drawer.open) {
          drawer.close();
          setExpanded(false);
        } else if (entry.intersectionRatio >= 0.98 && drawer.open) {
          setExpanded(true);
        }
      },
      { root: drawer, threshold: [visibleThreshold, 0.98, 1] }
    );
    observer.observe(sheet);

    openTriggers.forEach((b) => b.addEventListener('click', openDrawer));
    closeBtn?.addEventListener('click', closeDrawer);
    drawer.addEventListener('click', (ev) => {
      if (!sheet.contains(ev.target)) closeDrawer();
    });
    drawer.addEventListener('cancel', (ev) => {
      if (ev.cancelable) {
        ev.preventDefault();
        closeDrawer();
      }
    });

    return { open: openDrawer, close: closeDrawer };
  }

  function actualizarContextoAsistente() {
    const elCtx = document.getElementById('aiLiveContextText');
    if (!elCtx) return;
    const titulo = state.subastaActual?.titulo || 'Cronograma general';
    elCtx.textContent = `Bien activo: ${titulo} • S/. ${formatearNumero(state.precioActual)}`;
  }

  function agregarMensajeChat(rol, texto, etiqueta = '') {
    const container = document.getElementById('aiChatMessages');
    if (!container) return;

    const bubble = document.createElement('div');
    bubble.className = `ai-msg ${rol === 'user' ? 'user' : 'assistant'}`;

    const author = document.createElement('div');
    author.className = 'ai-msg-author';
    author.textContent = rol === 'user' ? 'Tú' : (etiqueta || '✨ Asistente Aureum');

    const body = document.createElement('div');
    body.className = 'ai-msg-text';
    String(texto).split(/(\*\*[^*]+\*\*)/g).forEach((part) => {
      if (part.startsWith('**') && part.endsWith('**') && part.length > 4) {
        const strong = document.createElement('strong');
        strong.textContent = part.slice(2, -2);
        body.appendChild(strong);
      } else {
        body.appendChild(document.createTextNode(part));
      }
    });

    bubble.append(author, body);
    container.appendChild(bubble);
    container.scrollTop = container.scrollHeight;
  }

  async function enviarConsultaAsistente(mensajeRaw) {
    const mensaje = String(mensajeRaw || '').trim();
    if (!mensaje) return;

    const input = document.getElementById('aiChatInput');
    const sendBtn = document.getElementById('btnSendAiChat');
    const badge = document.getElementById('aiProviderBadge');

    if (input) input.value = '';
    agregarMensajeChat('user', mensaje);
    if (sendBtn) sendBtn.disabled = true;
    if (badge) badge.textContent = 'Escribiendo...';

    try {
      const res = await fetch('/api/v1/ia/asistente', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          mensaje,
          subasta_id: state.subastaId,
          rol_usuario: obtenerRolUsuario(),
          nombre_usuario: state.usuarioActual?.name || 'Visitante',
          tiene_garantia: Boolean(state.garantiaActiva && state.garantiaActiva.estado === 'RETENIDA'),
          historial: state.aiHistory.slice(-6)
        })
      });
      const data = await res.json().catch(() => ({}));
      if (res.ok && data.respuesta) {
        agregarMensajeChat('assistant', data.respuesta, data.modo_local ? '✨ Concierge Aureum' : '✨ Google Gemini');
        if (badge) badge.textContent = data.modo_local ? 'Guía Activa' : 'Gemini en Vivo';
        state.aiHistory.push({ rol: 'user', texto: mensaje }, { rol: 'assistant', texto: data.respuesta });
      } else {
        agregarMensajeChat('assistant', 'No pude procesar tu mensaje en este instante, intenta nuevamente.');
        if (badge) badge.textContent = 'En línea';
      }
    } catch (_) {
      agregarMensajeChat('assistant', 'Error temporal de conexión.');
      if (badge) badge.textContent = 'En línea';
    } finally {
      if (sendBtn) sendBtn.disabled = false;
    }
  }

  document.getElementById('aiChatForm')?.addEventListener('submit', (e) => {
    e.preventDefault();
    const input = document.getElementById('aiChatInput');
    if (input?.value.trim()) enviarConsultaAsistente(input.value);
  });

  document.querySelectorAll('#aiSuggestionChips [data-ai-prompt]').forEach((chip) => {
    chip.addEventListener('click', () => enviarConsultaAsistente(chip.getAttribute('data-ai-prompt')));
  });

  // ---------------------------------------------------------------------------
  // 12. VALIDACIÓN ACCESIBLE POS-INTERACCIÓN Y UTILIDADES
  // ---------------------------------------------------------------------------
  function inicializarValidacionAccesible() {
    const linkError = (el, shown) => {
      const errorId = el.dataset?.error;
      if (!errorId) return;
      const ids = (el.getAttribute('aria-describedby') || '').split(/\s+/).filter((id) => id && id !== errorId);
      if (shown) ids.push(errorId);
      if (ids.length) el.setAttribute('aria-describedby', ids.join(' '));
      else el.removeAttribute('aria-describedby');
    };

    if (!CSS.supports('selector(:user-invalid)')) {
      const dirty = new WeakMap();
      const upd = (input) => {
        const ok = input.validity.valid;
        input.classList.toggle('user-invalid-fallback', !ok);
        input.setAttribute('aria-invalid', ok ? 'false' : 'true');
        linkError(input, !ok);
      };
      document.addEventListener('blur', (e) => {
        const st = dirty.get(e.target) || {};
        st.b = true;
        dirty.set(e.target, st);
        if (st.i && e.target.checkValidity) upd(e.target);
      }, true);
      document.addEventListener('input', (e) => {
        const st = dirty.get(e.target) || {};
        st.i = true;
        dirty.set(e.target, st);
        if (st.b && e.target.checkValidity) upd(e.target);
      });
      return;
    }

    const syncAria = (el) => {
      if (!el?.matches?.('input, select, textarea')) return;
      const inv = el.matches(':user-invalid');
      el.setAttribute('aria-invalid', inv ? 'true' : 'false');
      linkError(el, inv);
    };

    document.querySelectorAll('input, select, textarea').forEach(syncAria);
    document.addEventListener('blur', (e) => syncAria(e.target), true);
    document.addEventListener('input', (e) => syncAria(e.target));
    document.addEventListener('invalid', (e) => syncAria(e.target), true);
  }

  function inicialesDeNombre(nombre) {
    return String(nombre || 'US')
      .split(' ')
      .filter(Boolean)
      .map((p) => p[0])
      .join('')
      .slice(0, 2)
      .toUpperCase() || 'US';
  }

  function formatearNumero(num) {
    return Number(num || 0).toLocaleString('es-PE');
  }

  function escaparHTML(str) {
    return String(str ?? '')
      .replace(/&/g, '&amp;')
      .replace(/</g, '&lt;')
      .replace(/>/g, '&gt;')
      .replace(/"/g, '&quot;')
      .replace(/'/g, '&#39;');
  }

  function mostrarToast(mensaje, tipo = 'success') {
    elFeedbackToast.textContent = mensaje;
    elFeedbackToast.className = `feedback-toast ${tipo}`;
    elFeedbackToast.classList.remove('hidden');
    clearTimeout(elFeedbackToast.toastTimer);
    elFeedbackToast.toastTimer = setTimeout(() => {
      elFeedbackToast.classList.add('hidden');
    }, 4500);
  }
});
