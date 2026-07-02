import os
import time
from prometheus_client import start_http_server, Gauge
from minio import Minio

BUCKET = os.environ["BUCKET_BACKUP_NAME"]
MINIO_ENDPOINT = os.environ.get("MINIO_ENDPOINT", "minio:9000")
MINIO_ACCESS_KEY = os.environ["MINIO_BACKUP_USER"]
MINIO_SECRET_KEY = os.environ["MINIO_BACKUP_PASSWORD"]

last_backup_time = Gauge("backup_last_timestamp_seconds", "Unix timestamp of the last backup")
last_backup_size = Gauge("backup_last_size_bytes", "Size of the last backup in bytes")

client = Minio(MINIO_ENDPOINT, access_key=MINIO_ACCESS_KEY, secret_key=MINIO_SECRET_KEY, secure=False)


def update_metrics():
    try:
        objects = sorted(list(client.list_objects(BUCKET)), key=lambda o: o.last_modified)
        if objects:
            latest = objects[-1]
            last_backup_time.set(latest.last_modified.timestamp())
            last_backup_size.set(latest.size)
    except Exception:
        pass


start_http_server(8001)
while True:
    update_metrics()
    time.sleep(30)
