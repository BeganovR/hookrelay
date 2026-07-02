INSERT INTO providers (name, signature_algorithm)
VALUES
    ('github', 'sha256'),
    ('stripe', 'sha256'),
    ('twilio', 'sha1'),
    ('slack', 'sha256'),
    ('shopify', 'sha256')
ON CONFLICT (name) DO NOTHING;

INSERT INTO sources (workspace_id, provider_id, slug, signing_secret, is_active)
SELECT
    w.id,
    p.id,
    p.name || '-' || w.id || '-' || s.i,
    md5('secret-' || w.id::text || s.i::text),
    true
FROM (SELECT id FROM workspaces LIMIT :seed_count) w
CROSS JOIN (SELECT id, name FROM providers) p
CROSS JOIN generate_series(1, 2) AS s(i)
ON CONFLICT (slug) DO NOTHING;
