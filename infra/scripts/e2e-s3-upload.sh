#!/usr/bin/env bash
# Prova do S3 local (LocalStack) pelo compose real, em quatro blocos:
#   A. hooks forjados do tusd são recusados (sem token, token errado, token certo com bucket alheio) e o lote
#      continua em aguardando_upload;
#   B. upload multipart de bytes (dois PATCH de 6 MiB) pelo tusd: o objeto chega ao bucket com o sha256 enviado
#      e a API aceita o lote (sha256 conferido no post-finish);
#   C. upload do export reduzido (services/ingest/testdata): API, chave do objeto, S3, evento (relay) e worker
#      terminam com status `concluido` dentro do prazo;
#   D. a regra de expiração está gravada com o prazo certo.
# Pré-requisitos: `make dev-keys` feito e no ar api, api-relay, api-consumer, tusd, ingest, localstack, postgres e
# rabbitmq, com as migrações aplicadas. Qualquer divergência encerra com erro.
# Variáveis: API_URL, TUS_URL, UPLOADS_BUCKET, UPLOADS_RETENTION_DAYS, TUSD_HOOK_SECRET, ENV_FILE, DONE_TIMEOUT_S.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"
API_URL="${API_URL:-http://127.0.0.1:9501}"
TUS_URL="${TUS_URL:-http://127.0.0.1:8080/files/}"
BUCKET="${UPLOADS_BUCKET:-uploads}"
RETENTION="${UPLOADS_RETENTION_DAYS:-7}"
HOOK_SECRET="${TUSD_HOOK_SECRET:-dev-only-tusd-hook-secret}"
DONE_TIMEOUT_S="${DONE_TIMEOUT_S:-240}"
DC=(docker compose --profile core --env-file "${ENV_FILE:-.env.example}")
CHUNK=$((6 * 1024 * 1024))   # >= 5 MiB, o mínimo de uma parte do multipart do S3
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

fail() { echo "FALHA: $*" >&2; exit 1; }
awslocal() { "${DC[@]}" exec -T localstack awslocal "$@"; }
TOKEN="$(infra/scripts/dev-jwt.sh token)"

# create_import <arquivo> <nome> -> imprime o import_id
create_import() {
  local file="$1" name="$2" sha size body
  sha="$(sha256sum "$file" | cut -d' ' -f1)"; size="$(stat -c %s "$file")"
  body="$(jq -n --arg sha "$sha" --argjson size "$size" --arg name "$name" \
    '{nome_arquivo:$name,tamanho_bytes:$size,tipo_arquivo:"zip",sha256:$sha,consentimento:{aceito:true,versao_termo:"2026-11",categorias:["treinos"]}}')"
  curl -fsS --max-time 30 -X POST "$API_URL/imports" -H "Authorization: Bearer $TOKEN" \
    -H 'Content-Type: application/json' -d "$body" | jq -er .import_id
}

import_json() { curl -fsS --max-time 15 "$API_URL/imports/$1" -H "Authorization: Bearer $TOKEN"; }

# tus_upload <import_id> <arquivo> -> envia em partes de CHUNK bytes (o pre-create chama a API)
tus_upload() {
  local id="$1" file="$2" total meta location offset=0 new part
  total="$(stat -c %s "$file")"
  meta="import_id $(printf '%s' "$id" | base64 | tr -d '\n')"
  location="$(curl -fsS --max-time 30 -D - -o /dev/null -X POST "$TUS_URL" -H 'Tus-Resumable: 1.0.0' \
    -H "Upload-Length: $total" -H "Upload-Metadata: $meta" -H "Authorization: Bearer $TOKEN" \
    | tr -d '\r' | awk 'tolower($1)=="location:"{print $2}')"
  [ -n "$location" ] || fail "o tusd não devolveu Location (pre-create rejeitado?)"
  mkdir -p "$WORK/parts.$id"
  split -b "$CHUNK" "$file" "$WORK/parts.$id/p."
  for part in "$WORK/parts.$id"/p.*; do
    new="$(curl -fsS --max-time 120 -D - -o /dev/null -X PATCH "$location" -H 'Tus-Resumable: 1.0.0' \
      -H 'Content-Type: application/offset+octet-stream' -H "Upload-Offset: $offset" \
      --data-binary "@$part" | tr -d '\r' | awk 'tolower($1)=="upload-offset:"{print $2}')"
    offset=$((offset + $(stat -c %s "$part")))
    [ "$new" = "$offset" ] || fail "offset devolvido $new, esperado $offset"
  done
  [ "$offset" = "$total" ] || fail "enviou $offset de $total bytes"
}

# forged_hook <import_id> <bucket> <query> -> código HTTP do POST /hooks/tus
forged_hook() {
  local id="$1" bucket="$2" query="$3" body
  body="$(jq -n --arg id "$id" --arg b "$bucket" \
    '{Type:"post-finish",Event:{Upload:{ID:"forjado+x",Size:1,MetaData:{import_id:$id},Storage:{Type:"s3store",Bucket:$b,Key:"forjado"}}}}')"
  curl -sS --max-time 15 -o /dev/null -w '%{http_code}' -X POST "$API_URL/hooks/tus$query" \
    -H 'Content-Type: application/json' -d "$body"
}

