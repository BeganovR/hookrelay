INSERT INTO workspaces (name, description)
SELECT
    'Workspace-' || s.i,
    'Workspace number ' || s.i
FROM generate_series(1, 2 * :seed_count) AS s(i)
WHERE NOT EXISTS (
    SELECT 1 FROM workspaces WHERE name = 'Workspace-' || s.i
);

INSERT INTO api_keys (workspace_id, name, token_hash)
SELECT
    w.id,
    'key-' || s.i,
    md5('token-' || w.id::text || '-' || s.i)
FROM workspaces w
CROSS JOIN generate_series(1, 2) AS s(i)
ON CONFLICT (token_hash) DO NOTHING;
