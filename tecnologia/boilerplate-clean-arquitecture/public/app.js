/**
 * =============================================================================
 * AUREUM & CO. (LIVEBID) — FRONTEND RESPONSIVO, CATÁLOGO POSTGRESQL Y CONCIERGE GEMINI
 *  - Catálogo dinámico de bienes y objetos de valor respaldado en PostgreSQL
 *  - Flujo guiado para Vendedores (publicación con fotografías) y Postores
 *  - Herramientas de desarrollo visibles únicamente en localhost
 *  - Asistente lateral interactivo con Google Gemini (<dialog> scroll-snap drawer)
 * =============================================================================
 */

document.addEventListener('DOMContentLoaded', async () => {
  // ---------------------------------------------------------------------------
  // 0. DETECCIÓN DE ENTORNO LOCAL Y FALLBACKS DE PLATAFORMA WEB MODERNA
  // ---------------------------------------------------------------------------
  const isLocalhost = ['localhost', '127.0.0.1', '::1'].includes(window.location.hostname);
  document.querySelectorAll('.localhost-dev-only').forEach((el) => {
    el.classList.toggle('hidden', !isLocalhost);
  });

  // Fallback para <dialog closedby="any"> en modales centrados (no aplica al Drawer lateral)
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

  // Validación diferida pos-interacción (:user-invalid + sincronización ARIA)
  inicializarValidacionAccesible();

  // ---------------------------------------------------------------------------
  // 1. ESTADO REACTIVO DE LA APLICACIÓN (SIN USUARIO FICTICIO AL ENTRAR)
  // ---------------------------------------------------------------------------
  const savedUser = (() => {
    try {
      return JSON.parse(localStorage.getItem('livebid_user') || 'null');
    } catch (_) {
      return null;
    }
  })();

  const state = {
    subastaId: 'SUB-2026-001',
    subastas: [],
    filtroCategoria: 'TODOS',
    vitrinaCompacta: false,
    subastaActual: null,
    precioActual: 45000,
    precioBase: 45000,
    incrementoMinimo: 1000,
    postorLiderId: 'NINGUNO',
    postorLiderNombre: 'Esperando primera oferta en sala',
    authToken: localStorage.getItem('livebid_token') || '',
    usuarioActual: savedUser,
    imagenReferencialDataUrl: '',
    nuevaSubastaImagenDataUrl: '',
    stripeIntentActual: null,
    garantiaActiva: null,
    tiempoSegundos: 105,
    activa: true,
    totalPujas: 0,
    geminiOnline: true,
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
  // 2. REFERENCIAS AL DOM
  // ---------------------------------------------------------------------------
  const elAuctionSelector = document.getElementById('auctionSelector');
  const elTopAuctionStateLabel = document.getElementById('topAuctionStateLabel');
  const elNavUserName = document.getElementById('navUserName');
  const elNavUserRoleBadge = document.getElementById('navUserRoleBadge');
  const elUserInitials = document.getElementById('userInitials');
  const elAuthBtnLabel = document.getElementById('authBtnLabel');
  const elEscrowBadge = document.getElementById('escrowBadge');
  const elBtnLogoutUser = document.getElementById('btnLogoutUser');

  // Vitrina del Catálogo PostgreSQL
  const elCatalogGrid = document.getElementById('catalogGrid');
  const elCatalogGridWrap = document.getElementById('catalogGridWrap');
  const elCatalogCountBadge = document.getElementById('catalogCountBadge');
  const elBtnToggleCatalog = document.getElementById('btnToggleCatalog');

  // Columna 1: Ficha del Bien y Martillero
  const elLotCategoryBadge = document.getElementById('lotCategoryBadge');
  const elLotInspectorBadge = document.getElementById('lotInspectorBadge');
  const elLotLocationBadge = document.getElementById('lotLocationBadge');
  const elMainLotImage = document.getElementById('mainLotImage');
  const elDefaultLotIconStage = document.getElementById('defaultLotIconStage');
  const elDefaultLotEmoji = document.getElementById('defaultLotEmoji');
  const elLotCertLinkWrap = document.getElementById('lotCertLinkWrap');
  const elLotIdTag = document.getElementById('lotIdTag');
  const elLotTitle = document.getElementById('lotTitle');
  const elLotDescription = document.getElementById('lotDescription');
  const elLotSpecsGrid = document.getElementById('lotSpecsGrid');

  const elEncargadoNombre = document.getElementById('encargadoNombre');
  const elEncargadoCargo = document.getElementById('encargadoCargo');
  const elEncargadoLicencia = document.getElementById('encargadoLicencia');
  const elEncargadoContacto = document.getElementById('encargadoContacto');
  const elEncargadoExperiencia = document.getElementById('encargadoExperiencia');

  // Columna 2: Arena de Ofertas
  const elMinutes = document.getElementById('minutes');
  const elSeconds = document.getElementById('seconds');
  const elCountdownDisplay = document.getElementById('countdownDisplay');
  const elAuctionPhaseTag = document.getElementById('auctionPhaseTag');
  const elAntiSnipingBanner = document.getElementById('antiSnipingBanner');
  const elCurrentPriceValue = document.getElementById('currentPriceValue');
  const elCurrentPriceDisplay = document.getElementById('currentPriceDisplay');
  const elCurrentLeaderName = document.getElementById('currentLeaderName');
  const elTotalBidsCount = document.getElementById('totalBidsCount');
  const elBidInput = document.getElementById('bidInput');
  const elBtnEmitirPuja = document.getElementById('btnEmitirPuja');
  const elFeedbackToast = document.getElementById('feedbackToast');
  const quickBidButtons = document.querySelectorAll('.btn-quick');

  // Columna 3: Concierge IA, Historial y Martillero
  const elCircuitBadge = document.getElementById('circuitBadge');
  const elAiScoreNumber = document.getElementById('aiScoreNumber');
  const elAiMeterFill = document.getElementById('aiMeterFill');
  const elAiJustificationText = document.getElementById('aiJustificationText');
  const elBtnToggleGemini = document.getElementById('btnToggleGemini');
  const elBidsTimeline = document.getElementById('bidsTimeline');
  const elBtnActivarSubasta = document.getElementById('btnActivarSubasta');
  const elBtnMartilloConteo = document.getElementById('btnMartilloConteo');
  const elBtnAdjudicar = document.getElementById('btnAdjudicar');

  // Modales (<dialog>)
  const authDialog = document.getElementById('authDialog');
  const lotManagerDialog = document.getElementById('lotManagerDialog');
  const adminDialog = document.getElementById('adminDialog');
  const stripeDialog = document.getElementById('stripeDialog');

  // Inicializar Asistente Lateral Gemini
  const assistantDrawerApi = inicializarDrawerAsistente();

  // Inicializar estado visual y datos reales desde PostgreSQL
  actualizarCabeceraUsuario();
  await cargarCatalogoSubastas();
  await sincronizarEstadoInicial();

  // ---------------------------------------------------------------------------
  // 3. CABECERA DE USUARIO, SESIÓN REAL Y PERMISOS POR PERFIL
  // ---------------------------------------------------------------------------
  function obtenerRolUsuario() {
    if (!state.authToken || !state.usuarioActual) return 'VISITANTE';
    return state.usuarioActual.role || state.usuarioActual.rol || 'POSTOR';
  }

  function aplicarPermisosVisualesPorRol() {
    const autenticado = Boolean(state.authToken && state.usuarioActual);
    const rol = obtenerRolUsuario();

    const esAdmin = autenticado && rol === 'ADMIN';
    const puedeGestionarLote = autenticado && ['VENDEDOR', 'MARTILLERO', 'ADMIN'].includes(rol);
    const puedeUsarMartillo = autenticado && ['MARTILLERO', 'ADMIN'].includes(rol);

    const btnAdmin = document.getElementById('btnOpenAdminModal');
    const btnLot = document.getElementById('btnOpenLotModal');
    const btnQuickImg = document.getElementById('btnQuickUploadImage');
    const btnQuickEnc = document.getElementById('btnEditEncargadoQuick');
    const gavelSection = document.getElementById('gavelConsoleSection');

    if (btnAdmin) btnAdmin.classList.toggle('hidden', !esAdmin);
    if (btnLot) btnLot.classList.toggle('hidden', !puedeGestionarLote);
    if (btnQuickImg) btnQuickImg.classList.toggle('hidden', !puedeGestionarLote);
    if (btnQuickEnc) btnQuickEnc.classList.toggle('hidden', !puedeGestionarLote);
    if (gavelSection) gavelSection.classList.toggle('hidden', !puedeUsarMartillo);
    if (elBtnLogoutUser) elBtnLogoutUser.classList.toggle('hidden', !autenticado);
  }

  function actualizarCabeceraUsuario() {
    const autenticado = Boolean(state.authToken && state.usuarioActual);
    if (autenticado) {
      const u = state.usuarioActual;
      const nombre = u.name || u.nombre || 'Usuario';
      const rol = obtenerRolUsuario();

      elNavUserName.textContent = nombre;
      elNavUserRoleBadge.textContent = rol;
      elUserInitials.textContent = nombre
        .split(' ')
        .filter(Boolean)
        .map((p) => p[0])
        .join('')
        .slice(0, 2)
        .toUpperCase() || 'US';

      elAuthBtnLabel.textContent = 'Mi Cuenta';
      if (state.garantiaActiva && state.garantiaActiva.estado === 'RETENIDA') {
        elEscrowBadge.textContent = `💳 Garantía: S/. ${formatearNumero(state.garantiaActiva.monto)} (RETENIDA)`;
      } else if (['VENDEDOR', 'MARTILLERO', 'ADMIN'].includes(rol)) {
        elEscrowBadge.textContent = `✓ Perfil habilitado para publicar bienes`;
      } else {
        elEscrowBadge.textContent = `Cuenta activa • Activa tu garantía para ofertar`;
      }
    } else {
      elNavUserName.textContent = 'Visitante';
      elNavUserRoleBadge.textContent = 'EXPLORANDO';
      elUserInitials.textContent = 'VS';
      elAuthBtnLabel.textContent = 'Ingresar / Crear Cuenta';
      elEscrowBadge.textContent = 'Inicia sesión para ofertar o vender';
    }

    aplicarPermisosVisualesPorRol();
    actualizarContextoAsistente();
  }

  if (elBtnLogoutUser) {
    elBtnLogoutUser.addEventListener('click', () => {
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
      mostrarToast('Has cerrado sesión correctamente.', 'success');
    });
  }

  function conectarWebSocketSala() {
    if (!state.authToken || !state.subastaId) return;
    if (state.ws && state.wsSubastaId === state.subastaId && state.ws.readyState <= WebSocket.OPEN) {
      return;
    }
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
            state.postorLiderNombre = `Participante ${ev.bidder_id}`;
            state.totalPujas++;
            sincronizarPrecioEnCatalogoLocal(state.subastaId, state.precioActual, null);
            actualizarPreciosUI();
            if (ev.extended) dispararReglaAntiSniping();
            agregarPujaAlTimeline(Number(ev.amount), `Participante ${ev.bidder_id}`, ev.bid_id || 'verificado');
          } else if (ev.type === 'auction.closed' && ev.auction_id === state.subastaId) {
            if (ev.winner_id) {
              state.postorLiderId = ev.winner_id;
              state.postorLiderNombre = `Participante ${ev.winner_id}`;
              state.precioActual = Number(ev.amount || state.precioActual);
            }
            sincronizarPrecioEnCatalogoLocal(state.subastaId, state.precioActual, 'CERRADA');
            finalizarSubastaPorTiempo();
          }
        } catch (_) {}
      });
    } catch (_) {}
  }

  function guardarSesion(token, user) {
    state.authToken = token;
    state.usuarioActual = user;
    localStorage.setItem('livebid_token', token);
    localStorage.setItem('livebid_user', JSON.stringify(user));
    actualizarCabeceraUsuario();
    sincronizarGarantiaActivaBackend();
    conectarWebSocketSala();
  }

  function obtenerTokenActual() {
    return state.authToken || '';
  }

  // ---------------------------------------------------------------------------
  // 4. CATÁLOGO INTERACTIVO DE BIENES EN SUBASTA (POSTGRESQL)
  // ---------------------------------------------------------------------------
  function iconoPorCategoria(catRaw) {
    const c = String(catRaw || '').toUpperCase();
    if (c.includes('VEHIC') || c.includes('AUTO') || c.includes('4X4')) return '🚘';
    if (c.includes('RELOJ') || c.includes('JOYA') || c.includes('ORO')) return '⌚';
    if (c.includes('ARTE') || c.includes('ANTIG') || c.includes('PINT')) return '🖼️';
    if (c.includes('INMUEB') || c.includes('CASA') || c.includes('PROP')) return '🏛️';
    return '💎';
  }

  function coincideFiltroCategoria(subasta, filtro) {
    if (!filtro || filtro === 'TODOS') return true;
    const cat = String(subasta?.detalles_lote?.categoria || '').toUpperCase();
    const titulo = String(subasta?.titulo || '').toUpperCase();
    const texto = `${cat} ${titulo}`;
    switch (filtro) {
      case 'VEHICULOS':
        return texto.includes('VEHIC') || texto.includes('AUTO') || texto.includes('4X4') || texto.includes('HILUX') || texto.includes('PORSCHE');
      case 'JOYAS':
        return texto.includes('RELOJ') || texto.includes('JOYA') || texto.includes('ROLEX') || texto.includes('PATEK') || texto.includes('ORO');
      case 'ARTE':
        return texto.includes('ARTE') || texto.includes('COLECC') || texto.includes('ANTIG') || texto.includes('ALTA_GAMA') || texto.includes('VALOR');
      case 'INMUEBLES':
        return texto.includes('INMUEB') || texto.includes('PENTHOUSE') || texto.includes('CASA') || texto.includes('TERRENO');
      default:
        return true;
    }
  }

  function formatearCategoriaHumana(catRaw) {
    if (!catRaw) return 'Lote Especial';
    return String(catRaw)
      .replace(/_/g, ' ')
      .toLowerCase()
      .replace(/\b\w/g, (l) => l.toUpperCase());
  }

  function renderizarVitrinaCatalogo() {
    if (!elCatalogGrid) return;
    const total = state.subastas.length;
    if (elCatalogCountBadge) {
      elCatalogCountBadge.textContent = `${total} ${total === 1 ? 'bien publicado' : 'bienes publicados'}`;
    }

    const filtradas = state.subastas.filter((s) => coincideFiltroCategoria(s, state.filtroCategoria));
    elCatalogGrid.innerHTML = '';

    if (filtradas.length === 0) {
      const emptyBox = document.createElement('article');
      emptyBox.className = 'catalog-card';
      const header = document.createElement('header');
      const h3 = document.createElement('h3');
      h3.textContent = 'No hay bienes en esta categoría todavía';
      header.appendChild(h3);
      const p = document.createElement('p');
      p.className = 'catalog-card-details';
      p.textContent = 'Puedes ver todas las categorías o consignar un nuevo bien con fotografías.';
      emptyBox.append(header, p);
      elCatalogGrid.appendChild(emptyBox);
      return;
    }

    filtradas.forEach((s) => {
      const det = s.detalles_lote || {};
      const imgUrl = Array.isArray(det.imagenes_urls) && det.imagenes_urls.length > 0 ? det.imagenes_urls[0] : '';
      const esActual = s.id === state.subastaId;
      const estado = s.estado || 'ACTIVA';
      const precioMostrar = s.precio_actual ?? s.precio_base ?? 0;

      // Raíz semántica <article class="catalog-card"> (Modern Web Guidance: card)
      const card = document.createElement('article');
      card.className = `catalog-card${esActual ? ' is-selected' : ''}`;

      // Región 1 (Media): <img> directo sin envoltorio .media, o .media para ícono de respaldo
      if (imgUrl) {
        const img = document.createElement('img');
        img.className = 'catalog-card-img';
        img.src = imgUrl;
        img.alt = s.titulo || 'Bien en subasta';
        img.loading = 'lazy';
        card.appendChild(img);
      } else {
        const media = document.createElement('div');
        media.className = 'media';
        media.setAttribute('aria-hidden', 'true');
        const emoji = document.createElement('span');
        emoji.className = 'media-emoji';
        emoji.textContent = iconoPorCategoria(det.categoria || s.titulo);
        const cap = document.createElement('span');
        cap.className = 'media-caption';
        cap.textContent = det.ubicacion || 'Exhibición Oficial';
        media.append(emoji, cap);
        card.appendChild(media);
      }

      // Región 2 (Header): insignias + título h3
      const header = document.createElement('header');
      const badges = document.createElement('div');
      badges.className = 'card-badges-row';

      const catTag = document.createElement('span');
      catTag.className = 'card-cat-tag';
      catTag.textContent = formatearCategoriaHumana(det.categoria);

      const statePill = document.createElement('span');
      if (estado === 'CERRADA') {
        statePill.className = 'card-state-pill closed';
        statePill.textContent = 'Adjudicado';
      } else if (estado === 'EN_CONTEO') {
        statePill.className = 'card-state-pill countdown';
        statePill.textContent = 'Últimos llamados';
      } else {
        statePill.className = 'card-state-pill live';
        statePill.textContent = 'En vivo';
      }

      badges.append(catTag, statePill);

      const h3 = document.createElement('h3');
      h3.textContent = s.titulo || 'Bien en Subasta';
      header.append(badges, h3);

      // Región 3 (Content): metadatos + precio actual
      const content = document.createElement('div');
      content.className = 'catalog-card-details';

      const meta = document.createElement('p');
      meta.className = 'catalog-card-meta';
      meta.textContent = `${det.ubicacion || 'Sala Lima'} • ${det.inspeccionado_por || 'Verificado'}`;

      const priceRow = document.createElement('div');
      priceRow.className = 'catalog-card-price-row';
      const priceLbl = document.createElement('span');
      priceLbl.className = 'catalog-card-price-label';
      priceLbl.textContent = estado === 'CERRADA' ? 'Precio final' : 'Oferta actual';
      const priceVal = document.createElement('strong');
      priceVal.className = 'catalog-card-price-val';
      priceVal.textContent = `S/. ${formatearNumero(precioMostrar)}`;
      priceRow.append(priceLbl, priceVal);

      content.append(meta, priceRow);

      // Región 4 (Footer): botón nativo para entrar a la sala de este bien
      const footer = document.createElement('footer');
      const btn = document.createElement('button');
      btn.type = 'button';
      btn.className = 'btn-enter-lot';
      btn.textContent = esActual ? '✓ Viendo en Sala Principal' : 'Entrar a esta Subasta';
      btn.addEventListener('click', async () => {
        state.subastaId = s.id;
        if (elAuctionSelector) elAuctionSelector.value = s.id;
        renderizarVitrinaCatalogo();
        await sincronizarEstadoInicial();
        document.getElementById('liveSalonMain')?.scrollIntoView({ behavior: 'smooth', block: 'start' });
      });
      footer.appendChild(btn);

      card.append(header, content, footer);
      elCatalogGrid.appendChild(card);
    });
  }

  function sincronizarPrecioEnCatalogoLocal(id, nuevoPrecio, nuevoEstado) {
    const item = state.subastas.find((x) => x.id === id);
    if (item) {
      if (nuevoPrecio) item.precio_actual = nuevoPrecio;
      if (nuevoEstado) item.estado = nuevoEstado;
      renderizarVitrinaCatalogo();
    }
  }

  document.querySelectorAll('[data-cat-filter]').forEach((chip) => {
    chip.addEventListener('click', () => {
      document.querySelectorAll('[data-cat-filter]').forEach((c) => c.classList.remove('active'));
      chip.classList.add('active');
      state.filtroCategoria = chip.getAttribute('data-cat-filter') || 'TODOS';
      renderizarVitrinaCatalogo();
    });
  });

  if (elBtnToggleCatalog && elCatalogGridWrap) {
    elBtnToggleCatalog.addEventListener('click', () => {
      state.vitrinaCompacta = !state.vitrinaCompacta;
      elCatalogGridWrap.classList.toggle('hidden', state.vitrinaCompacta);
      elBtnToggleCatalog.setAttribute('aria-expanded', String(!state.vitrinaCompacta));
      elBtnToggleCatalog.textContent = state.vitrinaCompacta ? 'Mostrar vitrina' : 'Compactar vitrina';
    });
  }

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
      lista.forEach((s) => {
        const opt = document.createElement('option');
        opt.value = s.id;
        const estadoTxt = s.estado === 'CERRADA' ? 'Cerrada' : 'En vivo';
        opt.textContent = `${s.titulo.slice(0, 32)} (${estadoTxt})`;
        if (s.id === state.subastaId) opt.selected = true;
        elAuctionSelector.appendChild(opt);
      });

      renderizarVitrinaCatalogo();
    } catch (_) {}
  }

  elAuctionSelector.addEventListener('change', async (e) => {
    state.subastaId = e.target.value;
    renderizarVitrinaCatalogo();
    await sincronizarEstadoInicial();
    mostrarToast(`Has ingresado a la sala del bien seleccionado.`, 'success');
  });

  async function sincronizarEstadoInicial() {
    state.seenBidIds.clear();
    try {
      const res = await fetch(`/api/v1/subastas/${state.subastaId}`);
      if (res.ok) {
        const data = await res.json();
        state.subastaActual = data;
        renderizarSubastaCompleta(data);
      }
      const histRes = await fetch(`/api/v1/subastas/${state.subastaId}/pujas?page=1&per_page=50`);
      if (histRes.ok) {
        const hist = await histRes.json();
        const items = Array.isArray(hist.items) ? hist.items : [];
        state.totalPujas = hist.total ?? items.length;
        renderizarHistorialInicial(items, state.precioBase);
        actualizarPreciosUI();
      }
    } catch (_) {
      actualizarPreciosUI();
    }
    await sincronizarGarantiaActivaBackend();
    conectarWebSocketSala();
    actualizarContextoAsistente();
  }

  function renderizarHistorialInicial(items, precioBase) {
    if (!elBidsTimeline) return;
    elBidsTimeline.innerHTML = '';
    if (!Array.isArray(items) || items.length === 0) {
      const baseItem = document.createElement('div');
      baseItem.className = 'bid-item winner-bid';
      baseItem.innerHTML = `
        <div class="bid-correlative">#0</div>
        <div class="bid-details">
          <div class="bid-row-top">
            <span class="bid-amount">S/. ${formatearNumero(precioBase)}</span>
            <span class="bid-time">Apertura</span>
          </div>
          <div class="bid-row-bottom">
            <span class="bid-user">Precio Base de Salida</span>
            <span class="audit-tag">✓ Verificado</span>
          </div>
        </div>
      `;
      elBidsTimeline.appendChild(baseItem);
      return;
    }

    items.forEach((b, idx) => {
      if (b.id) state.seenBidIds.add(b.id);
      const row = document.createElement('div');
      row.className = `bid-item${idx === 0 ? ' winner-bid' : ''}`;
      const correlativo = Math.max(1, (state.totalPujas || items.length) - idx);
      row.innerHTML = `
        <div class="bid-correlative">#${correlativo}</div>
        <div class="bid-details">
          <div class="bid-row-top">
            <span class="bid-amount">S/. ${formatearNumero(b.monto)}</span>
            <span class="bid-time">${b.recibida_en ? new Date(b.recibida_en).toLocaleTimeString('es-PE', { hour: '2-digit', minute: '2-digit' }) : 'Registrada'}</span>
          </div>
          <div class="bid-row-bottom">
            <span class="bid-user">Participante ${b.postor_id || ''}</span>
            <span class="audit-tag">✓ Oferta Sellada</span>
          </div>
        </div>
      `;
      elBidsTimeline.appendChild(row);
    });
  }

  function renderizarSubastaCompleta(data) {
    state.precioBase = data.precio_base ?? 45000;
    state.precioActual = data.precio_actual ?? state.precioBase;
    state.postorLiderId = data.postor_lider_id || 'NINGUNO';
    if (state.postorLiderId !== 'NINGUNO') {
      state.postorLiderNombre = `Participante ${state.postorLiderId}`;
    } else {
      state.postorLiderNombre = 'Esperando primera oferta en sala';
    }

    const estado = data.estado || 'ACTIVA';
    if (estado === 'CERRADA') {
      state.activa = false;
      state.tiempoSegundos = 0;
      elTopAuctionStateLabel.textContent = 'SALA CERRADA';
      elAuctionPhaseTag.textContent = 'ESTRADO CERRADO • LOTE ADJUDICADO';
    } else if (estado === 'EN_CONTEO') {
      state.activa = true;
      if (state.tiempoSegundos > 15 || state.tiempoSegundos <= 0) state.tiempoSegundos = 15;
      elTopAuctionStateLabel.textContent = 'ÚLTIMOS LLAMADOS';
      elAuctionPhaseTag.textContent = '🔔 ¡A LA UNA, A LAS DOS! ÚLTIMOS SEGUNDOS';
    } else {
      state.activa = true;
      if (data.cierre) {
        const diffSec = Math.floor((new Date(data.cierre).getTime() - Date.now()) / 1000);
        state.tiempoSegundos = diffSec > 0 ? Math.min(diffSec, 5999) : 120;
      } else {
        state.tiempoSegundos = 120;
      }
      elTopAuctionStateLabel.textContent = 'SALA EN VIVO';
      elAuctionPhaseTag.textContent = 'ESTRADO ABIERTO • PUJAS EN VIVO';
    }

    elLotIdTag.textContent = `LOTE OFICIAL #${data.id}`;
    elLotTitle.textContent = data.titulo || 'Bien en Subasta Oficial';
    elLotDescription.textContent =
      data.descripcion ||
      'Bien verificado por especialistas acreditados y disponible para ofertas en tiempo real.';

    const det = data.detalles_lote || {};
    elLotCategoryBadge.textContent = formatearCategoriaHumana(det.categoria || 'Catálogo General').toUpperCase();
    elLotLocationBadge.textContent = det.ubicacion || 'LIMA - PERÚ';
    elLotInspectorBadge.textContent = det.inspeccionado_por
      ? `★ TASADOR: ${det.inspeccionado_por}`
      : '★ BIEN VERIFICADO';

    if (elDefaultLotEmoji) {
      elDefaultLotEmoji.textContent = iconoPorCategoria(det.categoria || data.titulo);
    }

    const primeraImagen = Array.isArray(det.imagenes_urls) && det.imagenes_urls.length > 0 ? det.imagenes_urls[0] : '';
    if (primeraImagen) {
      elMainLotImage.src = primeraImagen;
      elMainLotImage.classList.remove('hidden');
      elDefaultLotIconStage.classList.add('hidden');
    } else {
      elMainLotImage.classList.add('hidden');
      elDefaultLotIconStage.classList.remove('hidden');
    }

    if (det.certificado_peritaje_url) {
      elLotCertLinkWrap.innerHTML = '';
      const link = document.createElement('a');
      link.href = det.certificado_peritaje_url;
      link.target = '_blank';
      link.rel = 'noopener noreferrer';
      link.style.color = 'var(--gold-champagne)';
      link.style.textDecoration = 'underline';
      link.textContent = '📄 Ver Certificado de Autenticidad / Tasación';
      elLotCertLinkWrap.appendChild(link);
    } else {
      elLotCertLinkWrap.textContent = 'EXHIBICIÓN Y TASACIÓN OFICIAL VERIFICADA';
    }

    const specs = det.especificaciones || {};
    const specEntries = Object.entries(specs);
    let specsHTML = `
      <div class="spec-box">
        <span class="spec-label">PRECIO DE APERTURA</span>
        <span class="spec-value highlight">S/. ${formatearNumero(state.precioBase)}</span>
      </div>
      <div class="spec-box">
        <span class="spec-label">VALOR DE RESERVA</span>
        <span class="spec-value">S/. ${formatearNumero(det.precio_reserva || state.precioBase)}</span>
      </div>
      <div class="spec-box">
        <span class="spec-label">LUGAR DE EXHIBICIÓN</span>
        <span class="spec-value">${escaparHTML(det.ubicacion || 'Sala Oficial Lima')}</span>
      </div>
      <div class="spec-box">
        <span class="spec-label">ESPECIALISTA</span>
        <span class="spec-value highlight-gold">${escaparHTML(det.inspeccionado_por || 'Especialista Aureum')}</span>
      </div>
    `;
    specEntries.slice(0, 4).forEach(([k, v]) => {
      specsHTML += `
        <div class="spec-box">
          <span class="spec-label">${escaparHTML(String(k).toUpperCase())}</span>
          <span class="spec-value">${escaparHTML(String(v))}</span>
        </div>
      `;
    });
    elLotSpecsGrid.innerHTML = specsHTML;

    const enc = data.encargado || {};
    elEncargadoNombre.textContent = enc.nombre || 'Martillero Oficial Asignado';
    elEncargadoCargo.textContent = enc.cargo || 'Martillero Público Acreditado';
    elEncargadoLicencia.textContent = enc.licencia ? `Lic. ${enc.licencia}` : `Ref. ${data.martillero_id || 'MART-01'}`;
    elEncargadoContacto.textContent = enc.contacto ? `📧 ${enc.contacto}` : 'Atención oficial en sala';
    elEncargadoExperiencia.textContent =
      enc.experiencia || 'Responsable de la transparencia y adjudicación oficial de la subasta.';

    document.getElementById('detCategoria').value = det.categoria || 'VEHICULOS_COLECCION';
    document.getElementById('detUbicacion').value = det.ubicacion || 'Sala de Exhibición Lima';
    document.getElementById('detReserva').value = det.precio_reserva || state.precioBase;
    document.getElementById('detInspector').value = det.inspeccionado_por || 'Ing. Roberto Salazar';
    document.getElementById('detCertUrl').value = det.certificado_peritaje_url || '';

    document.getElementById('encNombre').value = enc.nombre || 'Dr. Carlos Mendoza Ríos';
    document.getElementById('encMartilleroId').value = enc.martillero_id || data.martillero_id || 'MART-01';
    document.getElementById('encCargo').value = enc.cargo || 'Martillero Público Oficial SUNARP';
    document.getElementById('encLicencia').value = enc.licencia || 'REG-SUNARP-2026-0412';
    document.getElementById('encContacto').value = enc.contacto || 'cmendoza@livebid.pe | +51 998 112 334';
    document.getElementById('encExperiencia').value =
      enc.experiencia || '14 años dirigiendo subastas de vehículos, alta relojería, arte e inmuebles.';

    renderizarRelojInstantaneo();
    actualizarPreciosUI();
    renderizarVitrinaCatalogo();
    actualizarContextoAsistente();
  }

  // ---------------------------------------------------------------------------
  // 5. FLUJO PÚBLICO DE PUBLICACIÓN DE BIENES + MODAL DE ACCESO Y REGISTRO
  // ---------------------------------------------------------------------------
  function cambiarTabAuth(tab, modoVendedor = false) {
    document.querySelectorAll('[data-auth-tab]').forEach((b) => {
      b.classList.toggle('active', b.getAttribute('data-auth-tab') === tab);
    });
    document.getElementById('loginForm').classList.toggle('hidden', tab !== 'login');
    document.getElementById('registerForm').classList.toggle('hidden', tab !== 'register');
    const sellerBanner = document.getElementById('sellerOnboardingBanner');
    if (sellerBanner) sellerBanner.classList.toggle('hidden', !modoVendedor);
    if (modoVendedor) {
      const regRole = document.getElementById('regRole');
      if (regRole) regRole.value = 'VENDEDOR';
    }
  }

  function solicitarPublicarBien() {
    const rol = obtenerRolUsuario();
    if (state.authToken && ['VENDEDOR', 'MARTILLERO', 'ADMIN'].includes(rol)) {
      abrirModalLote('nueva');
      return;
    }
    if (!state.authToken) {
      cambiarTabAuth('register', true);
      authDialog.showModal();
      return;
    }
    mostrarToast(
      'Tu cuenta actual es de Comprador. Para consignar bienes, registra o solicita un perfil de Vendedor.',
      'error'
    );
    cambiarTabAuth('register', true);
    authDialog.showModal();
  }

  document.getElementById('btnPublishItemNav')?.addEventListener('click', solicitarPublicarBien);
  document.getElementById('btnPublishItemCatalog')?.addEventListener('click', solicitarPublicarBien);

  document.getElementById('btnOpenAuthModal').addEventListener('click', () => {
    cambiarTabAuth('login', false);
    authDialog.showModal();
  });
  document.getElementById('userProfileBadge').addEventListener('click', () => {
    cambiarTabAuth('login', false);
    authDialog.showModal();
  });

  document.querySelectorAll('[data-auth-tab]').forEach((btn) => {
    btn.addEventListener('click', () => {
      cambiarTabAuth(btn.getAttribute('data-auth-tab'), false);
    });
  });

  const btnFillAdmin = document.getElementById('btnFillAdminCreds');
  if (btnFillAdmin) {
    btnFillAdmin.addEventListener('click', () => {
      document.getElementById('loginEmail').value = 'admin@livebid.pe';
      document.getElementById('loginPassword').value = 'AdminLiveBid2026!';
    });
  }

  function mostrarFeedbackDialog(elId, msg, tipo = 'success') {
    const el = document.getElementById(elId);
    if (!el) return;
    el.textContent = msg;
    el.className = `dialog-feedback ${tipo}`;
    el.classList.remove('hidden');
  }

  document.getElementById('loginForm').addEventListener('submit', async (e) => {
    e.preventDefault();
    const email = document.getElementById('loginEmail').value.trim();
    const password = document.getElementById('loginPassword').value;

    try {
      const res = await fetch('/api/v1/auth/login', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ email, password })
      });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) {
        mostrarFeedbackDialog(
          'authFeedbackMsg',
          'No pudimos iniciar sesión: verifica tu correo y contraseña, o espera la activación de tu cuenta.',
          'error'
        );
        return;
      }

      guardarSesion(data.access_token, data.user);
      mostrarFeedbackDialog('authFeedbackMsg', `¡Bienvenido, ${data.user.name}!`, 'success');
      mostrarToast(`Sesión iniciada como ${data.user.name} (${data.user.role}).`, 'success');
      setTimeout(() => authDialog.close(), 500);
    } catch (_) {
      mostrarFeedbackDialog('authFeedbackMsg', 'No se pudo conectar con la plataforma en este momento.', 'error');
    }
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

    try {
      const res = await fetch('/api/v1/auth/register', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(payload)
      });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) {
        mostrarFeedbackDialog('authFeedbackMsg', `No se pudo crear la cuenta: ${data.error || 'revisa los datos ingresados'}`, 'error');
        return;
      }

      mostrarToast(
        `✅ ¡Cuenta creada! Tu correo ${data.email} quedó registrado con solicitud de rol ${data.requested_role || payload.rol_solicitado}. Un administrador verificará tu identidad para activarla.`,
        'success'
      );
      authDialog.close();
      e.target.reset();
      if (state.authToken && obtenerRolUsuario() === 'ADMIN') {
        abrirPanelAdmin();
      }
    } catch (_) {
      mostrarFeedbackDialog('authFeedbackMsg', 'Error de conexión al crear tu cuenta.', 'error');
    }
  });

  // ---------------------------------------------------------------------------
  // 6. PUBLICACIÓN Y CURADURÍA DE BIENES CON FOTOGRAFÍAS REALES
  // ---------------------------------------------------------------------------
  function abrirModalLote(tab = 'detalles') {
    document.querySelectorAll('[data-lot-tab]').forEach((b) => {
      b.classList.toggle('active', b.getAttribute('data-lot-tab') === tab);
    });
    document.getElementById('lotDetailsForm').classList.toggle('hidden', tab !== 'detalles');
    document.getElementById('encargadoForm').classList.toggle('hidden', tab !== 'encargado');
    document.getElementById('newAuctionForm').classList.toggle('hidden', tab !== 'nueva');
    lotManagerDialog.showModal();
  }

  document.getElementById('btnOpenLotModal').addEventListener('click', () => abrirModalLote('detalles'));
  document.getElementById('btnQuickUploadImage').addEventListener('click', () => abrirModalLote('detalles'));
  document.getElementById('btnEditEncargadoQuick').addEventListener('click', () => abrirModalLote('encargado'));

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
          const ctx = canvas.getContext('2d');
          ctx.drawImage(img, 0, 0, canvas.width, canvas.height);
          resolve(canvas.toDataURL('image/jpeg', 0.82));
        };
        img.onerror = reject;
        img.src = ev.target.result;
      };
      reader.onerror = reject;
      reader.readAsDataURL(file);
    });
  }

  const elLotImageFileInput = document.getElementById('lotImageFileInput');
  const elLotUploadPreviewImg = document.getElementById('lotUploadPreviewImg');
  const elLotUploadPlaceholder = document.getElementById('lotUploadPlaceholder');

  elLotImageFileInput.addEventListener('change', async (e) => {
    const file = e.target.files && e.target.files[0];
    if (!file) return;
    try {
      const dataUrl = await comprimirImagenADataURL(file);
      state.imagenReferencialDataUrl = dataUrl;
      elLotUploadPreviewImg.src = dataUrl;
      elLotUploadPreviewImg.classList.remove('hidden');
      elLotUploadPlaceholder.classList.add('hidden');
      mostrarFeedbackDialog('lotDialogFeedback', `✓ Fotografía "${file.name}" lista para guardar.`, 'success');
    } catch (_) {
      mostrarFeedbackDialog('lotDialogFeedback', 'No se pudo procesar la imagen seleccionada.', 'error');
    }
  });

  const elNewSubPreviewImg = document.getElementById('newSubPreviewImg');
  const elNewSubPreviewPlaceholder = document.getElementById('newSubPreviewPlaceholder');

  document.getElementById('newSubImageFile').addEventListener('change', async (e) => {
    const file = e.target.files && e.target.files[0];
    if (!file) return;
    try {
      const dataUrl = await comprimirImagenADataURL(file);
      state.nuevaSubastaImagenDataUrl = dataUrl;
      if (elNewSubPreviewImg && elNewSubPreviewPlaceholder) {
        elNewSubPreviewImg.src = dataUrl;
        elNewSubPreviewImg.classList.remove('hidden');
        elNewSubPreviewPlaceholder.classList.add('hidden');
      }
    } catch (_) {}
  });

  document.getElementById('newSubImageUrl')?.addEventListener('input', (e) => {
    const url = e.target.value.trim();
    if (url && !state.nuevaSubastaImagenDataUrl && elNewSubPreviewImg && elNewSubPreviewPlaceholder) {
      elNewSubPreviewImg.src = url;
      elNewSubPreviewImg.classList.remove('hidden');
      elNewSubPreviewPlaceholder.classList.add('hidden');
    }
  });

  document.getElementById('lotDetailsForm').addEventListener('submit', async (e) => {
    e.preventDefault();
    const token = obtenerTokenActual();
    const urlManual = document.getElementById('lotImageUrlInput').value.trim();
    const imagenes = [];
    if (state.imagenReferencialDataUrl) imagenes.push(state.imagenReferencialDataUrl);
    if (urlManual) imagenes.push(urlManual);
    if (imagenes.length === 0 && state.subastaActual?.detalles_lote?.imagenes_urls) {
      imagenes.push(...state.subastaActual.detalles_lote.imagenes_urls);
    }

    const specs = {};
    const specMotor = document.getElementById('detSpecMotor').value.trim();
    const specKm = document.getElementById('detSpecKm').value.trim();
    if (specMotor) specs['Característica'] = specMotor;
    if (specKm) specs['Conservación'] = specKm;

    const payload = {
      categoria: document.getElementById('detCategoria').value.trim(),
      ubicacion: document.getElementById('detUbicacion').value.trim(),
      precio_reserva: Number(document.getElementById('detReserva').value) || state.precioBase,
      inspeccionado_por: document.getElementById('detInspector').value.trim(),
      certificado_peritaje_url: document.getElementById('detCertUrl').value.trim(),
      imagenes_urls: imagenes,
      especificaciones: specs
    };

    try {
      const res = await fetch(`/api/v1/subastas/${state.subastaId}/detalles-lote`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`
        },
        body: JSON.stringify(payload)
      });
      const updated = await res.json().catch(() => ({}));
      if (!res.ok) {
        mostrarFeedbackDialog('lotDialogFeedback', `No se pudo guardar: ${updated.error || 'verifica tu perfil de vendedor'}`, 'error');
        return;
      }

      state.subastaActual = updated;
      await cargarCatalogoSubastas();
      renderizarSubastaCompleta(updated);
      mostrarToast('📷 Fotografía y ficha del bien actualizadas correctamente.', 'success');
      lotManagerDialog.close();
    } catch (_) {
      mostrarFeedbackDialog('lotDialogFeedback', 'Error de conexión al guardar los detalles del bien.', 'error');
    }
  });

  document.getElementById('encargadoForm').addEventListener('submit', async (e) => {
    e.preventDefault();
    const token = obtenerTokenActual();
    const payload = {
      martillero_id: document.getElementById('encMartilleroId').value.trim(),
      nombre: document.getElementById('encNombre').value.trim(),
      cargo: document.getElementById('encCargo').value.trim(),
      licencia: document.getElementById('encLicencia').value.trim(),
      contacto: document.getElementById('encContacto').value.trim(),
      experiencia: document.getElementById('encExperiencia').value.trim()
    };

    try {
      const res = await fetch(`/api/v1/subastas/${state.subastaId}/encargado`, {
        method: 'PUT',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`
        },
        body: JSON.stringify(payload)
      });
      const updated = await res.json().catch(() => ({}));
      if (!res.ok) {
        mostrarToast(`No se pudo actualizar el martillero responsable: ${updated.error || res.statusText}`, 'error');
        return;
      }

      state.subastaActual = updated;
      renderizarSubastaCompleta(updated);
      mostrarToast(`⚖️ Martillero responsable actualizado: ${payload.nombre}`, 'success');
      lotManagerDialog.close();
    } catch (_) {
      mostrarToast('Error de red al guardar los datos del martillero.', 'error');
    }
  });

  document.getElementById('newAuctionForm').addEventListener('submit', async (e) => {
    e.preventDefault();
    const token = obtenerTokenActual();
    const duracionMin = Number(document.getElementById('newSubDuracionMin').value) || 60;
    const inicio = new Date();
    const cierre = new Date(inicio.getTime() + duracionMin * 60 * 1000);
    const precioBase = Number(document.getElementById('newSubPrecioBase').value) || 35000;
    const reservaInput = Number(document.getElementById('newSubReserva')?.value || 0);
    const categoria = document.getElementById('newSubCategoria')?.value || 'OBJETOS_DE_VALOR';
    const ubicacion = document.getElementById('newSubUbicacion')?.value.trim() || 'Sala Principal San Isidro, Lima';
    const urlExterna = document.getElementById('newSubImageUrl')?.value.trim() || '';

    const imagenes = [];
    if (state.nuevaSubastaImagenDataUrl) imagenes.push(state.nuevaSubastaImagenDataUrl);
    if (urlExterna) imagenes.push(urlExterna);

    const nombreVendedor = state.usuarioActual?.name || 'Consignante Oficial Aureum';

    const payload = {
      titulo: document.getElementById('newSubTitulo').value.trim(),
      descripcion: document.getElementById('newSubDesc').value.trim(),
      precio_base: precioBase,
      inicio: inicio.toISOString(),
      cierre: cierre.toISOString(),
      encargado: {
        martillero_id: 'MART-01',
        nombre: document.getElementById('encNombre').value.trim() || 'Dr. Carlos Mendoza Ríos',
        cargo: 'Martillero Público Oficial SUNARP',
        licencia: 'REG-SUNARP-2026-0412',
        contacto: 'cmendoza@livebid.pe'
      },
      detalles_lote: {
        categoria,
        ubicacion,
        precio_reserva: reservaInput >= precioBase ? reservaInput : precioBase,
        inspeccionado_por: `Consignado por ${nombreVendedor}`,
        imagenes_urls: imagenes,
        especificaciones: {
          Consignante: nombreVendedor
        }
      }
    };

    try {
      const res = await fetch('/api/v1/subastas', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${token}`
        },
        body: JSON.stringify(payload)
      });
      const created = await res.json().catch(() => ({}));
      if (!res.ok) {
        mostrarToast(`No se pudo publicar el bien: ${created.error || res.statusText}`, 'error');
        return;
      }

      await fetch(`/api/v1/subastas/${created.id}/activar`, {
        method: 'POST',
        headers: { Authorization: `Bearer ${token}` }
      });

      state.nuevaSubastaImagenDataUrl = '';
      if (elNewSubPreviewImg && elNewSubPreviewPlaceholder) {
        elNewSubPreviewImg.classList.add('hidden');
        elNewSubPreviewPlaceholder.classList.remove('hidden');
      }
      e.target.reset();

      state.filtroCategoria = 'TODOS';
      document.querySelectorAll('[data-cat-filter]').forEach((c) => {
        c.classList.toggle('active', c.getAttribute('data-cat-filter') === 'TODOS');
      });

      state.subastaId = created.id;
      await cargarCatalogoSubastas();
      await sincronizarEstadoInicial();
      lotManagerDialog.close();
      mostrarToast(`✨ Tu bien "${created.titulo}" ya está publicado en el catálogo y recibiendo ofertas en vivo.`, 'success');
    } catch (_) {
      mostrarToast('Error de red al publicar el nuevo bien.', 'error');
    }
  });

  // ---------------------------------------------------------------------------
  // 7. PANEL DEL ADMINISTRADOR (VERIFICACIÓN DE IDENTIDAD Y PERFILES)
  // ---------------------------------------------------------------------------
  async function abrirPanelAdmin() {
    adminDialog.showModal();
    await cargarUsuariosAdmin();
  }

  document.getElementById('btnOpenAdminModal').addEventListener('click', abrirPanelAdmin);
  document.getElementById('btnRefreshUsers').addEventListener('click', cargarUsuariosAdmin);
  document.getElementById('adminFilterRole').addEventListener('change', cargarUsuariosAdmin);
  document.getElementById('adminFilterStatus').addEventListener('change', cargarUsuariosAdmin);

  async function cargarUsuariosAdmin() {
    const tbody = document.getElementById('adminUsersTableBody');
    const token = obtenerTokenActual();
    const roleFilter = document.getElementById('adminFilterRole').value;
    const statusFilter = document.getElementById('adminFilterStatus').value;

    const params = new URLSearchParams();
    if (roleFilter) params.set('rol', roleFilter);
    if (statusFilter) params.set('estado', statusFilter);

    try {
      const res = await fetch(`/api/v1/admin/users?${params.toString()}`, {
        headers: { Authorization: `Bearer ${token}` }
      });
      if (!res.ok) {
        tbody.innerHTML = `<tr><th scope="row">Acceso restringido</th><td colspan="5">Inicia sesión como administrador para gestionar cuentas.</td></tr>`;
        return;
      }
      const users = await res.json();
      if (!Array.isArray(users) || users.length === 0) {
        tbody.innerHTML = `<tr><th scope="row">Sin resultados</th><td colspan="5">No hay usuarios que coincidan con el filtro seleccionado.</td></tr>`;
        return;
      }

      tbody.innerHTML = '';
      users.forEach((u) => {
        const tr = document.createElement('tr');
        const reqRole = u.requested_role || u.role;
        const kycText = u.kyc_aprobado ? '✓ Identidad Verificada' : '⏳ Pendiente de Revisión';

        tr.innerHTML = `
          <th scope="row">
            <div>${escaparHTML(u.name)}</div>
            <div style="font-size:0.7rem; color:var(--text-muted); font-weight:400;">${escaparHTML(u.email)}</div>
          </th>
          <td>
            <div>${escaparHTML(u.documento_identidad || 'Sin documento')}</div>
            <div style="font-size:0.7rem; color:var(--text-muted);">${escaparHTML(u.telefono || 'Sin teléfono')}</div>
          </td>
          <td>
            <span class="tier-pill">${escaparHTML(reqRole)}</span>
          </td>
          <td>
            <select class="admin-role-select" data-user-id="${escaparHTML(u.id)}" aria-label="Asignar perfil a ${escaparHTML(u.name)}" style="background:#070a10; color:#fff; border:1px solid var(--gold-soft-border); padding:4px 8px; border-radius:6px; font-size:0.74rem;">
              <option value="POSTOR" ${u.role === 'POSTOR' ? 'selected' : ''}>POSTOR</option>
              <option value="VENDEDOR" ${u.role === 'VENDEDOR' ? 'selected' : ''}>VENDEDOR</option>
              <option value="MARTILLERO" ${u.role === 'MARTILLERO' ? 'selected' : ''}>MARTILLERO</option>
              <option value="ADMIN" ${u.role === 'ADMIN' ? 'selected' : ''}>ADMIN</option>
            </select>
          </td>
          <td>
            <div><strong>${escaparHTML(u.status)}</strong></div>
            <div style="font-size:0.7rem; color:${u.kyc_aprobado ? '#6ee7b7' : '#fde68a'};">${kycText}</div>
          </td>
          <td>
            <div class="admin-action-group">
              <button type="button" class="btn-table-act approve" data-action="approve" data-id="${escaparHTML(u.id)}" data-req-role="${escaparHTML(reqRole)}">
                ✅ Verificar y Activar
              </button>
              <button type="button" class="btn-table-act suspend" data-action="suspend" data-id="${escaparHTML(u.id)}">
                ⏸️ Suspender
              </button>
              <button type="button" class="btn-table-act block" data-action="block" data-id="${escaparHTML(u.id)}">
                🚫 Bloquear
              </button>
            </div>
          </td>
        `;
        tbody.appendChild(tr);
      });

      tbody.querySelectorAll('.admin-role-select').forEach((sel) => {
        sel.addEventListener('change', async (ev) => {
          const userId = ev.target.getAttribute('data-user-id');
          const nuevoRol = ev.target.value;
          const r = await fetch(`/api/v1/admin/users/${userId}/role`, {
            method: 'PATCH',
            headers: {
              'Content-Type': 'application/json',
              Authorization: `Bearer ${state.authToken}`
            },
            body: JSON.stringify({ rol: nuevoRol })
          });
          if (r.ok) {
            mostrarFeedbackDialog('adminFeedbackMsg', `✓ Perfil del usuario ${userId} actualizado a ${nuevoRol}.`, 'success');
            await cargarUsuariosAdmin();
          } else {
            mostrarFeedbackDialog('adminFeedbackMsg', 'No se pudo actualizar el perfil.', 'error');
          }
        });
      });

      tbody.querySelectorAll('[data-action]').forEach((btn) => {
        btn.addEventListener('click', async () => {
          const action = btn.getAttribute('data-action');
          const userId = btn.getAttribute('data-id');
          const reqRole = btn.getAttribute('data-req-role');

          if (action === 'approve') {
            if (reqRole && reqRole !== 'ADMIN') {
              await fetch(`/api/v1/admin/users/${userId}/role`, {
                method: 'PATCH',
                headers: {
                  'Content-Type': 'application/json',
                  Authorization: `Bearer ${state.authToken}`
                },
                body: JSON.stringify({ rol: reqRole })
              });
            }
            await fetch(`/api/v1/admin/users/${userId}/status`, {
              method: 'PATCH',
              headers: {
                'Content-Type': 'application/json',
                Authorization: `Bearer ${state.authToken}`
              },
              body: JSON.stringify({ estado: 'ACTIVO', kyc_aprobado: true })
            });
            mostrarFeedbackDialog('adminFeedbackMsg', `✓ Usuario ${userId} activado con KYC aprobado y perfil ${reqRole}.`, 'success');
          } else if (action === 'suspend') {
            await fetch(`/api/v1/admin/users/${userId}/status`, {
              method: 'PATCH',
              headers: {
                'Content-Type': 'application/json',
                Authorization: `Bearer ${state.authToken}`
              },
              body: JSON.stringify({ estado: 'SUSPENDIDO', kyc_aprobado: false })
            });
            mostrarFeedbackDialog('adminFeedbackMsg', `⚠️ Cuenta ${userId} suspendida temporalmente.`, 'error');
          } else if (action === 'block') {
            await fetch(`/api/v1/admin/users/${userId}/status`, {
              method: 'PATCH',
              headers: {
                'Content-Type': 'application/json',
                Authorization: `Bearer ${state.authToken}`
              },
              body: JSON.stringify({ estado: 'BLOQUEADO', kyc_aprobado: false })
            });
            mostrarFeedbackDialog('adminFeedbackMsg', `🚫 Cuenta ${userId} bloqueada.`, 'error');
          }
          await cargarUsuariosAdmin();
        });
      });
    } catch (_) {
      mostrarFeedbackDialog('adminFeedbackMsg', 'Error al consultar el directorio de cuentas.', 'error');
    }
  }

  // ---------------------------------------------------------------------------
  // 8. CRONÓMETRO REGRESIVO Y EXTENSIÓN DE CIERRE JUSTO (+30 s)
  // ---------------------------------------------------------------------------
  function renderizarRelojInstantaneo() {
    const m = Math.floor(Math.max(0, state.tiempoSegundos) / 60);
    const s = Math.max(0, state.tiempoSegundos) % 60;
    elMinutes.textContent = String(m).padStart(2, '0');
    elSeconds.textContent = String(s).padStart(2, '0');
  }

  function actualizarCronometro() {
    if (!state.activa) return;

    if (state.tiempoSegundos > 0) {
      state.tiempoSegundos--;
    } else {
      finalizarSubastaPorTiempo();
      return;
    }

    renderizarRelojInstantaneo();

    if (state.tiempoSegundos <= 15) {
      elCountdownDisplay.style.borderColor = 'rgba(245, 158, 11, 0.9)';
      elCountdownDisplay.style.boxShadow = '0 0 30px rgba(245, 158, 11, 0.4)';
    } else {
      elCountdownDisplay.style.borderColor = '';
      elCountdownDisplay.style.boxShadow = '';
    }
  }

  function dispararReglaAntiSniping() {
    if (state.tiempoSegundos <= 15) {
      state.tiempoSegundos += 30;
      renderizarRelojInstantaneo();
      elAntiSnipingBanner.classList.remove('hidden');
      elCountdownDisplay.classList.add('price-bump');
      setTimeout(() => elCountdownDisplay.classList.remove('price-bump'), 600);
      setTimeout(() => elAntiSnipingBanner.classList.add('hidden'), 5000);
      mostrarToast('⚡ Cierre justo activado: se añadieron +30 segundos al reloj de sala.', 'success');
    }
  }

  function finalizarSubastaPorTiempo() {
    state.activa = false;
    elMinutes.textContent = '00';
    elSeconds.textContent = '00';
    elAuctionPhaseTag.textContent = 'ESTRADO CERRADO • LOTE ADJUDICADO';
    elTopAuctionStateLabel.textContent = 'SALA CERRADA';
    mostrarToast(
      `🔨 Subasta finalizada: bien adjudicado a ${state.postorLiderNombre} por S/. ${formatearNumero(state.precioActual)}`,
      'success'
    );
  }

  state.timerInterval = setInterval(actualizarCronometro, 1000);

  // ---------------------------------------------------------------------------
  // 9. EMISIÓN DE OFERTAS EN VIVO Y CONTROLES DEL MARTILLERO
  // ---------------------------------------------------------------------------
  function actualizarPreciosUI() {
    elCurrentPriceValue.textContent = formatearNumero(state.precioActual);
    elCurrentLeaderName.textContent = state.postorLiderNombre;
    elTotalBidsCount.textContent = `${state.totalPujas} oferta${state.totalPujas === 1 ? '' : 's'}`;

    const siguienteMinimo = state.precioActual + state.incrementoMinimo;
    elBidInput.min = siguienteMinimo;
    elBidInput.value = siguienteMinimo;

    elCurrentPriceDisplay.classList.add('price-bump');
    setTimeout(() => elCurrentPriceDisplay.classList.remove('price-bump'), 350);
    actualizarContextoAsistente();
  }

  quickBidButtons.forEach((btn) => {
    btn.addEventListener('click', () => {
      const incremento = Number(btn.getAttribute('data-add'));
      elBidInput.value = state.precioActual + incremento;
      elBidInput.focus();
    });
  });

  elBtnEmitirPuja.addEventListener('click', async () => {
    if (!state.authToken || !state.usuarioActual) {
      mostrarToast('Inicia sesión o crea tu cuenta para realizar una oferta en vivo.', 'error');
      cambiarTabAuth('login', false);
      authDialog.showModal();
      return;
    }

    const monto = Number(elBidInput.value);
    if (monto < state.precioActual + state.incrementoMinimo) {
      mostrarToast(
        `Tu oferta debe ser de al menos S/. ${formatearNumero(state.precioActual + state.incrementoMinimo)}.`,
        'error'
      );
      return;
    }

    const userId = state.usuarioActual.id || '';
    const userName = state.usuarioActual.name || state.usuarioActual.nombre || 'Postor';

    if (userId && userId === state.postorLiderId) {
      mostrarToast(
        `Ya tienes la oferta más alta en sala con S/. ${formatearNumero(state.precioActual)}.`,
        'error'
      );
      return;
    }

    elBtnEmitirPuja.disabled = true;
    try {
      const headers = {
        'Content-Type': 'application/json',
        'Idempotency-Key': `idem-${state.subastaId}-${userId}-${monto}-${Date.now()}`,
        Authorization: `Bearer ${state.authToken}`
      };

      const res = await fetch(`/api/v1/subastas/${state.subastaId}/pujas`, {
        method: 'POST',
        headers,
        body: JSON.stringify({ monto })
      });

      const data = await res.json().catch(() => ({}));
      if (res.ok) {
        if (data.puja_id) state.seenBidIds.add(data.puja_id);
        procesarPujaExitosa(monto, userName, userId, data.hash_auditoria || 'verificado');
      } else if (res.status === 409 && String(data.error || '').toLowerCase().includes('guarantee')) {
        mostrarToast('Antes de ofertar en este bien, activa tu depósito de garantía reembolsable.', 'error');
        abrirModalStripe();
      } else {
        mostrarToast(`No se pudo registrar la oferta: ${data.error || res.statusText}`, 'error');
      }
    } catch (_) {
      mostrarToast('Error de conexión al enviar tu oferta.', 'error');
    } finally {
      elBtnEmitirPuja.disabled = false;
    }
  });

  function procesarPujaExitosa(monto, postorNombre, postorId, hash) {
    state.precioActual = monto;
    state.postorLiderId = postorId;
    state.postorLiderNombre = postorNombre;
    state.totalPujas++;

    sincronizarPrecioEnCatalogoLocal(state.subastaId, monto, null);
    actualizarPreciosUI();
    dispararReglaAntiSniping();
    agregarPujaAlTimeline(monto, postorNombre, hash);
    mostrarToast(`✓ ¡Tu oferta de S/. ${formatearNumero(monto)} lidera la sala!`, 'success');
  }

  function agregarPujaAlTimeline(monto, postor, hash) {
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
          <span class="bid-user">${escaparHTML(postor)}</span>
          <span class="audit-tag" title="Certificado: ${escaparHTML(hash)}">✓ Oferta Sellada</span>
        </div>
      </div>
    `;
    const prevWinner = elBidsTimeline.querySelector('.winner-bid');
    if (prevWinner) prevWinner.classList.remove('winner-bid');
    elBidsTimeline.insertBefore(item, elBidsTimeline.firstChild);
  }

  elBtnActivarSubasta.addEventListener('click', async () => {
    const token = obtenerTokenActual();
    await fetch(`/api/v1/subastas/${state.subastaId}/activar`, {
      method: 'POST',
      headers: { Authorization: `Bearer ${token}` }
    });
    state.activa = true;
    state.tiempoSegundos = 120;
    await cargarCatalogoSubastas();
    await sincronizarEstadoInicial();
    mostrarToast('🟢 Sala reabierta para recibir ofertas.', 'success');
  });

  elBtnMartilloConteo.addEventListener('click', async () => {
    const token = obtenerTokenActual();
    await fetch(`/api/v1/subastas/${state.subastaId}/conteo`, {
      method: 'POST',
      headers: { Authorization: `Bearer ${token}` }
    });
    if (state.tiempoSegundos > 15) state.tiempoSegundos = 15;
    await cargarCatalogoSubastas();
    await sincronizarEstadoInicial();
    mostrarToast('🔔 ¡Últimos llamados antes de adjudicar el bien!', 'success');
  });

  elBtnAdjudicar.addEventListener('click', async () => {
    const token = obtenerTokenActual();
    await fetch(`/api/v1/subastas/${state.subastaId}/cerrar`, {
      method: 'POST',
      headers: { Authorization: `Bearer ${token}` }
    });
    finalizarSubastaPorTiempo();
    await cargarCatalogoSubastas();
    await sincronizarEstadoInicial();
  });

  // ---------------------------------------------------------------------------
  // 10. DEPÓSITO DE GARANTÍA REEMBOLSABLE (STRIPE HOLD)
  // ---------------------------------------------------------------------------
  const elGuaranteeStatusNote = document.getElementById('guaranteeStatusNote');
  const elStripeSummaryAuction = document.getElementById('stripeSummaryAuction');
  const elStripeSummaryRisk = document.getElementById('stripeSummaryRisk');
  const elStripeSummaryAmount = document.getElementById('stripeSummaryAmount');
  const elStripeSummaryIntentId = document.getElementById('stripeSummaryIntentId');
  const elStripeHoldAmountInput = document.getElementById('stripeHoldAmountInput');
  const elStripeCardHolder = document.getElementById('stripeCardHolder');

  async function sincronizarGarantiaActivaBackend() {
    if (!state.authToken) {
      state.garantiaActiva = null;
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
        if (elGuaranteeStatusNote) {
          elGuaranteeStatusNote.innerHTML = `✓ <strong>Garantía Activa:</strong> Tienes S/. ${formatearNumero(g.monto)} reservados de forma reembolsable para ofertar libremente en este lote.`;
        }
      } else {
        state.garantiaActiva = null;
        if (elGuaranteeStatusNote) {
          elGuaranteeStatusNote.innerHTML = `<strong>Participación Protegida:</strong> Antes de ofertar, activa tu depósito de garantía reembolsable (10% del precio base). Si no ganas, se libera sin costo.`;
        }
        actualizarCabeceraUsuario();
      }
    } catch (_) {}
    actualizarContextoAsistente();
  }

  async function prepararStripePaymentIntent() {
    const requeridoDefault = Math.round((state.precioBase || 45000) * 0.1);
    elStripeSummaryAuction.textContent = state.subastaActual?.titulo || state.subastaId;
    elStripeSummaryAmount.textContent = `S/. ${formatearNumero(requeridoDefault)}`;
    elStripeHoldAmountInput.value = requeridoDefault;
    elStripeCardHolder.value = state.usuarioActual?.name || '';

    if (!state.authToken) {
      elStripeSummaryIntentId.textContent = 'Inicia sesión para continuar';
      mostrarFeedbackDialog(
        'stripeFeedbackMsg',
        'Primero inicia sesión con tu cuenta para activar tu garantía reembolsable.',
        'error'
      );
      return;
    }

    try {
      const montoSolicitado = Number(elStripeHoldAmountInput.value) || requeridoDefault;
      const res = await fetch(`/api/v1/subastas/${state.subastaId}/garantias/stripe-intent`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${state.authToken}`
        },
        body: JSON.stringify({ monto: montoSolicitado })
      });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) {
        if (res.status === 409) {
          mostrarFeedbackDialog(
            'stripeFeedbackMsg',
            '✓ Ya cuentas con tu garantía activa para ofertar en este bien.',
            'success'
          );
          await sincronizarGarantiaActivaBackend();
          return;
        }
        mostrarFeedbackDialog('stripeFeedbackMsg', `No se pudo preparar la autorización: ${data.error || 'intenta nuevamente'}`, 'error');
        return;
      }

      state.stripeIntentActual = data;
      const pct = Math.round((data.porcentaje_garantia || 0.1) * 100);
      elStripeSummaryAuction.textContent = state.subastaActual?.titulo || data.subasta_id || state.subastaId;
      elStripeSummaryRisk.textContent = `Calificación ${data.score_confianza}/100 • Depósito del ${pct}%`;
      elStripeSummaryAmount.textContent = `S/. ${formatearNumero(data.monto_requerido)}`;
      elStripeSummaryIntentId.textContent = `${data.payment_intent_id}`;
      elStripeHoldAmountInput.min = data.monto_requerido;
      elStripeHoldAmountInput.value = data.monto_retencion || data.monto_requerido;

      mostrarFeedbackDialog(
        'stripeFeedbackMsg',
        '✓ Autorización lista. Pulsa «Autorizar Garantía Reembolsable» para quedar habilitado en sala.',
        'success'
      );
    } catch (_) {
      mostrarFeedbackDialog('stripeFeedbackMsg', 'Error de conexión con la pasarela de garantías.', 'error');
    }
  }

  async function abrirModalStripe() {
    if (stripeDialog) {
      stripeDialog.showModal();
      await prepararStripePaymentIntent();
    }
  }

  document.getElementById('btnOpenStripeModal')?.addEventListener('click', abrirModalStripe);
  document.getElementById('btnOpenStripeNav')?.addEventListener('click', abrirModalStripe);
  document.getElementById('btnRecalcStripeIntent')?.addEventListener('click', prepararStripePaymentIntent);

  document.getElementById('stripeGuaranteeForm').addEventListener('submit', async (e) => {
    e.preventDefault();
    if (!state.authToken) {
      mostrarFeedbackDialog('stripeFeedbackMsg', 'Inicia sesión antes de autorizar tu garantía.', 'error');
      return;
    }

    const monto = Number(elStripeHoldAmountInput.value) || Math.round(state.precioBase * 0.1);
    const paymentIntentId = state.stripeIntentActual?.payment_intent_id || '';

    try {
      const res = await fetch(`/api/v1/subastas/${state.subastaId}/garantias`, {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${state.authToken}`
        },
        body: JSON.stringify({
          monto,
          payment_intent_id: paymentIntentId
        })
      });
      const data = await res.json().catch(() => ({}));
      if (!res.ok) {
        mostrarFeedbackDialog(
          'stripeFeedbackMsg',
          `No se pudo autorizar la garantía: ${data.error || res.statusText}`,
          'error'
        );
        return;
      }

      state.garantiaActiva = data;
      actualizarCabeceraUsuario();
      if (elGuaranteeStatusNote) {
        elGuaranteeStatusNote.innerHTML = `✓ <strong>Garantía Activa:</strong> Tienes S/. ${formatearNumero(data.monto)} reservados de forma reembolsable para ofertar libremente en este lote.`;
      }
      mostrarToast(
        `💳 Garantía reembolsable RETENIDA (S/. ${formatearNumero(data.monto)}). ¡Ya puedes ofertar en sala!`,
        'success'
      );
      stripeDialog.close();
    } catch (_) {
      mostrarFeedbackDialog('stripeFeedbackMsg', 'Error de red al confirmar tu garantía.', 'error');
    }
  });

  if (elBtnToggleGemini) {
    elBtnToggleGemini.addEventListener('click', () => {
      state.geminiOnline = !state.geminiOnline;
      if (!state.geminiOnline) {
        elCircuitBadge.className = 'circuit-badge open';
        elCircuitBadge.textContent = 'MODO LOCAL';
        elAiScoreNumber.innerHTML = '75<small>/100 (Estándar)</small>';
        elAiMeterFill.style.width = '75%';
        elAiJustificationText.innerHTML =
          'Modo de respaldo local activo: la sala continúa operando sin interrupciones.';
      } else {
        elCircuitBadge.className = 'circuit-badge closed';
        elCircuitBadge.textContent = 'IA ACTIVA';
        elAiScoreNumber.innerHTML = '92<small>/100 (Preferente)</small>';
        elAiMeterFill.style.width = '92%';
        elAiJustificationText.innerHTML =
          '✓ Perfil elegible para garantía reducida del <strong>10% reembolsable</strong> sobre el precio base.';
      }
    });
  }

  // ---------------------------------------------------------------------------
  // 11. ASISTENTE LATERAL CON GOOGLE GEMINI (navigation-drawer + /api/v1/ia/asistente)
  // ---------------------------------------------------------------------------
  function inicializarDrawerAsistente() {
    const drawer = document.getElementById('aiAssistantDrawer');
    const scroller = drawer?.querySelector('.Drawer-scroller');
    const sheet = drawer?.querySelector('.Drawer-sheet');
    const closeBtn = document.getElementById('btnCloseAiDrawer');
    const openTriggers = [
      document.getElementById('btnOpenAiDrawer'),
      document.getElementById('btnFloatingAiAssistant'),
      document.getElementById('btnOpenAiFromCard')
    ].filter(Boolean);

    if (!drawer || !scroller || !sheet) {
      return { open: () => {}, close: () => {} };
    }

    const setExpanded = (val) => {
      openTriggers.forEach((btn) => {
        if (btn.hasAttribute('aria-expanded')) {
          btn.setAttribute('aria-expanded', String(val));
        }
      });
    };

    async function openDrawer() {
      drawer.showModal();
      actualizarContextoAsistente();

      // Si el navegador no soporta scroll-initial-target, posicionamos en el tope cerrado (left: 0)
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

    // Fallback progresivo para animación de backdrop donde animation-timeline no esté disponible
    if (!CSS.supports('animation-timeline: scroll()')) {
      scroller.addEventListener('scroll', () => {
        const w = sheet.offsetWidth || 1;
        const ratio = Math.min(1, Math.max(0, scroller.scrollLeft / w));
        drawer.style.setProperty('--drawer-backdrop', String(ratio));
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

    openTriggers.forEach((btn) => btn.addEventListener('click', openDrawer));
    closeBtn?.addEventListener('click', closeDrawer);

    // Light-dismiss al tocar fuera de la hoja (nunca usar closedby="any" en un drawer de pantalla completa)
    drawer.addEventListener('click', (ev) => {
      if (!sheet.contains(ev.target)) {
        closeDrawer();
      }
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
    const titulo = state.subastaActual?.titulo || 'Lote en sala';
    elCtx.textContent = `Viendo: ${titulo} • Oferta actual S/. ${formatearNumero(state.precioActual)}`;
  }

  function agregarMensajeChat(rol, texto, autorLabel = '') {
    const container = document.getElementById('aiChatMessages');
    if (!container) return;

    const bubble = document.createElement('div');
    bubble.className = `ai-msg ${rol === 'user' ? 'user' : 'assistant'}`;

    const author = document.createElement('div');
    author.className = 'ai-msg-author';
    author.textContent = rol === 'user' ? 'Tú' : (autorLabel || '✨ Asistente Aureum');

    const body = document.createElement('div');
    body.className = 'ai-msg-text';
    // Renderizado seguro línea por línea admitiendo **negrita** sin inyectar HTML crudo
    const fragments = String(texto).split(/(\*\*[^*]+\*\*)/g);
    fragments.forEach((part) => {
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

  function renderizarSugerenciasAsistente(sugerencias) {
    const wrap = document.getElementById('aiSuggestionChips');
    if (!wrap || !Array.isArray(sugerencias) || sugerencias.length === 0) return;
    wrap.innerHTML = '';
    sugerencias.slice(0, 4).forEach((s) => {
      const chip = document.createElement('button');
      chip.type = 'button';
      chip.className = 'ai-chip';
      chip.textContent = s;
      chip.addEventListener('click', () => enviarConsultaAsistente(s));
      wrap.appendChild(chip);
    });
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
    if (badge) badge.textContent = 'Pensando...';

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
      if (!res.ok || !data.respuesta) {
        agregarMensajeChat('assistant', 'En este momento no pude procesar tu pregunta. Inténtalo de nuevo en unos segundos.');
        if (badge) badge.textContent = 'En línea';
        return;
      }

      const etiquetaModelo = data.modo_local ? '✨ Concierge Aureum' : '✨ Google Gemini';
      agregarMensajeChat('assistant', data.respuesta, etiquetaModelo);
      if (badge) {
        badge.textContent = data.modo_local ? 'Guía Contextual' : 'Gemini Activo';
      }

      state.aiHistory.push({ rol: 'user', texto: mensaje });
      state.aiHistory.push({ rol: 'assistant', texto: data.respuesta });
      if (state.aiHistory.length > 10) {
        state.aiHistory = state.aiHistory.slice(-10);
      }

      if (Array.isArray(data.sugerencias)) {
        renderizarSugerenciasAsistente(data.sugerencias);
      }
    } catch (_) {
      agregarMensajeChat('assistant', 'Hubo un inconveniente de conexión. Verifica tu red e inténtalo otra vez.');
      if (badge) badge.textContent = 'En línea';
    } finally {
      if (sendBtn) sendBtn.disabled = false;
    }
  }

  document.getElementById('aiChatForm')?.addEventListener('submit', (e) => {
    e.preventDefault();
    const input = document.getElementById('aiChatInput');
    if (input && input.value.trim()) {
      enviarConsultaAsistente(input.value);
    }
  });

  document.querySelectorAll('#aiSuggestionChips [data-ai-prompt]').forEach((chip) => {
    chip.addEventListener('click', () => {
      enviarConsultaAsistente(chip.getAttribute('data-ai-prompt'));
    });
  });

  // Exponer apertura rápida por teclado (Alt+A) si no se está escribiendo en un campo
  document.addEventListener('keydown', (ev) => {
    if (ev.altKey && ev.key.toLowerCase() === 'a') {
      ev.preventDefault();
      assistantDrawerApi.open();
    }
  });

  // ---------------------------------------------------------------------------
  // 12. VALIDACIÓN ACCESIBLE POS-INTERACCIÓN (:user-invalid + WeakMap Fallback)
  // ---------------------------------------------------------------------------
  function inicializarValidacionAccesible() {
    const linkError = (el, shown) => {
      const errorId = el.dataset?.error;
      if (!errorId) return;
      const ids = (el.getAttribute('aria-describedby') || '')
        .split(/\s+/)
        .filter((id) => id && id !== errorId);
      if (shown) ids.push(errorId);
      if (ids.length) el.setAttribute('aria-describedby', ids.join(' '));
      else el.removeAttribute('aria-describedby');
    };

    if (!CSS.supports('selector(:user-invalid)')) {
      const dirtyState = new WeakMap();
      const updateFallback = (input) => {
        const isValid = input.validity.valid;
        input.classList.toggle('user-invalid-fallback', !isValid);
        input.classList.toggle('user-valid-fallback', isValid);
        input.setAttribute('aria-invalid', isValid ? 'false' : 'true');
        linkError(input, !isValid);
      };

      document.querySelectorAll('input, select, textarea').forEach((el) => {
        if (!el.hasAttribute('aria-invalid')) el.setAttribute('aria-invalid', 'false');
      });

      const onEv = (ev) => {
        const input = ev.target;
        if (ev.type === 'reset') {
          for (const c of input.elements || []) {
            dirtyState.delete(c);
            c.classList.remove('user-invalid-fallback', 'user-valid-fallback');
            if (c.matches?.('input, select, textarea')) c.setAttribute('aria-invalid', 'false');
            linkError(c, false);
          }
          return;
        }
        if (!input?.checkValidity) return;
        if (ev.type === 'invalid') {
          dirtyState.set(input, { hasInteracted: true, hasBlurred: true });
          updateFallback(input);
          return;
        }
        const st = dirtyState.get(input) || { hasInteracted: false, hasBlurred: false };
        if (ev.type === 'input' || ev.type === 'change') st.hasInteracted = true;
        if (ev.type === 'blur') st.hasBlurred = true;
        dirtyState.set(input, st);
        if (st.hasInteracted && st.hasBlurred) updateFallback(input);
      };

      document.addEventListener('blur', onEv, true);
      document.addEventListener('input', onEv);
      document.addEventListener('change', onEv);
      document.addEventListener('invalid', onEv, true);
      document.addEventListener('reset', onEv, true);
      return;
    }

    const announcer = document.createElement('div');
    announcer.setAttribute('aria-live', 'polite');
    announcer.classList.add('visually-hidden');
    document.body.append(announcer);
    let announceTimer;
    const flagged = new WeakSet();
    const announced = new WeakSet();

    const syncAria = (el, shouldAnnounce = false) => {
      if (!el?.matches?.('input, select, textarea')) return;
      const invalid = el.matches(':user-invalid');
      if (invalid) {
        if (shouldAnnounce && !announced.has(el) && el.dataset?.error) {
          const errNode = document.getElementById(el.dataset.error);
          if (errNode) {
            clearTimeout(announceTimer);
            announcer.textContent = '';
            announceTimer = setTimeout(() => {
              announcer.textContent = errNode.textContent.replace(/\s+/g, ' ').trim();
            }, 500);
            announced.add(el);
          }
        }
        el.setAttribute('aria-invalid', 'true');
        linkError(el, true);
        flagged.add(el);
      } else if (flagged.has(el) || el.getAttribute('aria-invalid') !== 'true') {
        el.setAttribute('aria-invalid', 'false');
        linkError(el, false);
        flagged.delete(el);
        announced.delete(el);
      }
    };

    document.querySelectorAll('input, select, textarea').forEach((el) => syncAria(el));
    document.addEventListener('blur', (e) => syncAria(e.target, true), true);
    document.addEventListener('input', (e) => {
      if (e.target?.hasAttribute?.('aria-invalid')) syncAria(e.target);
    });
    document.addEventListener('change', (e) => syncAria(e.target, true), true);
    document.addEventListener('invalid', (e) => {
      announced.add(e.target);
      syncAria(e.target);
    }, true);
    document.addEventListener('reset', (e) => {
      setTimeout(() => [...(e.target.elements || [])].forEach((el) => syncAria(el)));
    }, true);
  }

  // ---------------------------------------------------------------------------
  // 13. UTILIDADES DE FORMATO Y SEGURIDAD XSS
  // ---------------------------------------------------------------------------
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
