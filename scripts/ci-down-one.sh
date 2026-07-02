#!/bin/bash
set -e

TEST_DB_URL="postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${POSTGRES_HOST:-postgres}:${POSTGRES_PORT:-5432}/${TEST_POSTGRES_DB}?sslmode=disable"

migrate -path ./migrations -database "${TEST_DB_URL}" down 1 2>&1
