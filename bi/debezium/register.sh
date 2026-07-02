#!/bin/sh
set -eu

CONNECT_URL="${CONNECT_URL:-http://localhost:8083}"
SCRIPT_DIR="$(cd "$(dirname "$0")" && pwd)"

printf 'Waiting for Kafka Connect'
until curl -sf "$CONNECT_URL/connectors" > /dev/null 2>&1; do
    printf '.'
    sleep 3
done
printf '\n'

curl -s -X PUT "$CONNECT_URL/connectors/postgres-connector/config" \
    -H 'Content-Type: application/json' \
    -d "@$SCRIPT_DIR/connector.config.json"
printf '\n'
