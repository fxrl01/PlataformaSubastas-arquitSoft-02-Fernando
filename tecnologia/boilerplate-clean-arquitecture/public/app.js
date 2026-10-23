/**
 * =============================================================================
 * LIVEBID — CLIENT INTERFACE & REAL-TIME INTERACTION CONTROLLER
 * Arquitectura: Event-Driven UI con Mockup Dinámico y Conexión REST
 * =============================================================================
 */

document.addEventListener('DOMContentLoaded', () => {
  // ---------------------------------------------------------------------------
  // 1. ESTADO DE LA SALA DE SUBASTAS
  // ---------------------------------------------------------------------------
  const state = {
    subastaId: 'SUB-2026-001',
    precioActual: 46000,
    incrementoMinimo: 1000,
    postorLiderId: 'USR-VIP-01',
    postorLiderNombre: 'Fernando O. (USR-VIP-01)',
    usuarioActual: {
      id: 'USR-VIP-01',
      nombre: 'Fernando Olivares (VIP)',
      garantia: 5000,
      escrowId: 'ESC-2026-901'
    },
    tiempoSegundos: 105, // 1 min 45 seg
    activa: true,
    totalPujas: 1,
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

  // Insertar selector rápido de postores para probar reglas de auto-puja
  inyectarSelectorPostor();

  // ---------------------------------------------------------------------------
  // 3. CRONÓMETRO REGRESIVO Y LÓGICA ANTI-SNIPING
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

    // Alerta visual de últimos 15 segundos
    if (state.tiempoSegundos <= 15) {
      elCountdownDisplay.style.borderColor = 'rgba(239, 68, 68, 0.8)';
      elCountdownDisplay.style.boxShadow = '0 0 25px rgba(239, 68, 68, 0.4)';
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
      elCountdownDisplay.classList.add('extended-pulse');
      setTimeout(() => elCountdownDisplay.classList.remove('extended-pulse'), 1000);

      // Ocultar banner tras 5 segundos
      setTimeout(() => {
        elAntiSnipingBanner.classList.add('hidden');
      }, 5000);

      mostrarToast('⚡ ANTI-SNIPING: Puja de último segundo detectada (+30 segundos añadidos)', 'success');
    }
  }

  function finalizarSubastaPorTiempo() {
    state.activa = false;
    clearInterval(state.timerInterval);
    elMinutes.textContent = '00';
    elSeconds.textContent = '00';
    elAuctionPhaseTag.textContent = 'SUBASTA CERRADA - ADJUDICADA';
    elAuctionPhaseTag.style.background = 'rgba(239, 68, 68, 0.2)';
    elAuctionPhaseTag.style.color = '#ef4444';

    elBtnEmitirPuja.disabled = true;
    elBidInput.disabled = true;
    quickBidButtons.forEach(btn => btn.disabled = true);

    mostrarToast(`🏆 TIEMPO CONCLUIDO: Adjudicación oficial para ${state.postorLiderNombre} por S/. ${formatearNumero(state.precioActual)}`, 'success');
  }

  state.timerInterval = setInterval(actualizarCronometro, 1000);

  // ---------------------------------------------------------------------------
  // 4. GESTIÓN DE PUJAS EN VIVO (EVENT-DRIVEN)
  // ---------------------------------------------------------------------------
  function actualizarPreciosUI() {
    elCurrentPriceValue.textContent = formatearNumero(state.precioActual);
    elCurrentLeaderName.textContent = state.postorLiderNombre;
    elTotalBidsCount.textContent = `${state.totalPujas} puja${state.totalPujas > 1 ? 's' : ''}`;

    const siguienteMinimo = state.precioActual + state.incrementoMinimo;
    elBidInput.min = siguienteMinimo;
    elBidInput.value = siguienteMinimo;

    // Efecto visual de rebote de precio
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
      mostrarToast(`⚠️ El monto debe ser al menos S/. ${formatearNumero(state.precioActual + state.incrementoMinimo)}`, 'error');
      return;
    }

    // Validación 2: Regla de Dominio: Evitar auto-puja líder
    if (state.usuarioActual.id === state.postorLiderId) {
      mostrarToast(`❌ REGLA DE DOMINIO (ERR_AUTO_PUJA_LIDER): Ya eres el postor líder con S/. ${formatearNumero(state.precioActual)}. No puedes sobrepujar tu propia oferta.`, 'error');
      return;
    }

    // Deshabilitar temporalmente para simular latencia atómica
    elBtnEmitirPuja.disabled = true;
    elBtnEmitirPuja.innerHTML = '<span class="btn-text">VALIDANDO EN REDIS...</span>';

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
        // Si el backend responde con error de dominio, mostrarlo
        if (errorData.error) {
          mostrarToast(`❌ ${errorData.error}`, 'error');
        } else {
          // Fallback a simulación cliente
          procesarPujaExitosa(monto, state.usuarioActual.nombre, generarHashMock());
        }
      }
    } catch (netErr) {
      // Fallback offline autónomo
      procesarPujaExitosa(monto, state.usuarioActual.nombre, generarHashMock());
    } finally {
      elBtnEmitirPuja.disabled = false;
      elBtnEmitirPuja.innerHTML = `
        <span class="btn-text">¡OFERTAR AHORA!</span>
        <span class="btn-subtext">Confirmación Atómica &lt; 200 ms</span>
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

    mostrarToast(`✓ ¡Oferta de S/. ${formatearNumero(monto)} aceptada! Bloqueo atómico confirmado en 142 ms.`, 'success');
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
          <span class="audit-tag" title="Hash: ${hash}">✓ Hash SHA-256</span>
        </div>
      </div>
    `;

    // Quitar destaque del item anterior
    const prevWinner = elBidsTimeline.querySelector('.winner-bid');
    if (prevWinner) prevWinner.classList.remove('winner-bid');

    elBidsTimeline.insertBefore(item, elBidsTimeline.firstChild);
  }

  // ---------------------------------------------------------------------------
  // 5. WIDGET DE INTELIGENCIA ARTIFICIAL & CIRCUIT BREAKER
  // ---------------------------------------------------------------------------
  elBtnToggleGemini.addEventListener('click', () => {
    state.geminiOnline = !state.geminiOnline;

    if (!state.geminiOnline) {
      // ESTADO OPEN (FALLBACK ACTIVO)
      elCircuitBadge.className = 'circuit-badge open';
      elCircuitBadge.textContent = 'CIRCUITO: OPEN (FALLBACK ACTIVO)';

      elAiScoreNumber.innerHTML = '75<small>/100</small>';
      elAiMeterFill.style.width = '75%';
      elAiMeterFill.style.background = 'linear-gradient(90deg, #f59e0b, #d97706)';

      elAiJustificationText.innerHTML = `
        <strong style="color: #f59e0b;">⚠️ Timeout en Gemini AI (>800ms):</strong> Circuit Breaker activado.
        El sistema recurrió a la <em>Heurística Local</em> sin interrumpir la subasta.
        Garantía preventiva ajustada al <strong>20%</strong> (S/. 9,000).
      `;

      elBtnToggleGemini.textContent = '🟢 Restaurar Servicio Gemini';
      elBtnToggleGemini.classList.add('btn-success-sim');

      mostrarToast('⚡ Circuit Breaker abrió el circuito: Conmutado a Heurística Local en 12 ms.', 'error');
    } else {
      // ESTADO CLOSED (NORMALIZADO)
      elCircuitBadge.className = 'circuit-badge closed';
      elCircuitBadge.textContent = 'CIRCUITO: CLOSED';

      elAiScoreNumber.innerHTML = '92<small>/100</small>';
      elAiMeterFill.style.width = '92%';
      elAiMeterFill.style.background = 'linear-gradient(90deg, #10b981, #06b6d4)';

      elAiJustificationText.innerHTML = `
        ✓ Postor con KYC verificado, historial crediticio solvente y tasa de cumplimiento del 100%. Garantía estándar: 10% (S/. 4,500).
      `;

      elBtnToggleGemini.textContent = 'Simular Caída de Gemini';
      elBtnToggleGemini.classList.remove('btn-success-sim');

      mostrarToast('✓ Servicio Gemini 1.5 Flash restaurado. Circuito normalizado a CLOSED.', 'success');
    }
  });

  // ---------------------------------------------------------------------------
  // 6. ACCIONES DEL MARTILLERO OFICIAL (MODERACIÓN EN VIVO)
  // ---------------------------------------------------------------------------
  elBtnMartillo1.addEventListener('click', () => {
    if (state.tiempoSegundos > 20) state.tiempoSegundos = 20;
    agregarMensajeMartillero(`🔨 Martillero Oficial: "¡A la una por S/. ${formatearNumero(state.precioActual)} para ${state.postorLiderNombre}!"`);
    mostrarToast('🔨 ¡A LA UNA!', 'success');
  });

  elBtnMartillo2.addEventListener('click', () => {
    if (state.tiempoSegundos > 10) state.tiempoSegundos = 10;
    agregarMensajeMartillero(`🔨 Martillero Oficial: "¡A las dos por S/. ${formatearNumero(state.precioActual)}! ¿Alguna otra oferta en sala?"`);
    mostrarToast('🔨 ¡A LAS DOS!', 'success');
  });

  elBtnAdjudicar.addEventListener('click', () => {
    finalizarSubastaPorTiempo();
    agregarMensajeMartillero(`🎉 ¡MARTILLAZO FINAL! Lote SUB-2026-001 adjudicado formalmente a ${state.postorLiderNombre} por S/. ${formatearNumero(state.precioActual)}.`);
    
    // Disparar celebración visual
    lanzarConfetti();
  });

  function agregarMensajeMartillero(mensaje) {
    const item = document.createElement('div');
    item.className = 'bid-item auctioneer-msg';
    item.style.borderLeft = '3px solid #f59e0b';
    item.style.background = 'rgba(245, 158, 11, 0.08)';
    item.innerHTML = `
      <div class="bid-correlative" style="background:#f59e0b; color:#000;">🔨</div>
      <div class="bid-details">
        <div class="bid-row-top">
          <span class="bid-amount" style="font-size: 0.95rem; color:#f59e0b;">Anuncio de Sala</span>
          <span class="bid-time">En vivo</span>
        </div>
        <div class="bid-row-bottom">
          <span class="bid-user" style="color: #fde68a;">${mensaje}</span>
        </div>
      </div>
    `;
    elBidsTimeline.insertBefore(item, elBidsTimeline.firstChild);
  }

  // ---------------------------------------------------------------------------
  // 7. SELECTOR DE POSTOR PARA DEMOSTRACIONES
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
      padding: 8px 12px;
      background: rgba(255, 255, 255, 0.03);
      border-radius: 8px;
      border: 1px solid rgba(255, 255, 255, 0.06);
      font-size: 0.8rem;
    `;

    selectorDiv.innerHTML = `
      <span style="color: var(--text-secondary);">👤 Simular Usuario:</span>
      <select id="userSelector" style="
        background: #0e1526;
        color: #f8fafc;
        border: 1px solid rgba(59, 130, 246, 0.3);
        border-radius: 6px;
        padding: 4px 8px;
        font-size: 0.8rem;
        cursor: pointer;
        outline: none;
      ">
        <option value="USR-VIP-01" selected>Fernando Olivares (VIP Líder)</option>
        <option value="USR-882">Carlos Rodríguez (Postor 2)</option>
        <option value="USR-904">Ana María Paredes (Postor 3)</option>
      </select>
    `;

    cardArena.insertBefore(selectorDiv, cardArena.firstChild);

    const selectEl = document.getElementById('userSelector');
    selectEl.addEventListener('change', (e) => {
      const val = e.target.value;
      if (val === 'USR-VIP-01') {
        state.usuarioActual = { id: 'USR-VIP-01', nombre: 'Fernando Olivares (VIP)', garantia: 5000 };
      } else if (val === 'USR-882') {
        state.usuarioActual = { id: 'USR-882', nombre: 'Carlos Rodríguez', garantia: 6000 };
      } else {
        state.usuarioActual = { id: 'USR-904', nombre: 'Ana María Paredes', garantia: 5500 };
      }

      const escrowBadge = document.getElementById('escrowBadge');
      if (escrowBadge) {
        escrowBadge.textContent = `🛡️ Custodia: S/. ${formatearNumero(state.usuarioActual.garantia)} (Activa)`;
      }

      const userNameEl = document.querySelector('.user-name');
      if (userNameEl) {
        userNameEl.textContent = state.usuarioActual.nombre;
      }

      mostrarToast(`Cambiado a perfil: ${state.usuarioActual.nombre}`, 'success');
    });
  }

  // ---------------------------------------------------------------------------
  // 8. UTILIDADES
  // ---------------------------------------------------------------------------
  function formatearNumero(num) {
    return Number(num).toLocaleString('es-PE');
  }

  function generarHashMock() {
    return Array.from({ length: 8 }, () => Math.floor(Math.random() * 16).toString(16)).join('') + '...sha256';
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

  function lanzarConfetti() {
    const colores = ['#10b981', '#06b6d4', '#f59e0b', '#ec4899', '#3b82f6'];
    for (let i = 0; i < 40; i++) {
      const p = document.createElement('div');
      p.className = 'confetti-particle';
      p.style.cssText = `
        position: fixed;
        top: -10px;
        left: ${Math.random() * 100}vw;
        width: ${Math.random() * 8 + 6}px;
        height: ${Math.random() * 14 + 8}px;
        background: ${colores[Math.floor(Math.random() * colores.length)]};
        transform: rotate(${Math.random() * 360}deg);
        z-index: 9999;
        pointer-events: none;
        transition: transform 3s ease-out, top 3s ease-in, opacity 3s ease-out;
      `;
      document.body.appendChild(p);

      requestAnimationFrame(() => {
        p.style.top = '105vh';
        p.style.transform = `rotate(${Math.random() * 720}deg) translateX(${Math.random() * 100 - 50}px)`;
        p.style.opacity = '0';
      });

      setTimeout(() => p.remove(), 3200);
    }
  }
});
