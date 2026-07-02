#!/bin/bash
set -euo pipefail

log() { printf '[%s] %s\n' "$(date '+%H:%M:%S')" "$*"; }

cd "$(dirname "$0")/.."

node_state() {
    docker compose exec -T "$1" \
        curl -s --max-time 3 http://localhost:8008/patroni 2>/dev/null \
        | python3 -c "
import json, sys, time
try:
    d = json.load(sys.stdin)
    dcs = d.get('dcs_last_seen', 0)
    lag = round(time.time() - dcs, 0) if dcs else '?'
    print(f'role={d.get(\"role\",\"?\")} state={d.get(\"state\",\"?\")} dcs_lag={lag}s')
except Exception:
    print('unreachable')
" 2>/dev/null || echo "unreachable"
}

patroni_list() {
    docker compose exec -T "$1" \
        timeout 8 patronictl -c /etc/patroni/patroni.yml list 2>/dev/null || true
}

check_db() {
    docker compose exec -T patroni1 pg_isready -h haproxy -p 5000 -q 2>/dev/null \
        && echo "доступна" || echo "недоступна"
}

log "Кластер до сбоя:"
patroni_list patroni1
log "PostgreSQL через HAProxy: $(check_db)"

log "Останавливаем etcd (все 3 ноды)..."
docker compose stop etcd1 etcd2 etcd3
START_TS=$(date +%s)

log "Мониторинг состояния кластера..."
DB_LOGGED=0
for i in 1 2 3 4 5 6; do
    sleep 10
    ELAPSED=$(( $(date +%s) - START_TS ))
    log "[${ELAPSED}s] patroni1: $(node_state patroni1)"
    log "[${ELAPSED}s] patroni2: $(node_state patroni2)"
    if [ $DB_LOGGED -eq 0 ]; then
        DB_STATUS=$(check_db)
        log "[${ELAPSED}s] PostgreSQL: $DB_STATUS"
        [ "$DB_STATUS" = "недоступна" ] && DB_LOGGED=1 || true
    fi
done

log "Попытка failover без DCS:"
docker compose exec -T patroni1 \
    timeout 10 patronictl -c /etc/patroni/patroni.yml failover postgres-cluster --force 2>&1 \
    | grep -v "WARNING\|ERROR\|Failed\|MaxRetry\|NewConn\|getaddr\|etcd" | grep . || true
log "Failover без DCS невозможен"

log "Восстанавливаем etcd..."
docker compose rm -sf etcd1 etcd2 etcd3

PROJECT=$(docker compose config --format json 2>/dev/null \
    | python3 -c "import json,sys; print(json.load(sys.stdin).get('name',''))" 2>/dev/null \
    || echo "m3207_beganovrz")
docker volume rm "${PROJECT}_etcd1_data" "${PROJECT}_etcd2_data" "${PROJECT}_etcd3_data" 2>/dev/null || true
docker compose up -d etcd1 etcd2 etcd3
RESTORE_TS=$(date +%s)

until docker compose exec -T etcd1 \
    /usr/local/bin/etcdctl --endpoints=http://localhost:2379 endpoint health 2>/dev/null \
    | grep -q "healthy"; do sleep 2; done
log "etcd healthy"

log "Перезапускаем Patroni..."
docker compose restart patroni1 patroni2

until patroni_list patroni1 2>/dev/null | grep -q "Leader"; do sleep 3; done
ELAPSED=$(( $(date +%s) - RESTORE_TS ))
log "Лидер выбран за ${ELAPSED}s"

log "Финальное состояние кластера:"
patroni_list patroni1

for i in $(seq 1 5); do
    DB_STATUS=$(check_db)
    [ "$DB_STATUS" = "доступна" ] && break || true
    sleep 4
done
log "PostgreSQL через HAProxy после восстановления: $DB_STATUS"
