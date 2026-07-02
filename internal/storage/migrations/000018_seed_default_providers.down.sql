BEGIN;

DELETE FROM providers WHERE name IN ('github', 'stripe', 'telegram', 'custom');

COMMIT;
