#!/bin/bash
set -euo pipefail

log() { printf '[%s] %s\n' "$(date '+%H:%M:%S')" "$*"; }

cd "$(dirname "$0")/.."

get_role() {
    docker compose exec -T "$1" \
        curl -s http://localhost:8008/patroni 2>/dev/null \
        | python3 -c "import json,sys; print(json.load(sys.stdin).get('role',''))" 2>/dev/null || echo ""
}

PRIMARY=""
for node in patroni1 patroni2; do
    if [ "$(get_role "$node")" = "primary" ]; then
        PRIMARY=$node; break
    fi
done
[ -z "$PRIMARY" ] && { log "primary не найден"; exit 1; }

REPLICA=$([ "$PRIMARY" = "patroni1" ] && echo "patroni2" || echo "patroni1")
log "primary=$PRIMARY replica=$REPLICA"

log "Кластер до сбоя:"
docker compose exec -T "$PRIMARY" patronictl -c /etc/patroni/patroni.yml list

log "Доступность через хапроски до сбоя:"
docker compose exec -T "$REPLICA" pg_isready -h haproxy -p 5000

CONTAINER=$(docker compose ps -q "$PRIMARY" | xargs docker inspect --format '{{.Name}}' | sed 's/^\///')
log "Инъекция сбоя: Pumba SIGKILL -> $CONTAINER"
docker run --rm -v /var/run/docker.sock:/var/run/docker.sock \
    gaiaadm/pumba kill --signal SIGKILL "$CONTAINER"

START_TS=$(date +%s)
log "Ждём нового лидера..."

for i in $(seq 1 24); do
    sleep 5
    ELAPSED=$(( $(date +%s) - START_TS ))
    role=$(get_role "$REPLICA")
    if [ "$role" = "primary" ]; then
        log "Новый primary: $REPLICA (failover за ${ELAPSED}s)"
        break
    fi
    log "  [${ELAPSED}s] $REPLICA role=$role"
done

log "Ждём обновления хапрокси на health check..."
for i in $(seq 1 6); do
    docker compose exec -T "$REPLICA" pg_isready -h haproxy -p 5000 -q 2>/dev/null && break || true
    sleep 3
done

log "Доступность через хапроски после failover:"
docker compose exec -T "$REPLICA" pg_isready -h haproxy -p 5000 \
    && log "  порт 5000: OK" || log "  порт 5000: недоступен"

log "Восстанавливаем $PRIMARY..."
docker compose start "$PRIMARY"
sleep 25

log "Финальное состояние:"
docker compose exec -T "$REPLICA" patronictl -c /etc/patroni/patroni.yml list
