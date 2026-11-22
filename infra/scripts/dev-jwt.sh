#!/usr/bin/env bash
# Par de chaves RS256 de desenvolvimento e emissor de tokens de teste (ADR 0011 e 0014).
# Uso:
#   infra/scripts/dev-jwt.sh keys                      cria o par em services/api/runtime (ignorado pelo Git)
#   infra/scripts/dev-jwt.sh token [sub] [papeis]      imprime um JWT; papeis separados por vírgula (padrão: atleta)
# Exemplos:
#   TOKEN=$(infra/scripts/dev-jwt.sh token)
#   infra/scripts/dev-jwt.sh token e5f6a7b8-c9d0-4e1f-8a2b-3c4d5e6f7a8b treinador
# Variáveis: JWT_ISSUER, JWT_AUDIENCE, JWT_TTL (segundos, padrão 3600), JWT_KEY_DIR.
# Só para desenvolvimento: a chave privada nunca sai desta máquina e não deve ser reutilizada.
set -euo pipefail

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
DIR="${JWT_KEY_DIR:-$ROOT/services/api/runtime}"
ISS="${JWT_ISSUER:-https://idp.example.test/}"
AUD="${JWT_AUDIENCE:-athlete-api}"
TTL="${JWT_TTL:-3600}"

b64url() { openssl base64 -A | tr '+/' '-_' | tr -d '='; }

keys() {
  mkdir -p "$DIR"
  if [ -f "$DIR/jwt_private.pem" ]; then
    echo "par já existe em $DIR (apague os dois arquivos para gerar outro)" >&2
    return 0
  fi
  umask 077
  openssl genpkey -algorithm RSA -pkeyopt rsa_keygen_bits:2048 -out "$DIR/jwt_private.pem" 2>/dev/null
  openssl pkey -in "$DIR/jwt_private.pem" -pubout -out "$DIR/jwt_public.pem"
  chmod 644 "$DIR/jwt_public.pem"
  echo "chaves criadas em $DIR" >&2
}

token() {
  local sub="${1:-2f0f0a2e-6a53-4c3a-9d8e-1b7e6c5d4a01}" roles="${2:-atleta}"
  [ -f "$DIR/jwt_private.pem" ] || { echo "sem chave: rode '$0 keys' antes" >&2; exit 1; }
  local now exp roles_json header payload sig
  now=$(date +%s); exp=$((now + TTL))
  roles_json=$(printf '%s' "$roles" | awk -F, '{for(i=1;i<=NF;i++){printf "%s\"%s\"", (i>1?",":""), $i}}')
  header=$(printf '{"alg":"RS256","typ":"JWT"}' | b64url)
  payload=$(printf '{"iss":"%s","aud":"%s","sub":"%s","roles":[%s],"iat":%s,"exp":%s}' "$ISS" "$AUD" "$sub" "$roles_json" "$now" "$exp" | b64url)
  sig=$(printf '%s.%s' "$header" "$payload" | openssl dgst -sha256 -sign "$DIR/jwt_private.pem" | b64url)
  printf '%s.%s.%s\n' "$header" "$payload" "$sig"
}

case "${1:-}" in
  keys) keys ;;
  token) shift; token "$@" ;;
  *) sed -n '2,10p' "$0" >&2; exit 2 ;;
esac
