CREATE TABLE IF NOT EXISTS webhooks (
    id UUID     PRIMARY KEY,
    client_id   VARCHAR(255)             NOT NULL,
    body        JSONB                    NOT NULL,
    status      VARCHAR(50)              NOT NULL DEFAULT 'pending',
    retry_count INTEGER                  NOT NULL DEFAULT 0,
    created_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);