#!/bin/sh
# Cria o bucket de uploads e a regra de expiração de UPLOADS_RETENTION_DAYS dias (ADR 0019 e ADR 0035).
# O LocalStack executa os scripts de /etc/localstack/init/ready.d quando está pronto; o healthcheck do compose
# só fica saudável depois deste script. Idempotente. Variáveis: UPLOADS_BUCKET e UPLOADS_RETENTION_DAYS.
set -eu

: "${UPLOADS_BUCKET:?defina UPLOADS_BUCKET}"
: "${UPLOADS_RETENTION_DAYS:?defina UPLOADS_RETENTION_DAYS}"

if ! awslocal s3api head-bucket --bucket "$UPLOADS_BUCKET" 2>/dev/null; then
  awslocal s3api create-bucket --bucket "$UPLOADS_BUCKET"
fi
# A regra vale para qualquer objeto do bucket (prefixo vazio), como a regra anterior (expire-days no bucket todo).
awslocal s3api put-bucket-lifecycle-configuration --bucket "$UPLOADS_BUCKET" \
  --lifecycle-configuration "{\"Rules\":[{\"ID\":\"expira-uploads\",\"Status\":\"Enabled\",\"Filter\":{\"Prefix\":\"\"},\"Expiration\":{\"Days\":${UPLOADS_RETENTION_DAYS}}}]}"

# Confere o que foi aplicado: se a regra não ficar gravada com o prazo certo, o init falha em vez de fingir.
awslocal s3api get-bucket-lifecycle-configuration --bucket "$UPLOADS_BUCKET" | grep -Eq "\"Days\": ${UPLOADS_RETENTION_DAYS}([^0-9]|\$)"
echo "bucket $UPLOADS_BUCKET pronto, com expiração de ${UPLOADS_RETENTION_DAYS} dias"
