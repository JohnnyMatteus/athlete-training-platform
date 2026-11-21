#!/bin/sh
# Executado pelo entrypoint da imagem na primeira inicialização do volume.
set -eu
psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d postgres -f /docker-entrypoint-initdb.d/init.sql
psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d atividades -c 'CREATE EXTENSION IF NOT EXISTS timescaledb;'
