#!/bin/sh
psql "${1}" -c "CREATE USER ${POSTGRES_USER} WITH PASSWORD '${POSTGRES_PASSWORD}' SUPERUSER CREATEDB;" || true
psql "${1}" -c "CREATE USER metrics_user WITH PASSWORD '${METRICS_USER_PASSWORD}';" || true
psql "${1}" -c "CREATE DATABASE ${POSTGRES_DB} OWNER ${POSTGRES_USER};" || true
psql "${1}" -c "GRANT pg_monitor TO metrics_user; GRANT CONNECT ON DATABASE ${POSTGRES_DB} TO metrics_user;" || true
