#!/usr/bin/env bash
set -euo pipefail

# ==============================================================================
# PRUEBA DE TOLERANCIA A FALLOS Y ALTA DISPONIBILIDAD (CHAOS / FAILOVER 99.95%)
# Verifica experimentalmente que al caer abruptamente una réplica Go (livebid-api-2),
# el Gateway Nginx conmuta el 100% del tráfico hacia livebid-api-1 con 0% de errores.
# ==============================================================================

BASE_URL="${BASE_URL:-http://localhost:3000}"
REPLICA_TO_STOP="livebid-api-2"

restore_replica() {
  echo "[Failover HA] Restaurando réplica ${REPLICA_TO_STOP}..."
  docker start "${REPLICA_TO_STOP}" >/dev/null 2>&1 || true
  for _ in $(seq 1 30); do
    STATUS=$(docker inspect -f '{{.State.Health.Status}}' "${REPLICA_TO_STOP}" 2>/dev/null || echo "starting")
    if [ "${STATUS}" = "healthy" ]; then
      break
    fi
    sleep 1
  done
}
trap restore_replica EXIT

echo "[Failover HA] 1/4 Verificando balanceo inicial entre las 2 réplicas Go..."
SEEN_BEFORE=""
for _ in $(seq 1 8); do
  INST=$(curl -sf "${BASE_URL}/health" | grep -o '"instance":"[^"]*"' | cut -d'"' -f4)
  SEEN_BEFORE="${SEEN_BEFORE} ${INST}"
done
echo "  Instancias activas antes del fallo:${SEEN_BEFORE}"
echo "${SEEN_BEFORE}" | grep -q "livebid-api-1"
echo "${SEEN_BEFORE}" | grep -q "livebid-api-2"

echo "[Failover HA] 2/4 Simulando caída abrupta de nodo (${REPLICA_TO_STOP})..."
docker stop -t 1 "${REPLICA_TO_STOP}" >/dev/null

echo "[Failover HA] 3/4 Enviando 40 peticiones continuas durante la caída (meta: 0 errores)..."
SUCCESS_COUNT=0
FAIL_COUNT=0
for _ in $(seq 1 20); do
  CODE_H=$(curl -s -o /dev/null -w "%{http_code}" "${BASE_URL}/health")
  CODE_A=$(curl -s -o /dev/null -w "%{http_code}" "${BASE_URL}/api/v1/subastas")
  if [ "${CODE_H}" = "200" ]; then SUCCESS_COUNT=$((SUCCESS_COUNT + 1)); else FAIL_COUNT=$((FAIL_COUNT + 1)); fi
  if [ "${CODE_A}" = "200" ]; then SUCCESS_COUNT=$((SUCCESS_COUNT + 1)); else FAIL_COUNT=$((FAIL_COUNT + 1)); fi
done

echo "  Resultado en degradación: ${SUCCESS_COUNT}/40 respuestas 200 OK | Fallos: ${FAIL_COUNT}"
if [ "${FAIL_COUNT}" -ne 0 ]; then
  echo "ERROR: Se detectaron ${FAIL_COUNT} fallos durante el failover."
  exit 1
fi

SURVIVOR=$(curl -sf "${BASE_URL}/health" | grep -o '"instance":"[^"]*"' | cut -d'"' -f4)
echo "  Nodo sobreviviente sirviendo tráfico sin interrupción: ${SURVIVOR}"
test "${SURVIVOR}" = "livebid-api-1"

echo "[Failover HA] 4/4 Recuperando nodo y verificando retorno al clúster balanceado..."
restore_replica
trap - EXIT

SEEN_AFTER=""
for _ in $(seq 1 10); do
  INST=$(curl -sf "${BASE_URL}/health" | grep -o '"instance":"[^"]*"' | cut -d'"' -f4)
  SEEN_AFTER="${SEEN_AFTER} ${INST}"
done
echo "  Instancias activas tras auto-recuperación:${SEEN_AFTER}"
echo "${SEEN_AFTER}" | grep -q "livebid-api-1"
echo "${SEEN_AFTER}" | grep -q "livebid-api-2"

echo "OK: Prueba de Resiliencia y Alta Disponibilidad (0% downtime) SUPERADA."
