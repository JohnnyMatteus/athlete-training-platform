#!/usr/bin/env bash
# Prova do S3 local (LocalStack) pelo compose real: upload resumível pelo tusd (multipart, dois PATCH de 6 MiB),
# o objeto chega ao bucket com o sha256 declarado, a API aceita o lote e a regra de expiração está gravada
# com o prazo certo. Pré-requisitos: `make dev-keys` feito e os serviços api, tusd, localstack, postgres,
# rabbitmq no ar com as migrações aplicadas. Qualquer divergência encerra com erro.
# Variáveis: API_URL, TUS_URL, UPLOADS_BUCKET, UPLOADS_RETENTION_DAYS, ENV_FILE.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"
API_URL="${API_URL:-http://127.0.0.1:9501}"
TUS_URL="${TUS_URL:-http://127.0.0.1:8080/files/}"
BUCKET="${UPLOADS_BUCKET:-uploads}"
RETENTION="${UPLOADS_RETENTION_DAYS:-7}"
DC=(docker compose --profile core --env-file "${ENV_FILE:-.env.example}")
CHUNK=$((6 * 1024 * 1024))   # >= 5 MiB, o mínimo de uma parte do multipart do S3
TOTAL=$((2 * CHUNK))
WORK="$(mktemp -d)"
trap 'rm -rf "$WORK"' EXIT

fail() { echo "FALHA: $*" >&2; exit 1; }
s3() { "${DC[@]}" exec -T localstack awslocal "$@"; }

head -c "$TOTAL" /dev/urandom > "$WORK/file.bin"
SHA="$(sha256sum "$WORK/file.bin" | cut -d' ' -f1)"
TOKEN="$(infra/scripts/dev-jwt.sh token)"

echo "1. cria a importação (sha256 ${SHA:0:12}..., $TOTAL bytes)"
BODY="$(jq -n --arg sha "$SHA" --argjson size "$TOTAL" \
  '{nome_arquivo:"export.zip",tamanho_bytes:$size,tipo_arquivo:"zip",sha256:$sha,consentimento:{aceito:true,versao_termo:"2026-11",categorias:["treinos"]}}')"
IMPORT_ID="$(curl -fsS --max-time 30 -X POST "$API_URL/imports" -H "Authorization: Bearer $TOKEN" \
  -H 'Content-Type: application/json' -d "$BODY" | jq -er .import_id)"
echo "   import_id=$IMPORT_ID"

echo "2. cria o upload no tusd (o pre-create chama a API)"
META="import_id $(printf '%s' "$IMPORT_ID" | base64 | tr -d '\n')"
LOCATION="$(curl -fsS --max-time 30 -D - -o /dev/null -X POST "$TUS_URL" -H 'Tus-Resumable: 1.0.0' \
  -H "Upload-Length: $TOTAL" -H "Upload-Metadata: $META" -H "Authorization: Bearer $TOKEN" \
  | tr -d '\r' | awk 'tolower($1)=="location:"{print $2}')"
[ -n "$LOCATION" ] || fail "o tusd não devolveu Location (pre-create rejeitado?)"
echo "   upload=$LOCATION"

echo "3. envia em duas partes (PATCH)"
split -b "$CHUNK" "$WORK/file.bin" "$WORK/part."
OFFSET=0
for part in "$WORK"/part.*; do
  NEW="$(curl -fsS --max-time 120 -D - -o /dev/null -X PATCH "$LOCATION" -H 'Tus-Resumable: 1.0.0' \
    -H 'Content-Type: application/offset+octet-stream' -H "Upload-Offset: $OFFSET" \
    --data-binary "@$part" | tr -d '\r' | awk 'tolower($1)=="upload-offset:"{print $2}')"
  OFFSET=$((OFFSET + $(stat -c %s "$part")))
  [ "$NEW" = "$OFFSET" ] || fail "offset devolvido $NEW, esperado $OFFSET"
done
[ "$OFFSET" = "$TOTAL" ] || fail "enviou $OFFSET de $TOTAL bytes"

echo "4. a API confere o sha256 no hook post-finish e tira o lote de aguardando_upload"
STATUS=""
for _ in $(seq 1 60); do
  RESP="$(curl -fsS --max-time 15 "$API_URL/imports/$IMPORT_ID" -H "Authorization: Bearer $TOKEN")"
  STATUS="$(jq -r .status <<<"$RESP")"
  [ "$STATUS" != "aguardando_upload" ] && break
  sleep 2
done
echo "   status=$STATUS"
[ "$STATUS" != "aguardando_upload" ] || fail "o lote continuou em aguardando_upload (hook post-finish não confirmou)"
if jq -r '.motivo_falha // ""' <<<"$RESP" | grep -qi 'sha256'; then
  fail "a API recusou o objeto por sha256 divergente: $(jq -r .motivo_falha <<<"$RESP")"
fi

echo "5. o objeto está no bucket com o tamanho e o sha256 enviados"
KEY="$(s3 s3api list-objects-v2 --bucket "$BUCKET" --query 'Contents[].[Key,Size]' --output text \
  | tr -d '\r' | awk -v size="$TOTAL" '$2==size && $1 !~ /\.info$/ {print $1; exit}')"
[ -n "$KEY" ] || fail "nenhum objeto de $TOTAL bytes no bucket $BUCKET"
REMOTE_SHA="$("${DC[@]}" exec -T localstack sh -c \
  "awslocal s3api get-object --bucket '$BUCKET' --key '$KEY' /tmp/e2e.bin >/dev/null && sha256sum /tmp/e2e.bin | cut -d' ' -f1" | tr -d '\r')"
[ "$REMOTE_SHA" = "$SHA" ] || fail "sha256 do objeto no bucket ($REMOTE_SHA) difere do enviado ($SHA)"
echo "   chave=$KEY sha256 confere"

echo "6. a regra de expiração está gravada com $RETENTION dias"
LIFECYCLE="$(s3 s3api get-bucket-lifecycle-configuration --bucket "$BUCKET" | tr -d '\r')"
grep -Eq "\"Days\": ${RETENTION}([^0-9]|\$)" <<<"$LIFECYCLE" || fail "regra de expiração ausente ou com prazo diferente: $LIFECYCLE"
grep -q '"Enabled"' <<<"$LIFECYCLE" || fail "regra de expiração não está Enabled: $LIFECYCLE"
echo "OK: upload multipart pelo tusd até o bucket e expiração de $RETENTION dias"
