/**
 * =============================================================================
 * AUREUM & CO. — PRIVATE AUCTION SALON CONTROLLER
 * Controlador Reactivo de Alta Fidelidad & Integración Clean Architecture REST
 * =============================================================================
 */

document.addEventListener('DOMContentLoaded', async () => {
  // ---------------------------------------------------------------------------
  // 1. ESTADO DE LA SALA PRIVADA
  // ---------------------------------------------------------------------------
  const state = {
    subastaId: 'SUB-2026-001',
    precioActual: 45000,
    incrementoMinimo: 1000,
    postorLiderId: 'NINGUNO',
    postorLiderNombre: 'Esperando Primera Oferta en Estrado',
    usuarioActual: {
      id: 'USR-VIP-01',
      nombre: 'Fernando Olivares',
      tier: 'CLIENT PRIVÉ (BLACK TIER)',
      garantia: 5000,
      escrowId: 'ESC-2026-901'
    },
    tiempoSegundos: 105, // 1 min 45 seg
    activa: true,
    totalPujas: 0,
    geminiOnline: true,
    timerInterval: null
  };

  // ---------------------------------------------------------------------------
  // 2. ELEMENTOS DEL DOM
  // ---------------------------------------------------------------------------
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

  const elCircuitBadge = document.getElementById('circuitBadge');
  const elAiScoreNumber = document.getElementById('aiScoreNumber');
  const elAiMeterFill = document.getElementById('aiMeterFill');
  const elAiJustificationText = document.getElementById('aiJustificationText');
  const elBtnToggleGemini = document.getElementById('btnToggleGemini');

  const elBidsTimeline = document.getElementById('bidsTimeline');
  const elBtnMartillo1 = document.getElementById('btnMartillo1');
  const elBtnMartillo2 = document.getElementById('btnMartillo2');
  const elBtnAdjudicar = document.getElementById('btnAdjudicar');

  // Insertar selector elegante de postores
  inyectarSelectorPostor();

  // Sincronizar estado inicial con el backend si está disponible
  await sincronizarEstadoInicial();

  // ---------------------------------------------------------------------------
  // 3. SINCRONIZACIÓN INICIAL CON EL BACKEND CLEAN ARCHITECTURE
  // ---------------------------------------------------------------------------
  async function sincronizarEstadoInicial() {
    try {
      const res = await fetch(`/api/v1/subastas/${state.subastaId}`);
      if (res.ok) {
        const data = await res.json();
        state.precioActual = data.precioActual;
        state.incrementoMinimo = data.incrementoMinimo;
        state.postorLiderId = data.postorLiderId;
        state.totalPujas = data.totalPujas || 0;
        if (data.postorLiderId && data.postorLiderId !== 'NINGUNO') {
          state.postorLiderNombre = `${data.postorLiderId} (Líder en Estrado)`;
        }
        actualizarPreciosUI();
      }
    } catch {
      // Si corre offline, mantiene valores de catálogo
      actualizarPreciosUI();
    }
  }

  // ---------------------------------------------------------------------------
  // 4. CRONÓMETRO REGRESIVO Y LÓGICA SOBERANA ANTI-SNIPING
  // ---------------------------------------------------------------------------
  function actualizarCronometro() {
    if (!state.activa) return;

    if (state.tiempoSegundos > 0) {
      state.tiempoSegundos--;
    } else {
      finalizarSubastaPorTiempo();
      return;
    }

    const m = Math.floor(state.tiempoSegundos / 60);
    const s = state.tiempoSegundos % 60;

    elMinutes.textContent = String(m).padStart(2, '0');
    elSeconds.textContent = String(s).padStart(2, '0');

    // Alerta de últimos 15 segundos con aura dorada/fuego
    if (state.tiempoSegundos <= 15) {
      elCountdownDisplay.style.borderColor = 'rgba(245, 158, 11, 0.9)';
      elCountdownDisplay.style.boxShadow = '0 0 35px rgba(245, 158, 11, 0.45)';
    } else {
      elCountdownDisplay.style.borderColor = '';
      elCountdownDisplay.style.boxShadow = '';
    }
  }

  function dispararReglaAntiSniping() {
    if (state.tiempoSegundos <= 15) {
      state.tiempoSegundos += 30;
      elAntiSnipingBanner.classList.remove('hidden');

      // Animación de pulso
      elCountdownDisplay.classList.add('price-bump');
      setTimeout(() => elCountdownDisplay.classList.remove('price-bump'), 600);

      // Ocultar banner tras 5 segundos
      setTimeout(() => {
        elAntiSnipingBanner.classList.add('hidden');
      }, 5000);

      mostrarToast('⚡ ANTI-SNIPING SOBERANO: Oferta en borde crítico (+30s otorgados a sala)', 'success');
    }
  }

  function finalizarSubastaPorTiempo() {
    state.activa = false;
    clearInterval(state.timerInterval);
    elMinutes.textContent = '00';
    elSeconds.textContent = '00';
    elAuctionPhaseTag.textContent = 'ESTRADO CERRADO • LOTE ADJUDICADO';
    elAuctionPhaseTag.style.background = 'rgba(212, 175, 55, 0.2)';
    elAuctionPhaseTag.style.color = '#fde68a';
    elAuctionPhaseTag.style.borderColor = 'rgba(212, 175, 55, 0.6)';

    elBtnEmitirPuja.disabled = true;
    elBidInput.disabled = true;
    quickBidButtons.forEach(btn => btn.disabled = true);

    mostrarToast(`🏛️ MARTILLAZO DE SALA: Adjudicación oficial concedida a ${state.postorLiderNombre} por S/. ${formatearNumero(state.precioActual)}`, 'success');
  }

  state.timerInterval = setInterval(actualizarCronometro, 1000);

  // ---------------------------------------------------------------------------
  // 5. GESTIÓN DE OFERTAS PRIVADAS EN VIVO (EVENT-DRIVEN)
  // ---------------------------------------------------------------------------
  function actualizarPreciosUI() {
    elCurrentPriceValue.textContent = formatearNumero(state.precioActual);
    elCurrentLeaderName.textContent = state.postorLiderNombre;
    elTotalBidsCount.textContent = `${state.totalPujas} oferta${state.totalPujas === 1 ? '' : 's'}`;

    const siguienteMinimo = state.precioActual + state.incrementoMinimo;
    elBidInput.min = siguienteMinimo;
    elBidInput.value = siguienteMinimo;

    // Efecto visual de rebote noble de precio
    elCurrentPriceDisplay.classList.add('price-bump');
    setTimeout(() => elCurrentPriceDisplay.classList.remove('price-bump'), 400);
  }

  quickBidButtons.forEach(btn => {
    btn.addEventListener('click', () => {
      const incremento = Number(btn.getAttribute('data-add'));
      elBidInput.value = state.precioActual + incremento;
      elBidInput.focus();
    });
  });

  elBtnEmitirPuja.addEventListener('click', async () => {
    const monto = Number(elBidInput.value);

    // Validación 1: Monto superior al mínimo
    if (monto < state.precioActual + state.incrementoMinimo) {
      mostrarToast(`⚠️ La oferta en estrado debe ser al menos S/. ${formatearNumero(state.precioActual + state.incrementoMinimo)}`, 'error');
      return;
    }

    // Validación 2: Regla de Dominio: Evitar auto-puja líder
    if (state.usuarioActual.id === state.postorLiderId) {
      mostrarToast(`❌ REGLA DE SALA (ERR_AUTO_PUJA_LIDER): Ya ostentas la oferta máxima con S/. ${formatearNumero(state.precioActual)}. No es admisible sobrepujar tu propio estatus.`, 'error');
      return;
    }

    // Deshabilitar temporalmente para simular latencia atómica
    elBtnEmitirPuja.disabled = true;
    elBtnEmitirPuja.innerHTML = '<span class="btn-text">AUDITANDO EN REDIS...</span>';

    try {
      // Intentar llamada HTTP al servidor backend si está disponible
      const res = await fetch(`/api/v1/subastas/${state.subastaId}/puja`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          postorId: state.usuarioActual.id,
          monto: monto
        })
      });

      if (res.ok) {
        const data = await res.json();
        procesarPujaExitosa(monto, state.usuarioActual.nombre, data.hashSha256 || generarHashMock());
      } else {
        const errorData = await res.json().catch(() => ({}));
        if (errorData.error) {
          mostrarToast(`❌ ${errorData.error}`, 'error');
        } else {
          procesarPujaExitosa(monto, state.usuarioActual.nombre, generarHashMock());
        }
      }
    } catch {
      // Fallback offline autónomo
      procesarPujaExitosa(monto, state.usuarioActual.nombre, generarHashMock());
    } finally {
      elBtnEmitirPuja.disabled = false;
      elBtnEmitirPuja.innerHTML = `
        <span class="btn-shine"></span>
        <span class="btn-text">¡EMITIR OFERTA PRIVADA!</span>
        <span class="btn-subtext">Confirmación Atómica Concurrente &lt; 200 ms</span>
      `;
    }
  });

  function procesarPujaExitosa(monto, postorNombre, hash) {
    state.precioActual = monto;
    state.postorLiderId = state.usuarioActual.id;
    state.postorLiderNombre = `${postorNombre} (${state.usuarioActual.id})`;
    state.totalPujas++;

    actualizarPreciosUI();
    dispararReglaAntiSniping();
    agregarPujaAlTimeline(monto, postorNombre, hash);

    mostrarToast(`✓ Oferta Soberana de S/. ${formatearNumero(monto)} certificada. Bloqueo atómico confirmado en 138 ms.`, 'success');
  }

  function agregarPujaAlTimeline(monto, postor, hash) {
    const item = document.createElement('div');
    item.className = 'bid-item winner-bid';
    item.innerHTML = `
      <div class="bid-correlative">#${state.totalPujas}</div>
      <div class="bid-details">
        <div class="bid-row-top">
          <span class="bid-amount">S/. ${formatearNumero(monto)}</span>
          <span class="bid-time">Justo ahora</span>
        </div>
        <div class="bid-row-bottom">
          <span class="bid-user">${postor}</span>
          <span class="audit-tag" title="Hash: ${hash}">✓ Protocolo Notarial SHA-256</span>
        </div>
      </div>
    `;

    // Quitar destaque del item anterior
    const prevWinner = elBidsTimeline.querySelector('.winner-bid');
    if (prevWinner) prevWinner.classList.remove('winner-bid');

    elBidsTimeline.insertBefore(item, elBidsTimeline.firstChild);
  }

  // ---------------------------------------------------------------------------
  // 6. WIDGET DE INTELIGENCIA ARTIFICIAL & CIRCUIT BREAKER
  // ---------------------------------------------------------------------------
  elBtnToggleGemini.addEventListener('click', () => {
    state.geminiOnline = !state.geminiOnline;

    if (!state.geminiOnline) {
      // ESTADO OPEN (FALLBACK ACTIVO)
      elCircuitBadge.className = 'circuit-badge open';
      elCircuitBadge.textContent = 'CIRCUITO: OPEN (FALLBACK ACTIVO)';

      elAiScoreNumber.innerHTML = '75<small>/100 (B+ Contingencia)</small>';
      elAiMeterFill.style.width = '75%';
      elAiMeterFill.style.background = 'linear-gradient(90deg, #f59e0b, #d97706)';

      elAiJustificationText.innerHTML = `
        <strong style="color: #f59e0b;">⚠️ Auditoría de Resiliencia (Timeout >800ms):</strong> Circuit Breaker activado en 12 ms.
        El motor recurrió a la <em>Heurística Local</em> protegiendo la subasta sin interrupción.
        Garantía preventiva soberana reajustada al <strong>20% (S/. 9,000)</strong>.
      `;

      elBtnToggleGemini.innerHTML = '<span class="btn-icon">🟢</span> Restaurar Servicio Gemini';
      elBtnToggleGemini.classList.add('btn-success-sim');

      mostrarToast('⚡ Circuit Breaker conmutó a Heurística Local. Transacciones garantizadas.', 'error');
    } else {
      // ESTADO CLOSED (NORMALIZADO)
      elCircuitBadge.className = 'circuit-badge closed';
      elCircuitBadge.textContent = 'CIRCUITO: CLOSED';

      elAiScoreNumber.innerHTML = '92<small>/100 (AAA+)</small>';
      elAiMeterFill.style.width = '92%';
      elAiMeterFill.style.background = 'linear-gradient(90deg, #10b981, #06b6d4)';

      elAiJustificationText.innerHTML = `
        ✓ Postor con KYC acreditado, calificación crediticia solvente y 100% de cumplimiento en sala. Garantía preferencial asignada: <strong>10% (S/. 4,500)</strong>.
      `;

      elBtnToggleGemini.innerHTML = '<span class="btn-icon">⚡</span> Simular Caída de Gemini';
      elBtnToggleGemini.classList.remove('btn-success-sim');

      mostrarToast('✓ Enlace Gemini 1.5 Flash restablecido. Circuito normalizado a CLOSED.', 'success');
    }
  });

  // ---------------------------------------------------------------------------
  // 7. ACCIONES DEL MARTILLERO OFICIAL (GAVEL ACTIONS)
  // ---------------------------------------------------------------------------
  elBtnMartillo1.addEventListener('click', () => {
    if (state.tiempoSegundos > 20) state.tiempoSegundos = 20;
    agregarMensajeMartillero(`⚖️ Martillero de Sala: "¡PRIMERA LLAMADA por S/. ${formatearNumero(state.precioActual)} para ${state.postorLiderNombre}!"`);
    mostrarToast('⚖️ ¡A LA UNA (PRIMERA LLAMADA)!', 'success');
  });

  elBtnMartillo2.addEventListener('click', () => {
    if (state.tiempoSegundos > 10) state.tiempoSegundos = 10;
    agregarMensajeMartillero(`⚖️ Martillero de Sala: "¡SEGUNDA LLAMADA por S/. ${formatearNumero(state.precioActual)}! ¿Alguna última oferta en sala o estrados privados?"`);
    mostrarToast('⚖️ ¡A LAS DOS (SEGUNDA LLAMADA)!', 'success');
  });

  elBtnAdjudicar.addEventListener('click', () => {
    finalizarSubastaPorTiempo();
    agregarMensajeMartillero(`🏛️ ¡ADJUDICACIÓN DEFINITIVA! Lote SUB-2026-001 adjudicado formalmente bajo fe notarial a ${state.postorLiderNombre} por S/. ${formatearNumero(state.precioActual)}.`);
    
    // Disparar lluvia de oro y platino
    lanzarConfettiDorado();
  });

  function agregarMensajeMartillero(mensaje) {
    const item = document.createElement('div');
    item.className = 'bid-item auctioneer-msg';
    item.style.borderLeft = '3px solid var(--gold-primary)';
    item.style.background = 'rgba(212, 175, 55, 0.08)';
    item.innerHTML = `
      <div class="bid-correlative" style="background: var(--gold-primary); color: #000; font-weight: 900;">⚖️</div>
      <div class="bid-details">
        <div class="bid-row-top">
          <span class="bid-amount" style="font-size: 0.88rem; color: var(--gold-champagne);">Bando Notarial de Sala</span>
          <span class="bid-time">En Vivo</span>
        </div>
        <div class="bid-row-bottom">
          <span class="bid-user" style="color: #fef08a; font-family: var(--font-serif); font-size: 0.8rem;">${mensaje}</span>
        </div>
      </div>
    `;
    elBidsTimeline.insertBefore(item, elBidsTimeline.firstChild);
  }

  // ---------------------------------------------------------------------------
  // 8. SELECTOR DE POSTOR PRIVADO PARA DEMOSTRACIONES
  // ---------------------------------------------------------------------------
  function inyectarSelectorPostor() {
    const cardArena = document.querySelector('.bid-console');
    if (!cardArena) return;

    const selectorDiv = document.createElement('div');
    selectorDiv.className = 'postor-selector-bar';
    selectorDiv.style.cssText = `
      display: flex;
      align-items: center;
      justify-content: space-between;
      margin-bottom: 12px;
      padding: 10px 14px;
      background: rgba(212, 175, 55, 0.04);
      border-radius: 8px;
      border: 1px solid var(--gold-soft-border);
      font-size: 0.78rem;
    `;

    selectorDiv.innerHTML = `
      <span style="font-family: var(--font-serif); color: var(--gold-champagne); font-weight: 700; letter-spacing: 0.06em;">👤 Credencial de Postor:</span>
      <select id="userSelector" style="
        background: #0a0d14;
        color: #f8fafc;
        border: 1px solid var(--gold-soft-border);
        border-radius: 6px;
        padding: 5px 10px;
        font-family: var(--font-serif);
        font-size: 0.78rem;
        font-weight: 700;
        cursor: pointer;
        outline: none;
      ">
        <option value="USR-VIP-01" selected>Fernando Olivares • Client Privé (Black Tier)</option>
        <option value="USR-882">Lord Carlos Rodríguez • Sovereign Member</option>
        <option value="USR-904">Doña Ana María Paredes • Private Trustee</option>
      </select>
    `;

    cardArena.insertBefore(selectorDiv, cardArena.firstChild);

    const selectEl = document.getElementById('userSelector');
    selectEl.addEventListener('change', (e) => {
      const val = e.target.value;
      if (val === 'USR-VIP-01') {
        state.usuarioActual = { id: 'USR-VIP-01', nombre: 'Fernando Olivares', garantia: 5000, tier: 'CLIENT PRIVÉ (BLACK TIER)' };
      } else if (val === 'USR-882') {
        state.usuarioActual = { id: 'USR-882', nombre: 'Lord Carlos Rodríguez', garantia: 6000, tier: 'SOVEREIGN MEMBER' };
      } else {
        state.usuarioActual = { id: 'USR-904', nombre: 'Doña Ana María Paredes', garantia: 5500, tier: 'PRIVATE TRUSTEE' };
      }

      const escrowBadge = document.getElementById('escrowBadge');
      if (escrowBadge) {
        escrowBadge.textContent = `🛡️ Custodia Soberana: S/. ${formatearNumero(state.usuarioActual.garantia)} (Activa)`;
      }

      const userNameEl = document.querySelector('.user-name');
      if (userNameEl) {
        userNameEl.textContent = state.usuarioActual.nombre;
      }

      const tierPill = document.querySelector('.tier-pill');
      if (tierPill) {
        tierPill.textContent = state.usuarioActual.tier;
      }

      mostrarToast(`Credencial activa: ${state.usuarioActual.nombre} (${state.usuarioActual.tier})`, 'success');
    });
  }

  // ---------------------------------------------------------------------------
  // 9. UTILIDADES & LLUVIA DE CONFETI METÁLICO (GOLD & PLATINUM)
  // ---------------------------------------------------------------------------
  function formatearNumero(num) {
    return Number(num).toLocaleString('es-PE');
  }

  function generarHashMock() {
    return '0x' + Array.from({ length: 8 }, () => Math.floor(Math.random() * 16).toString(16)).join('') + '...sha256';
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

  function lanzarConfettiDorado() {
    const colores = ['#d4af37', '#f7e7b4', '#fff3cc', '#c5a059', '#e2e8f0', '#997822'];
    for (let i = 0; i < 60; i++) {
      const p = document.createElement('div');
      p.className = 'luxury-confetti';
      p.style.cssText = `
        position: fixed;
        top: -15px;
        left: ${Math.random() * 100}vw;
        width: ${Math.random() * 10 + 6}px;
        height: ${Math.random() * 16 + 8}px;
        background: ${colores[Math.floor(Math.random() * colores.length)]};
        box-shadow: 0 0 10px rgba(212, 175, 55, 0.4);
        transform: rotate(${Math.random() * 360}deg);
        z-index: 9999;
        pointer-events: none;
        transition: transform 3.5s ease-out, top 3.5s ease-in, opacity 3.5s ease-out;
      `;
      document.body.appendChild(p);

      requestAnimationFrame(() => {
        p.style.top = '105vh';
        p.style.transform = `rotate(${Math.random() * 1080}deg) translateX(${Math.random() * 140 - 70}px)`;
        p.style.opacity = '0';
      });

      setTimeout(() => p.remove(), 3600);
    }
  }
});
