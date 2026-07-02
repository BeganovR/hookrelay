BEGIN;

CREATE TABLE IF NOT EXISTS subscriptions (
    id         UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    source_id  UUID          NOT NULL REFERENCES sources(id),
    target_url VARCHAR(2000) NOT NULL,
    secret     VARCHAR(255),
    created_at TIMESTAMPTZ   NOT NULL DEFAULT now()
);

CREATE TABLE IF NOT EXISTS deliveries (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    event_id        UUID        NOT NULL REFERENCES events(id),
    subscription_id UUID        NOT NULL REFERENCES subscriptions(id),
    status          VARCHAR(50) NOT NULL DEFAULT 'pending',
    attempts_count  INT         NOT NULL DEFAULT 0,
    next_retry_at   TIMESTAMPTZ
);

CREATE TABLE IF NOT EXISTS delivery_attempts (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    delivery_id      UUID        NOT NULL REFERENCES deliveries(id),
    attempt_number   INT         NOT NULL,
    status           VARCHAR(50) NOT NULL,
    http_code        INT,
    response_body    TEXT,
    response_time_ms INT         NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

COMMIT;