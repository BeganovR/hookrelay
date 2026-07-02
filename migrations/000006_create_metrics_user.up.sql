DO $$
BEGIN
    IF NOT EXISTS (SELECT FROM pg_roles WHERE rolname = 'metrics_user') THEN
        CREATE USER metrics_user WITH PASSWORD 'metrics_user_password';
    END IF;
END $$;

GRANT CONNECT ON DATABASE hookrelay TO metrics_user;
GRANT pg_monitor TO metrics_user;