expect_waiting() { [ "$(import_json "$1" | jq -r .status)" = "aguardando_upload" ] || fail "o lote saiu de aguardando_upload: $2"; }

echo "A. hooks forjados são recusados e não mexem no lote"
head -c "$((2 * CHUNK))" /dev/urandom > "$WORK/random.bin"
RID="$(create_import "$WORK/random.bin" export.zip)"
[ "$(forged_hook "$RID" "$BUCKET" '')" = 403 ] || fail "hook sem token não deu 403"
expect_waiting "$RID" "hook sem token"
[ "$(forged_hook "$RID" "$BUCKET" '?token=errado')" = 403 ] || fail "hook com token errado não deu 403"
expect_waiting "$RID" "hook com token errado"
[ "$(forged_hook "$RID" bucket-alheio "?token=$HOOK_SECRET")" = 403 ] || fail "hook com bucket alheio não foi recusado (403)"
expect_waiting "$RID" "hook com bucket alheio"
echo "   sem token, token errado e bucket alheio: 403, lote intacto"

echo "B. upload multipart de bytes até o bucket"
SHA="$(sha256sum "$WORK/random.bin" | cut -d' ' -f1)"; TOTAL="$(stat -c %s "$WORK/random.bin")"
tus_upload "$RID" "$WORK/random.bin"
STATUS=aguardando_upload
for _ in $(seq 1 60); do
  RESP="$(import_json "$RID")"; STATUS="$(jq -r .status <<<"$RESP")"
  [ "$STATUS" != aguardando_upload ] && break
  sleep 2
done
echo "   status=$STATUS"
case "$STATUS" in
  recebido|processando) ;;
  falhou) # bytes aleatórios não são um zip: o worker pode recusar, mas a API não pode ter recusado o sha256
    if jq -r '.motivo_falha // ""' <<<"$RESP" | grep -qi sha256; then fail "a API recusou o objeto por sha256: $(jq -r .motivo_falha <<<"$RESP")"; fi ;;
  *) fail "estado inesperado após o upload de bytes: $STATUS" ;;
esac
KEY="$(awslocal s3api list-objects-v2 --bucket "$BUCKET" --query 'Contents[].[Key,Size]' --output text \
  | tr -d '\r' | awk -v size="$TOTAL" '$2==size && $1 !~ /\.info$/ {print $1; exit}')"
[ -n "$KEY" ] || fail "nenhum objeto de $TOTAL bytes no bucket $BUCKET"
REMOTE_SHA="$("${DC[@]}" exec -T localstack sh -c \
  "awslocal s3api get-object --bucket '$BUCKET' --key '$KEY' /tmp/e2e.bin >/dev/null && sha256sum /tmp/e2e.bin | cut -d' ' -f1" | tr -d '\r')"
[ "$REMOTE_SHA" = "$SHA" ] || fail "sha256 do objeto no bucket ($REMOTE_SHA) difere do enviado ($SHA)"
echo "   chave=$KEY, tamanho e sha256 conferem"

echo "C. export reduzido: do upload ao status concluido (API, chave, S3, evento e worker)"
ZID="$(create_import services/ingest/testdata/apple/export-reduzido.zip export.zip)"
tus_upload "$ZID" services/ingest/testdata/apple/export-reduzido.zip
DEADLINE=$((SECONDS + DONE_TIMEOUT_S))
STATUS=""
while [ "$SECONDS" -lt "$DEADLINE" ]; do
  RESP="$(import_json "$ZID")"; STATUS="$(jq -r .status <<<"$RESP")"
  [ "$STATUS" = concluido ] && break
  [ "$STATUS" = falhou ] && fail "o lote do export reduzido falhou: $(jq -c '{motivo_falha,erros}' <<<"$RESP")"
  sleep 3
done
[ "$STATUS" = concluido ] || fail "o export reduzido não chegou a concluido em ${DONE_TIMEOUT_S}s (último status: $STATUS)"
IMPORTED="$(jq -r .importadas <<<"$RESP")"
echo "   $(jq -c '{status,importadas,duplicadas,erros}' <<<"$RESP")"
[ "$IMPORTED" -gt 0 ] || fail "o worker concluiu sem importar nenhuma atividade"

echo "D. a regra de expiração está gravada com $RETENTION dias"
LIFECYCLE="$(awslocal s3api get-bucket-lifecycle-configuration --bucket "$BUCKET" | tr -d '\r')"
grep -Eq "\"Days\": ${RETENTION}([^0-9]|\$)" <<<"$LIFECYCLE" || fail "regra de expiração ausente ou com prazo diferente: $LIFECYCLE"
grep -q '"Enabled"' <<<"$LIFECYCLE" || fail "regra de expiração não está Enabled: $LIFECYCLE"
echo "OK: hooks forjados recusados, multipart de bytes até o bucket, export reduzido concluido e expiração de $RETENTION dias"
