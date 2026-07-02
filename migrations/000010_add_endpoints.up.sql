BEGIN;

CREATE TABLE IF NOT EXISTS endpoints (
    id           UUID          PRIMARY KEY DEFAULT gen_random_uuid(),
    project_id   UUID          NOT NULL REFERENCES workspaces(id),
    name         VARCHAR(100)  NOT NULL,
    url          VARCHAR(2000) NOT NULL,
    description  TEXT,
    http_timeout INT           NOT NULL DEFAULT 30,
    auth_type    VARCHAR(20)   NOT NULL DEFAULT 'none',
    auth_cfg     JSONB,
    is_active    BOOLEAN       NOT NULL DEFAULT true,
    created_at   TIMESTAMPTZ   NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ   NOT NULL DEFAULT now()
);

ALTER TABLE subscriptions ADD COLUMN IF NOT EXISTS endpoint_id UUID REFERENCES endpoints(id);

COMMIT;
