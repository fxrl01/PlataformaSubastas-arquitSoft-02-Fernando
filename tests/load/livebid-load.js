import http from 'k6/http';
import ws from 'k6/ws';
import { check, sleep } from 'k6';
import { Counter, Trend } from 'k6/metrics';

const BASE_URL = __ENV.BASE_URL || 'http://localhost:3000';
const WS_BASE_URL = BASE_URL.replace(/^http/, 'ws');
const PROFILE = (__ENV.PROFILE || 'ci').toLowerCase();

const bidLatency = new Trend('livebid_bid_duration_ms');
const serverErrors = new Counter('livebid_http_5xx_errors');
const wsConnectionsOk = new Counter('livebid_ws_connected');

// Perfiles de escala seleccionables con -e PROFILE=ci | scale2000 | stress5000
const PROFILES = {
  ci: {
    label: 'CI Baseline (155 VUs)',
    readStages: [
      { duration: '5s', target: 50 },
      { duration: '15s', target: 100 },
      { duration: '5s', target: 0 },
    ],
    bidVUs: 15,
    bidDuration: '15s',
    readSleepSec: 0.1,
    bidSleepSec: 0.15,
    wsStages: [
      { duration: '5s', target: 40 },
      { duration: '15s', target: 40 },
      { duration: '3s', target: 0 },
    ],
    wsHoldMs: 3000,
  },
  scale2000: {
    label: 'Escala Produccion Azure (2,000 VUs: 1,650 WS + 325 Catalogo + 25 Pujadores)',
    readStages: [
      { duration: '15s', target: 150 },
      { duration: '20s', target: 325 },
      { duration: '25s', target: 325 },
      { duration: '10s', target: 0 },
    ],
    bidVUs: 25,
    bidDuration: '50s',
    readSleepSec: 1.0,
    bidSleepSec: 0.6,
    wsStages: [
      { duration: '20s', target: 850 },
      { duration: '20s', target: 1650 },
      { duration: '20s', target: 1650 },
      { duration: '10s', target: 0 },
    ],
    wsHoldMs: 42000,
  },
  stress5000: {
    label: 'Stress Futuro Crecimiento (5,000 VUs: 3,800 WS + 1,100 Catalogo + 100 Pujadores)',
    readStages: [
      { duration: '20s', target: 500 },
      { duration: '40s', target: 1100 },
      { duration: '40s', target: 1100 },
      { duration: '15s', target: 0 },
    ],
    bidVUs: 100,
    bidDuration: '80s',
    readSleepSec: 0.8,
    bidSleepSec: 0.6,
    wsStages: [
      { duration: '30s', target: 1800 },
      { duration: '40s', target: 3800 },
      { duration: '30s', target: 3800 },
      { duration: '15s', target: 0 },
    ],
    wsHoldMs: 15000,
  },
};

const activeCfg = PROFILES[PROFILE] || PROFILES.ci;

export const options = {
  scenarios: {
    catalog_read_load: {
      executor: 'ramping-vus',
      exec: 'catalogReadFlow',
      startVUs: 10,
      stages: activeCfg.readStages,
      gracefulRampDown: '3s',
    },
    bidding_spike: {
      executor: 'constant-vus',
      exec: 'concurrentBidFlow',
      vus: Number(__ENV.BID_VUS) || activeCfg.bidVUs,
      duration: activeCfg.bidDuration,
      startTime: '5s',
    },
    ws_spectators: {
      executor: 'ramping-vus',
      exec: 'websocketSpectatorFlow',
      startVUs: 10,
      stages: activeCfg.wsStages,
      startTime: '3s',
      gracefulRampDown: '3s',
    },
  },
  thresholds: {
    'http_req_duration{scenario:catalog_read_load}': ['p(95)<150'],
    'http_req_failed{scenario:catalog_read_load}': ['rate<0.01'],
    'livebid_bid_duration_ms': ['p(95)<200'],
    'livebid_http_5xx_errors': ['count==0'],
    'checks': ['rate>0.95'],
  },
};

function loginUser(email, password) {
  const res = http.post(
    `${BASE_URL}/api/v1/auth/login`,
    JSON.stringify({ email, password }),
    { headers: { 'Content-Type': 'application/json' } }
  );
  if (res.status !== 200) {
    throw new Error(`Login failed for ${email}: ${res.status} ${res.body}`);
  }
  return res.json();
}

