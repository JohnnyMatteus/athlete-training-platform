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
migrate_ planejamento up

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
assert "$(psql_ -d metricas -c "SELECT count(*) FROM information_schema.columns WHERE table_name IN ('outbox') AND column_name='traceparent'")" 1 "traceparent na outbox de metricas"
assert "$(psql_ -d metricas -c "SELECT count(*) FROM information_schema.tables WHERE table_name='vinculo_treinador'")" 1 "vinculo_treinador"
assert "$(psql_ -d atividades -c "SELECT count(*) FROM information_schema.columns WHERE table_name='outbox' AND column_name='traceparent'")" 1 "traceparent na outbox de atividades"
psql_ -d metricas -c "INSERT INTO limiar VALUES ('2f0f0a2e-6a53-4c3a-9d8e-1b7e6c5d4a01', 'ftp', 250, '2026-01-01')" >/dev/null
if psql_ -d metricas -c "INSERT INTO limiar VALUES ('2f0f0a2e-6a53-4c3a-9d8e-1b7e6c5d4a01', 'ftp', 260, '2026-01-01')" >/dev/null 2>&1; then
  echo "FALHOU: limiar duplicado na mesma data foi aceito"; exit 1
fi

# Semântica de concorrência das gravações do Planejamento (mesmas instruções de PgPlanRepository e PgLinkRepository)
psql_ -d planejamento -c "INSERT INTO plano (id, atleta_id, status, versao_plano, prova, data_prova, inicio, conteudo, revisao)
  VALUES ('b1c2d3e4-f5a6-4b7c-8d9e-0a1b2c3d4e5f', '2f0f0a2e-6a53-4c3a-9d8e-1b7e6c5d4a01', 'aguardando_aprovacao', 1, 'meio_ironman', '2027-06-06', '2026-11-30', '{}', 1)" >/dev/null
cas() { psql_ -d planejamento -c "WITH u AS (UPDATE plano SET status='ativo', revisao = revisao + 1 WHERE id='b1c2d3e4-f5a6-4b7c-8d9e-0a1b2c3d4e5f' AND revisao = $1 RETURNING 1) SELECT count(*) FROM u"; }
assert "$(cas 1)" 1 "primeira gravação com a revisão lida passa"
assert "$(cas 1)" 0 "gravação com revisão defasada não passa"
psql_ -d planejamento -c "INSERT INTO vinculo_treinador (id, atleta_id, treinador_id, status, categorias)
  VALUES ('d4e5f6a7-b8c9-4d0e-9f1a-2b3c4d5e6f7a', '2f0f0a2e-6a53-4c3a-9d8e-1b7e6c5d4a01', 'e5f6a7b8-c9d0-4e1f-8a2b-3c4d5e6f7a8b', 'revogado', '{plano}')" >/dev/null
assert "$(psql_ -d planejamento -c "WITH u AS (UPDATE vinculo_treinador SET status='ativo' WHERE id='d4e5f6a7-b8c9-4d0e-9f1a-2b3c4d5e6f7a' AND status='pendente' AND status <> 'revogado' RETURNING 1) SELECT count(*) FROM u")" 0 "aceite não ressuscita vínculo revogado"
migrate_ planejamento down -all
migrate_ metricas down -all
migrate_ atividades down -all
migrate_ importacao down -all
assert "$(psql_ -d atividades -c "SELECT count(*) FROM pg_tables WHERE schemaname='public' AND tablename<>'schema_migrations'")" 0 "atividades vazio após rollback"
assert "$(psql_ -d importacao -c "SELECT count(*) FROM pg_tables WHERE schemaname='public' AND tablename<>'schema_migrations'")" 0 "importacao vazio após rollback"

assert "$(psql_ -d metricas -c "SELECT count(*) FROM pg_tables WHERE schemaname='public' AND tablename<>'schema_migrations'")" 0 "metricas vazio após rollback"
assert "$(psql_ -d planejamento -c "SELECT count(*) FROM pg_tables WHERE schemaname='public' AND tablename<>'schema_migrations'")" 0 "planejamento vazio após rollback"

migrate_ importacao up
migrate_ atividades up
migrate_ metricas up
migrate_ planejamento up
echo "migrações ok"
