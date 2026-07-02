BEGIN;

DROP PUBLICATION IF EXISTS debezium_pub;

ALTER TABLE public.delivery_attempts REPLICA IDENTITY DEFAULT;

DO $$
BEGIN
    IF EXISTS (SELECT FROM pg_roles WHERE rolname = 'debezium') THEN
        EXECUTE 'DROP OWNED BY debezium';
        EXECUTE 'DROP USER debezium';
    END IF;
END $$;

COMMIT;
