#!/bin/sh
# Executado pelo entrypoint da imagem na primeira inicialização do volume.
# O init.sql fica fora de initdb.d: lá dentro o entrypoint o executaria uma segunda vez.
set -eu
psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d postgres -f /opt/atp/init.sql
psql -v ON_ERROR_STOP=1 -U "$POSTGRES_USER" -d atividades -c 'CREATE EXTENSION IF NOT EXISTS timescaledb;'
