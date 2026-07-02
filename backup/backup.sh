#!/bin/sh
set -e

TIMESTAMP=$(date +%Y%m%dT%H%M%S)
FILENAME="backup_${TIMESTAMP}.dump"
TMPFILE="/tmp/${FILENAME}"

PGPASSWORD="${POSTGRES_PASSWORD}" pg_dump \
    -h "${POSTGRES_HOST}" \
    -p "${POSTGRES_PORT}" \
    -U "${POSTGRES_USER}" \
    -d "${POSTGRES_DB}" \
    -Fc \
    -f "${TMPFILE}"

mc cp "${TMPFILE}" "minio/${BUCKET_BACKUP_NAME}/${FILENAME}"
rm -f "${TMPFILE}"

OBJECTS=$(mc ls "minio/${BUCKET_BACKUP_NAME}" 2>/dev/null | sort | awk '{print $NF}')
COUNT=$(printf '%s\n' "${OBJECTS}" | grep -c '[^[:space:]]' || echo 0)
TO_DELETE=$((COUNT - BACKUP_RETENTION_COUNT))

if [ "${TO_DELETE}" -gt 0 ]; then
    printf '%s\n' "${OBJECTS}" | head -n "${TO_DELETE}" | while read -r name; do
        mc rm "minio/${BUCKET_BACKUP_NAME}/${name}"
    done
fi
