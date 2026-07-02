#!/bin/bash
set -e

DB_URL="postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${POSTGRES_HOST:-postgres}:${POSTGRES_PORT:-5432}/${POSTGRES_DB}?sslmode=disable"
TEST_DB_URL="postgres://${POSTGRES_USER}:${POSTGRES_PASSWORD}@${POSTGRES_HOST:-postgres}:${POSTGRES_PORT:-5432}/${TEST_POSTGRES_DB}?sslmode=disable"

until psql "${DB_URL}" -c '\q' 2>/dev/null; do
    sleep 2
done

TARGET_VERSION=$(ls ./migrations/*.up.sql | sed -E 's#.*/([0-9]+)_.*#\1#' | sort -n | tail -1 | sed 's/^0*//')
APPLIED_VERSION=$(psql "${DB_URL}" -tAc "SELECT COALESCE(MAX(version), 0) FROM schema_migrations WHERE NOT dirty;" 2>/dev/null || echo 0)

if [ "${APPLIED_VERSION}" = "${TARGET_VERSION}" ]; then
    echo "БД уже на версии ${APPLIED_VERSION} — staircase-тест и сиды пропущены."
elif [ "${APPLIED_VERSION}" = "0" ]; then
    echo "Пересоздаём тестовую базу данных..."
    psql "${DB_URL}" -c "DROP DATABASE IF EXISTS ${TEST_POSTGRES_DB};"
    psql "${DB_URL}" -c "CREATE DATABASE ${TEST_POSTGRES_DB};"

    echo "Запускаем Seqwall — тестируем миграции..."
    seqwall staircase \
        --postgres-url "${TEST_DB_URL}" \
        --migrations-path ./migrations \
        --upgrade "./scripts/ci-up-one.sh {current_migration}" \
        --downgrade "./scripts/ci-down-one.sh {current_migration}" \
        --migrations-extension ".up.sql"

    echo "Тесты прошли успешно. Применяем миграции к основной БД..."

    if [ -n "${MIGRATION_VERSION}" ]; then
        migrate -path ./migrations -database "${DB_URL}" goto ${MIGRATION_VERSION}
    else
        migrate -path ./migrations -database "${DB_URL}" up
    fi

    if [ "${APP_ENV}" != "prod" ]; then
        for v in $(seq 1 "${TARGET_VERSION}"); do
            [ -f "./seeds/seed_v${v}.sql" ] && psql "${DB_URL}" -v seed_count="${SEED_COUNT:-10}" -f "./seeds/seed_v${v}.sql"
        done
    fi
else
    echo "БД уже забутстраплена (версия ${APPLIED_VERSION}) — накатываем только новые миграции без staircase-теста, т.к. роль debezium уже используется живым Kafka Connect коннектором."
    migrate -path ./migrations -database "${DB_URL}" up
fi

echo "Готово!"