#!/bin/sh
set -e

until mc alias set minio "http://minio:9000" "${MINIO_ROOT_USER}" "${MINIO_ROOT_PASSWORD}"; do
    sleep 1
done

mc mb --ignore-existing "minio/${BUCKET_BACKUP_NAME}"

printf '{"Version":"2012-10-17","Statement":[{"Effect":"Allow","Action":["s3:GetObject","s3:PutObject","s3:DeleteObject","s3:ListBucket","s3:GetBucketLocation"],"Resource":["arn:aws:s3:::%s","arn:aws:s3:::%s/*"]}]}' \
    "${BUCKET_BACKUP_NAME}" "${BUCKET_BACKUP_NAME}" > /tmp/policy.json

mc admin policy create minio backup-policy /tmp/policy.json || true
mc admin user add minio "${MINIO_BACKUP_USER}" "${MINIO_BACKUP_PASSWORD}" || true
mc admin policy attach minio backup-policy --user "${MINIO_BACKUP_USER}" || true
