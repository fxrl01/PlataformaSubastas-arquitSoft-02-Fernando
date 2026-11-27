import http from 'k6/http';
import { check, sleep } from 'k6';
import { Counter, Trend } from 'k6/metrics';

/**
 * PRUEBA DE ESTRÉS POR PICO SÚBITO (SPIKE / STAMPEDE ANTI-SNIPING)
 * A diferencia de tests/load/ (rampa gradual), esta suite simula una avalancha
 * repentina (0 -> 150 VUs en 4 segundos) sobre un lote en cierre para validar:
 *   1. Mitigación de Cache Stampede (singleflight + Redis Cache-Aside).
 *   2. Bloqueo distribuido atómico Redis en ráfagas simultáneas de ofertas (cero race conditions).
 *   3. Cero respuestas HTTP 5xx bajo presión abrupta.
 */

const BASE_URL = __ENV.BASE_URL || 'http://localhost:3000';
const spike5xxErrors = new Counter('spike_http_5xx_errors');
const acceptedBids = new Counter('spike_accepted_bids');
const rejectedLowerBids = new Counter('spike_expected_conflict_bids');
const spikeBidLatency = new Trend('spike_bid_latency_ms');

export const options = {
  scenarios: {
    sudden_read_stampede: {
      executor: 'ramping-vus',
      exec: 'stampedeReadBurst',
      startVUs: 0,
      stages: [
        { duration: '4s', target: 120 },
        { duration: '8s', target: 120 },
        { duration: '3s', target: 0 },
      ],
      gracefulRampDown: '2s',
    },
    antisniping_bid_burst: {
      executor: 'ramping-vus',
      exec: 'contendingBidBurst',
      startVUs: 0,
      stages: [
        { duration: '3s', target: 30 },
        { duration: '9s', target: 30 },
        { duration: '2s', target: 0 },
      ],
      gracefulRampDown: '2s',
    },
  },
  thresholds: {
    spike_http_5xx_errors: ['count==0'],
    'http_req_duration{scenario:sudden_read_stampede}': ['p(95)<400'],
  },
};

function login(email, password, ipSuffix) {
  const res = http.post(
    `${BASE_URL}/api/v1/auth/login`,
    JSON.stringify({ email, password }),
    {
      headers: {
        'Content-Type': 'application/json',
        'CF-Connecting-IP': `198.18.0.${ipSuffix}`,
      },
    }
  );
  if (res.status !== 200) {
    throw new Error(`Login fallo (${res.status}): ${res.body}`);
  }
  return res.json();
}

export function setup() {
  const admin = login('admin@livebid.pe', 'AdminLiveBid2026!', 1);

  const now = new Date();
  const closeAt = new Date(now.getTime() + 18 * 1000); // Ventana corta para disparar Anti-Sniping (+30s)
  const createRes = http.post(
    `${BASE_URL}/api/v1/subastas`,
    JSON.stringify({
      titulo: `Lote Spike Stampede ${Date.now()}`,
      descripcion: 'Prueba de estrés por pico súbito en ventana Anti-Sniping',
      precio_base: 2000,
      inicio: now.toISOString(),
      cierre: closeAt.toISOString(),
    }),
    {
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${admin.access_token}`,
      },
    }
  );
  if (createRes.status !== 201) {
    throw new Error(`No se pudo crear subasta spike: ${createRes.status} ${createRes.body}`);
  }
  const auction = createRes.json();

  http.post(`${BASE_URL}/api/v1/subastas/${auction.id}/activar`, null, {
    headers: { Authorization: `Bearer ${admin.access_token}` },
  });

  const bidders = [];
  for (let i = 1; i <= 3; i++) {
    const email = `spike.bidder${i}.${Date.now()}@livebid.pe`;
    const pass = 'SpikeTest2026!';
    const reg = http.post(
      `${BASE_URL}/api/v1/auth/register`,
      JSON.stringify({
        name: `Postor Spike ${i}`,
        email,
        password: pass,
        rol_solicitado: 'POSTOR',
      }),
      {
        headers: {
          'Content-Type': 'application/json',
          'CF-Connecting-IP': `198.18.1.${i}`,
        },
      }
    );
    const u = reg.json();
    http.patch(
      `${BASE_URL}/api/v1/admin/users/${u.id}/status`,
      JSON.stringify({ estado: 'ACTIVO', kyc_aprobado: true }),
      {
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${admin.access_token}`,
        },
      }
    );
    const auth = login(email, pass, 10 + i);
    const intent = http
      .post(
        `${BASE_URL}/api/v1/subastas/${auction.id}/garantias/stripe-intent`,
        JSON.stringify({ monto: 400 }),
        {
          headers: {
            'Content-Type': 'application/json',
            Authorization: `Bearer ${auth.access_token}`,
          },
        }
      )
      .json();
    http.post(
      `${BASE_URL}/api/v1/subastas/${auction.id}/garantias`,
      JSON.stringify({
        monto: intent.monto_retencion || 400,
        payment_intent_id: intent.payment_intent_id,
      }),
      {
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${auth.access_token}`,
        },
      }
    );
    bidders.push({ id: u.id, token: auth.access_token });
  }

  return { auctionId: auction.id, bidders };
}

export function stampedeReadBurst(data) {
  const detail = http.get(`${BASE_URL}/api/v1/subastas/${data.auctionId}`);
  if (detail.status >= 500) spike5xxErrors.add(1);
  check(detail, {
    'Spike GET detalle 200': (r) => r.status === 200,
  });
  sleep(0.08);
}

export function contendingBidBurst(data) {
  const bidder = data.bidders[__VU % data.bidders.length];
  const curRes = http.get(`${BASE_URL}/api/v1/subastas/${data.auctionId}`);
  if (curRes.status !== 200) {
    if (curRes.status >= 500) spike5xxErrors.add(1);
    return;
  }
  const current = curRes.json().precio_actual || 2000;
  const candidateAmount = current + 100 + (__ITER % 5) * 50;

  const t0 = Date.now();
  const bidRes = http.post(
    `${BASE_URL}/api/v1/subastas/${data.auctionId}/pujas`,
    JSON.stringify({ monto: candidateAmount }),
    {
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${bidder.token}`,
        'Idempotency-Key': `spike-${data.auctionId}-${__VU}-${__ITER}-${Date.now()}`,
      },
    }
  );
  spikeBidLatency.add(Date.now() - t0);

  if (bidRes.status >= 500) {
    spike5xxErrors.add(1);
  } else if (bidRes.status === 201 || bidRes.status === 200) {
    acceptedBids.add(1);
  } else if (bidRes.status === 400 || bidRes.status === 409 || bidRes.status === 422) {
    // Rechazo atómico esperado cuando otro postor ganó el lock de Redis con un monto mayor
    rejectedLowerBids.add(1);
  }

  check(bidRes, {
    'Spike puja sin error 5xx': (r) => r.status < 500,
  });
  sleep(0.12);
}

export function teardown(data) {
  const finalAuction = http.get(`${BASE_URL}/api/v1/subastas/${data.auctionId}`).json();
  const history = http.get(`${BASE_URL}/api/v1/subastas/${data.auctionId}/pujas`).json();
  check(finalAuction, {
    'Consistencia final: precio_actual subió sin corrupción': (a) => a.precio_actual > 2000,
  });
  if (Array.isArray(history) && history.length > 1) {
    let maxBid = 0;
    for (const b of history) {
      if (b.monto > maxBid) maxBid = b.monto;
    }
    check(finalAuction, {
      'Integridad exacta: precio_actual coincide con puja máxima registrada': (a) => a.precio_actual === maxBid,
    });
  }
}
