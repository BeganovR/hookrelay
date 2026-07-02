CREATE DATABASE IF NOT EXISTS analytics;

CREATE TABLE IF NOT EXISTS analytics.delivery_attempts_kafka
(
    id               String,
    delivery_id      String,
    attempt_number   Int32,
    status           String,
    http_code        Nullable(Int32),
    response_body    Nullable(String),
    response_time_ms Int32,
    created_at       String
)
ENGINE = Kafka
SETTINGS
    kafka_broker_list          = 'kafka:9092',
    kafka_topic_list           = 'hookrelay.public.delivery_attempts',
    kafka_group_name           = 'ch_hookrelay_delivery',
    kafka_format               = 'JSONEachRow',
    kafka_skip_broken_messages = 10;

CREATE TABLE IF NOT EXISTS analytics.delivery_attempts
(
    id               String,
    delivery_id      String,
    attempt_number   Int32,
    status           LowCardinality(String),
    http_code        Nullable(Int32),
    response_time_ms Int32,
    created_at       DateTime('UTC')
)
ENGINE = MergeTree
PARTITION BY toYYYYMM(created_at)
ORDER BY (status, created_at, id);

CREATE MATERIALIZED VIEW IF NOT EXISTS analytics.delivery_attempts_mv
TO analytics.delivery_attempts
AS SELECT
    id,
    delivery_id,
    attempt_number,
    status,
    http_code,
    response_time_ms,
    parseDateTimeBestEffort(created_at) AS created_at
FROM analytics.delivery_attempts_kafka;

CREATE USER IF NOT EXISTS metabase_ro IDENTIFIED WITH plaintext_password BY 'metabase_password';
GRANT SELECT ON analytics.* TO metabase_ro;
