DO $$
BEGIN
    IF EXISTS (SELECT FROM pg_roles WHERE rolname = 'metrics_user') THEN
        REVOKE pg_monitor FROM metrics_user;
        DROP OWNED BY metrics_user;
        DROP USER metrics_user;
    END IF;
END $$;