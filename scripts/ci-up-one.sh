#!/bin/bash
set -e

MIGRATION_FILE=$1
VERSION=$(basename "$MIGRATION_FILE" | grep -oE '^[0-9]+')
TEST_DB_URL="postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${POSTGRES_HOST:-postgres}:${POSTGRES_PORT:-5432}/${TEST_POSTGRES_DB}?sslmode=disable"

migrate -path ./migrations -database "${TEST_DB_URL}" goto ${VERSION} 2>&1
