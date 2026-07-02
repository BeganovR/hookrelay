WITH numbered AS (
    SELECT id, row_number() OVER (ORDER BY id) AS rn
    FROM delivery_attempts
)
UPDATE delivery_attempts da
SET created_at = NOW() - INTERVAL '20 days' + (numbered.rn * INTERVAL '86 seconds')
FROM numbered
WHERE da.id = numbered.id;
