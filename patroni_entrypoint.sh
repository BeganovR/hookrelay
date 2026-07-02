#!/bin/bash
set -e

mkdir -p /var/lib/postgresql/data
chown postgres:postgres /var/lib/postgresql/data

envsubst < /etc/patroni/patroni.yml.template > /etc/patroni/patroni.yml

exec gosu postgres patroni /etc/patroni/patroni.yml
