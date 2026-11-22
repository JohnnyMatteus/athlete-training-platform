#!/usr/bin/env bash
# Valida as migrações: sobe, confere o esquema, reverte tudo e sobe de novo.
# Sem PGHOST definido, cria um container Timescale descartável; no CI usa o service container.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
export PGUSER="${PGUSER:-atp}" PGPASSWORD="${PGPASSWORD:-dev-only-change-me}" PGPORT="${PGPORT:-55432}"
MIGRATE_IMAGE="migrate/migrate:v4.18.1"
CONTAINER=""

cleanup() { [ -z "$CONTAINER" ] || docker rm -f "$CONTAINER" >/dev/null 2>&1 || true; }
trap cleanup EXIT

if [ -z "${PGHOST:-}" ]; then
  export PGHOST=127.0.0.1
  CONTAINER=atp-migrations-test
  docker run -d --name "$CONTAINER" -e POSTGRES_USER="$PGUSER" -e POSTGRES_PASSWORD="$PGPASSWORD" \
    -p "$PGPORT:5432" timescale/timescaledb:2.17.2-pg16 >/dev/null
fi

for _ in $(seq 1 60); do pg_isready_out=$(docker run --rm --network host -e PGPASSWORD timescale/timescaledb:2.17.2-pg16 \
  pg_isready -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" 2>&1) && break; sleep 1; done

psql_() { docker run --rm --network host -e PGPASSWORD -v "$ROOT/infra/postgres:/init:ro" timescale/timescaledb:2.17.2-pg16 \
  psql -v ON_ERROR_STOP=1 -h "$PGHOST" -p "$PGPORT" -U "$PGUSER" -tA "$@"; }
migrate_() { # <banco> <comando...>
  local db=$1; shift
  docker run --rm --network host -v "$ROOT/infra/db/migrations/$db:/m:ro" "$MIGRATE_IMAGE" \
    -path /m -database "postgres://$PGUSER:$PGPASSWORD@$PGHOST:$PGPORT/$db?sslmode=disable" "$@"
}
assert() { [ "$1" = "$2" ] || { echo "FALHOU: $3 (esperado '$2', obtido '$1')"; exit 1; }; }

psql_ -d postgres -f /init/init.sql >/dev/null
psql_ -d atividades -c 'CREATE EXTENSION IF NOT EXISTS timescaledb;' >/dev/null

migrate_ importacao up
migrate_ atividades up
migrate_ metricas up

assert "$(psql_ -d atividades -c "SELECT hypertable_name FROM timescaledb_information.hypertables")" amostra "amostra é hypertable"
assert "$(psql_ -d atividades -c "SELECT count(*) FROM timescaledb_information.dimensions WHERE column_name='ts'")" 1 "particionada por ts"

psql_ -d atividades -c "INSERT INTO atividade (id, atleta_id, tipo, inicio_utc, duracao_s, chave_natural, fonte)
  VALUES (gen_random_uuid(), '2f0f0a2e-6a53-4c3a-9d8e-1b7e6c5d4a01', 'corrida', now(), 60, 'K', 'fit')" >/dev/null
if psql_ -d atividades -c "INSERT INTO atividade (id, atleta_id, tipo, inicio_utc, duracao_s, chave_natural, fonte)
  VALUES (gen_random_uuid(), '2f0f0a2e-6a53-4c3a-9d8e-1b7e6c5d4a01', 'corrida', now(), 60, 'K', 'fit')" >/dev/null 2>&1; then
  echo "FALHOU: chave natural duplicada foi aceita"; exit 1
fi
assert "$(psql_ -d importacao -c "SELECT count(*) FROM information_schema.columns WHERE table_name='lote_importacao'
  AND column_name IN ('import_id','atleta_id','sha256','tipo_arquivo','status','versao_formato','importadas','duplicadas','erros')")" 9 "colunas do lote"

assert "$(psql_ -d metricas -c "SELECT count(*) FROM information_schema.columns WHERE table_name='carga_diaria'
  AND column_name IN ('atleta_id','dia','tss','ctl','atl','tsb')")" 6 "colunas de carga_diaria"
psql_ -d metricas -c "INSERT INTO limiar VALUES ('2f0f0a2e-6a53-4c3a-9d8e-1b7e6c5d4a01', 'ftp', 250, '2026-01-01')" >/dev/null
if psql_ -d metricas -c "INSERT INTO limiar VALUES ('2f0f0a2e-6a53-4c3a-9d8e-1b7e6c5d4a01', 'ftp', 260, '2026-01-01')" >/dev/null 2>&1; then
  echo "FALHOU: limiar duplicado na mesma data foi aceito"; exit 1
fi

migrate_ metricas down -all
migrate_ atividades down -all
migrate_ importacao down -all
assert "$(psql_ -d atividades -c "SELECT count(*) FROM pg_tables WHERE schemaname='public' AND tablename<>'schema_migrations'")" 0 "atividades vazio após rollback"
assert "$(psql_ -d importacao -c "SELECT count(*) FROM pg_tables WHERE schemaname='public' AND tablename<>'schema_migrations'")" 0 "importacao vazio após rollback"

assert "$(psql_ -d metricas -c "SELECT count(*) FROM pg_tables WHERE schemaname='public' AND tablename<>'schema_migrations'")" 0 "metricas vazio após rollback"

migrate_ importacao up
migrate_ atividades up
migrate_ metricas up
echo "migrações ok"
