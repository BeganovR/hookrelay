BEGIN;

CREATE TABLE IF NOT EXISTS events (
    id              UUID         PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id       UUID         NOT NULL REFERENCES sources(id),
    idempotency_key VARCHAR(255),
    headers         JSONB        NOT NULL,
    payload         JSONB        NOT NULL,
    payload_size    INT          NOT NULL,
    sender_ip       INET         NOT NULL,
    received_at     TIMESTAMPTZ  NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_events_source_idempotency
    ON events (source_id, idempotency_key)
    WHERE idempotency_key IS NOT NULL;

COMMIT;