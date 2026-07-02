#!/bin/sh
mc alias set minio "http://${MINIO_ENDPOINT:-minio:9000}" "${MINIO_BACKUP_USER}" "${MINIO_BACKUP_PASSWORD}"
echo "${BACKUP_INTERVAL} /usr/local/bin/backup.sh" > /tmp/crontab
exec supercronic /tmp/crontab
