BEGIN;

DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'debezium') THEN
        CREATE USER debezium WITH PASSWORD 'debezium_password' REPLICATION LOGIN;
    END IF;
END $$;

GRANT CONNECT ON DATABASE hookrelay TO debezium;
GRANT USAGE ON SCHEMA public TO debezium;
GRANT SELECT ON TABLE public.delivery_attempts TO debezium;

ALTER TABLE public.delivery_attempts REPLICA IDENTITY FULL;

DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_publication WHERE pubname = 'debezium_pub') THEN
        EXECUTE 'CREATE PUBLICATION debezium_pub FOR TABLE public.delivery_attempts';
    END IF;
END $$;

COMMIT;