export function setup() {
  const adminEmail = __ENV.ADMIN_EMAIL || 'admin@livebid.pe';
  const adminPass = __ENV.ADMIN_PASSWORD || 'AdminLiveBid2026!';
  const adminAuth = loginUser(adminEmail, adminPass);

  // Crear y activar una subasta dedicada para las pruebas de carga
  const now = new Date();
  const later = new Date(now.getTime() + 60 * 60 * 1000);
  const createRes = http.post(
    `${BASE_URL}/api/v1/subastas`,
    JSON.stringify({
      titulo: `Lote k6 Load Test ${Date.now()}`,
      descripcion: 'Subasta dedicada para benchmark k6 sobre Nginx + 2 réplicas Go',
      precio_base: 1000,
      inicio: now.toISOString(),
      cierre: later.toISOString(),
    }),
    {
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${adminAuth.access_token}`,
      },
    }
  );
  if (createRes.status !== 201) {
    throw new Error(`Create auction failed: ${createRes.status} ${createRes.body}`);
  }
  const auction = createRes.json();

  http.post(`${BASE_URL}/api/v1/subastas/${auction.id}/activar`, null, {
    headers: { Authorization: `Bearer ${adminAuth.access_token}` },
  });

  // Crear 2 postores dedicados, aprobarlos vía Admin y constituir garantías Stripe
  const bidders = [];
  for (let i = 1; i <= 2; i++) {
    const email = `k6.bidder${i}.${Date.now()}@livebid.pe`;
    const pass = 'K6LoadPass2026!';
    const reg = http.post(
      `${BASE_URL}/api/v1/auth/register`,
      JSON.stringify({
        name: `Postor K6 ${i}`,
        email,
        password: pass,
        rol_solicitado: 'POSTOR',
      }),
      { headers: { 'Content-Type': 'application/json' } }
    );
    const user = reg.json();
    http.patch(
      `${BASE_URL}/api/v1/admin/users/${user.id}/status`,
      JSON.stringify({ estado: 'ACTIVO', kyc_aprobado: true }),
      {
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${adminAuth.access_token}`,
        },
      }
    );
    const auth = loginUser(email, pass);
    const intentRes = http.post(
      `${BASE_URL}/api/v1/subastas/${auction.id}/garantias/stripe-intent`,
      JSON.stringify({ monto: 200 }),
      {
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${auth.access_token}`,
        },
      }
    );
    const intent = intentRes.json();
    http.post(
      `${BASE_URL}/api/v1/subastas/${auction.id}/garantias`,
      JSON.stringify({
        monto: intent.monto_retencion || 200,
        payment_intent_id: intent.payment_intent_id,
      }),
      {
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${auth.access_token}`,
        },
      }
    );
    bidders.push({ id: user.id, token: auth.access_token });
  }

  return {
    auctionId: auction.id,
    adminToken: adminAuth.access_token,
    bidders,
  };
}

export function catalogReadFlow(data) {
  const listRes = http.get(`${BASE_URL}/api/v1/subastas`);
  if (listRes.status >= 500) serverErrors.add(1);
  check(listRes, {
    'GET /subastas status 200': (r) => r.status === 200,
  });

  const detailRes = http.get(`${BASE_URL}/api/v1/subastas/${data.auctionId}`);
  if (detailRes.status >= 500) serverErrors.add(1);
  check(detailRes, {
    'GET /subastas/{id} status 200': (r) => r.status === 200,
  });

  const healthRes = http.get(`${BASE_URL}/health`);
  if (healthRes.status >= 500) serverErrors.add(1);
  check(healthRes, {
    'GET /health UP': (r) => r.status === 200,
  });

  sleep(activeCfg.readSleepSec);
}

export function concurrentBidFlow(data) {
  const bidder = data.bidders[(__VU + __ITER) % data.bidders.length];
  const currentRes = http.get(`${BASE_URL}/api/v1/subastas/${data.auctionId}`);
  let nextAmount = 1500 + __VU * 100 + __ITER * 50;
  if (currentRes.status === 200) {
    const sub = currentRes.json();
    const minInc = sub.precio_actual < 1000 ? 50 : sub.precio_actual < 10000 ? 100 : 500;
    nextAmount = Number(sub.precio_actual) + minInc;
  }

  const res = http.post(
    `${BASE_URL}/api/v1/subastas/${data.auctionId}/pujas`,
    JSON.stringify({ monto: nextAmount }),
    {
      headers: {
        'Content-Type': 'application/json',
        Authorization: `Bearer ${bidder.token}`,
        'Idempotency-Key': `k6-${__VU}-${__ITER}-${Date.now()}`,
      },
    }
  );
  bidLatency.add(res.timings.duration);
  if (res.status >= 500) {
    serverErrors.add(1);
  }
  // En ráfaga concurrente real, 201 (ganó lock) o 409 (contención/auto-puja protegida) son respuestas válidas sin 5xx
  check(res, {
    'Bid handled atomically (201 or 409, zero 5xx)': (r) => r.status === 201 || r.status === 409,
  });
  sleep(activeCfg.bidSleepSec);
}

export function websocketSpectatorFlow(data) {
  const url = `${WS_BASE_URL}/ws/subastas/${data.auctionId}?token=${encodeURIComponent(data.adminToken)}`;
  const res = ws.connect(url, {}, function (socket) {
    socket.on('open', () => {
      wsConnectionsOk.add(1);
    });
    socket.setTimeout(() => {
      socket.close();
    }, activeCfg.wsHoldMs);
  });
  check(res, {
    'WebSocket upgraded 101': (r) => r && r.status === 101,
  });
  sleep(0.5);
}

