#!/bin/bash
set -e
DC=${DOCKER_COMPOSE:-docker compose}
$DC up -d --wait haproxy
$DC run --rm --no-deps migrations
