# HookRelay

Webhook ingestion, fan-out and retry service backed by PostgreSQL.

Entities: `Project → Source → Endpoint → Subscription → Event → EventDelivery → DeliveryAttempt`.

## Requirements

- Go 1.26+
- PostgreSQL 17
- Docker (optional, for `docker compose`)

## Quick start

```bash
cp .env.example .env
docker compose up -d
```

API listens on `:8080` (`PORT`, default `8080`).

## Configuration

| Variable | Default | Description |
|---|---|---|
| `DATABASE_URL` | — | PostgreSQL connection string (required) |
| `PORT` | `8080` | HTTP listen port |
| `BASE_URL` | `http://localhost:8080` | Used to build ingest URLs |
| `INGEST_RATE_PER_SEC` | `20` | Token bucket refill rate, per source |
| `INGEST_RATE_BURST` | `40` | Token bucket capacity, per source |
| `WORKER_CONCURRENCY` | `10` | Parallel delivery workers |
| `WORKER_POLL_INTERVAL` | `5s` | Poll interval for pending deliveries |
| `WORKER_BATCH_SIZE` | `100` | Deliveries claimed per poll |
| `WORKER_STUCK_TIMEOUT` | `5m` | Reset threshold for stuck `processing` deliveries |

## API

All `/v1/*` routes except `POST /v1/projects` require `Authorization: Bearer <api_key>`.

| Method | Path | Description |
|---|---|---|
| `POST` | `/v1/projects` | Create project, returns API key |
| `PATCH` | `/v1/projects/{id}` | Update project |
| `DELETE` | `/v1/projects/{id}` | Delete project |
| `POST` | `/v1/api-keys` | Create API key |
| `GET` | `/v1/api-keys` | List API keys |
| `DELETE` | `/v1/api-keys/{id}` | Revoke API key |
| `POST` | `/v1/sources` | Create source |
| `GET` | `/v1/sources` | List sources |
| `GET` | `/v1/sources/{id}` | Get source |
| `PATCH` | `/v1/sources/{id}` | Update source |
| `DELETE` | `/v1/sources/{id}` | Delete source |
| `POST` | `/v1/endpoints` | Create endpoint |
| `GET` | `/v1/endpoints` | List endpoints |
| `GET` | `/v1/endpoints/{id}` | Get endpoint |
| `PATCH` | `/v1/endpoints/{id}` | Update endpoint |
| `DELETE` | `/v1/endpoints/{id}` | Delete endpoint |
| `POST` | `/v1/subscriptions` | Create subscription |
| `GET` | `/v1/subscriptions` | List subscriptions |
| `GET` | `/v1/subscriptions/{id}` | Get subscription |
| `PATCH` | `/v1/subscriptions/{id}` | Update subscription |
| `DELETE` | `/v1/subscriptions/{id}` | Delete subscription |
| `GET` | `/v1/events` | List events |
| `GET` | `/v1/events/{id}` | Get event |
| `GET` | `/v1/events/{id}/deliveries` | List deliveries for event |
| `GET` | `/v1/deliveries/{id}` | Get delivery |
| `GET` | `/v1/deliveries/{id}/attempts` | List delivery attempts |
| `POST` | `/v1/deliveries/{id}/retry` | Re-queue a discarded delivery |
| `POST` | `/ingest/{source_uid}` | Receive an inbound webhook |
| `GET` | `/v1/healthz` | Health check |
| `GET` | `/metrics` | Prometheus metrics |

### Example

```bash
TOKEN=$(curl -s -X POST http://localhost:8080/v1/projects \
  -H "Content-Type: application/json" -d '{"name": "acme"}' | jq -r '.api_key.key')

SOURCE=$(curl -s -X POST http://localhost:8080/v1/sources \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"name": "github", "uid": "github", "verifier_type": "noop"}')
SOURCE_ID=$(echo "$SOURCE" | jq -r '.id')
SOURCE_UID=$(echo "$SOURCE" | jq -r '.uid')

ENDPOINT_ID=$(curl -s -X POST http://localhost:8080/v1/endpoints \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d '{"name": "my-service", "url": "https://my-service.internal/webhooks", "http_timeout": 10}' | jq -r '.id')

curl -X POST http://localhost:8080/v1/subscriptions \
  -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d "{\"name\": \"github-to-my-service\", \"source_id\": \"$SOURCE_ID\", \"endpoint_id\": \"$ENDPOINT_ID\", \"max_retries\": 5, \"retry_interval\": 10, \"retry_type\": \"exponential\"}"

curl -X POST http://localhost:8080/ingest/$SOURCE_UID \
  -H "Content-Type: application/json" -H "X-HookRelay-Event: push" \
  -d '{"ref": "refs/heads/main", "repository": "acme/app"}'
```

## Source verification (`verifier_type`)

| Type | Config (`verifier_cfg`) |
|---|---|
| `noop` | — |
| `hmac` | `header`, `prefix`, `secret`, `encoding` |

## Endpoint auth (`auth_type`)

| Type | Config (`auth_cfg`) |
|---|---|
| `none` | — |
| `bearer` | `token` |
| `basic` | `username`, `password` |
| `api_key` | `header`, `value` |

## Outbound signature

```
X-HookRelay-Delivery: <delivery_id>
X-HookRelay-Timestamp: <unix_timestamp>
X-HookRelay-Signature: v1,<base64(hmac_sha256(delivery_id + "." + timestamp + "." + body))>
```

Key: `subscription.signing_secret`.

## Retry policy

| `retry_type` | Delay before next attempt |
|---|---|
| `exponential` | `2^attempt × retry_interval` seconds, ±10% jitter, capped at 3600s |
| `linear` | `(attempt + 1) × retry_interval` seconds |
| `constant` | `retry_interval` seconds |

Delivery moves to `discarded` once attempts reach `max_retries`.

## Delivery status

`deliveries.status`:

| Status | |
|---|---|
| `pending` | Queued, waiting for `next_retry_at` |
| `processing` | Claimed by a worker |
| `success` | Last attempt got a 2xx response |
| `discarded` | `max_retries` exhausted |

`delivery_attempts.status` (one row per HTTP call):

| Status | |
|---|---|
| `success` | 2xx response |
| `fail` | Non-2xx response or request error |
| `timeout` | Request timed out or context deadline exceeded |

## Development

```bash
docker compose up -d

DATABASE_URL=postgres://<user>:<password>@localhost:5432/<db> go run ./cmd/hookrelay

go test ./internal/service/... ./internal/filter/... ./internal/verifier/... ./internal/auth/...

go test ./... -tags integration -count=1 -race -timeout 300s
```