export function teardown(data) {
  http.post(`${BASE_URL}/api/v1/subastas/${data.auctionId}/cerrar`, null, {
    headers: { Authorization: `Bearer ${data.adminToken}` },
  });
}

function metricVal(data, name, key) {
  const m = data.metrics[name];
  if (!m || !m.values || m.values[key] === undefined) return 'N/A';
  const v = m.values[key];
  return typeof v === 'number' ? v.toFixed(2) : String(v);
}

export function handleSummary(data) {
  const checksRate = Number(metricVal(data, 'checks', 'rate'));
  const checksPct = Number.isNaN(checksRate) ? 'N/A' : `${(checksRate * 100).toFixed(2)}%`;

  const html = `<!DOCTYPE html>
<html lang="es">
<head>
  <meta charset="UTF-8" />
  <title>LiveBid — Reporte de Pruebas de Carga k6</title>
  <style>
    body { font-family: system-ui, -apple-system, sans-serif; background: #0b1120; color: #e2e8f0; margin: 0; padding: 32px; }
    .wrap { max-width: 960px; margin: 0 auto; }
    h1 { color: #fbbf24; margin-bottom: 6px; }
    .sub { color: #94a3b8; margin-bottom: 24px; }
    .grid { display: grid; grid-template-columns: repeat(auto-fit, minmax(210px, 1fr)); gap: 16px; margin-bottom: 24px; }
    .card { background: #111827; border: 1px solid #1e293b; border-radius: 12px; padding: 16px; }
    .label { font-size: 12px; color: #94a3b8; text-transform: uppercase; letter-spacing: .05em; }
    .val { font-size: 28px; font-weight: 700; color: #38bdf8; margin-top: 6px; }
    .ok { color: #34d399; }
    table { width: 100%; border-collapse: collapse; background: #111827; border-radius: 12px; overflow: hidden; }
    th, td { padding: 12px 16px; border-bottom: 1px solid #1e293b; text-align: left; }
    th { background: #1e293b; color: #fbbf24; font-size: 13px; }
  </style>
</head>
<body>
  <div class="wrap">
    <h1>LiveBid — Reporte Oficial de Pruebas de Carga (k6)</h1>
    <p class="sub">Perfil: <strong>${activeCfg.label}</strong> &bull; Entorno: Cloudflare Edge / Nginx LB &rarr; 2 Réplicas Go &rarr; PgBouncer + PostgreSQL 16 + Redis 7</p>
    <div class="grid">
      <div class="card"><div class="label">Pico Usuarios Concurrentes (VUs)</div><div class="val ok">${metricVal(data, 'vus', 'max')}</div></div>
      <div class="card"><div class="label">Total Peticiones HTTP</div><div class="val">${metricVal(data, 'http_reqs', 'count')}</div></div>
      <div class="card"><div class="label">Throughput (req/s)</div><div class="val">${metricVal(data, 'http_reqs', 'rate')}</div></div>
      <div class="card"><div class="label">P95 Puja Concurrente</div><div class="val ok">${metricVal(data, 'livebid_bid_duration_ms', 'p(95)')} ms</div></div>
      <div class="card"><div class="label">Tasa de Checks Exitosos</div><div class="val ok">${checksPct}</div></div>
      <div class="card"><div class="label">Errores 5xx</div><div class="val ok">${metricVal(data, 'livebid_http_5xx_errors', 'count')}</div></div>
      <div class="card"><div class="label">Conexiones WebSocket</div><div class="val">${metricVal(data, 'livebid_ws_connected', 'count')}</div></div>
      <div class="card"><div class="label">Eventos WS Difundidos</div><div class="val">${metricVal(data, 'ws_msgs_received', 'count')}</div></div>
    </div>
    <table>
      <thead><tr><th>Métrica</th><th>Avg</th><th>Med (P50)</th><th>P90</th><th>P95</th><th>Max</th></tr></thead>
      <tbody>
        <tr><td>http_req_duration (General)</td><td>${metricVal(data, 'http_req_duration', 'avg')} ms</td><td>${metricVal(data, 'http_req_duration', 'med')} ms</td><td>${metricVal(data, 'http_req_duration', 'p(90)')} ms</td><td>${metricVal(data, 'http_req_duration', 'p(95)')} ms</td><td>${metricVal(data, 'http_req_duration', 'max')} ms</td></tr>
        <tr><td>livebid_bid_duration_ms (Pujas)</td><td>${metricVal(data, 'livebid_bid_duration_ms', 'avg')} ms</td><td>${metricVal(data, 'livebid_bid_duration_ms', 'med')} ms</td><td>${metricVal(data, 'livebid_bid_duration_ms', 'p(90)')} ms</td><td>${metricVal(data, 'livebid_bid_duration_ms', 'p(95)')} ms</td><td>${metricVal(data, 'livebid_bid_duration_ms', 'max')} ms</td></tr>
      </tbody>
    </table>
  </div>
</body>
</html>`;

  return {
    '/scripts/results/k6-summary.json': JSON.stringify(data, null, 2),
    '/scripts/results/k6-report.html': html,
  };
}
