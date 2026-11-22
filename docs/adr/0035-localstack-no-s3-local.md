# 0035. LocalStack no S3 local

- **Status:** Aceito. Substitui a parte do ADR 0010 que apontava o `tusd` para um servidor S3 específico no ambiente local e a parte do ADR 0019 que aplicava a regra de expiração com o cliente de linha de comando desse servidor. O restante de ambos continua valendo.
- **Data:** 2026-11-22

## Contexto

As imagens do servidor S3 e do cliente de linha de comando do ambiente local saíram dos registros públicos (conferido por manifesto em 22/11/2026). O `make up` deixou de subir em máquina limpa.

## Decisão

- O S3 local passa a ser o `localstack/localstack:4.4.0` com `SERVICES=s3`, porta publicada só em `127.0.0.1`. O `tusd` e a API usam path-style (o `tusd` v2.6.0 liga `UsePathStyle` quando há `-s3-endpoint`, conferido no código-fonte da versão); o worker usa `BucketLookupPath`.
- `infra/localstack/init-s3.sh` (bit executável no Git, montado em `/etc/localstack/init/ready.d`) cria o bucket de uploads e a regra de expiração de `UPLOADS_RETENTION_DAYS` dias e confere o prazo gravado. O healthcheck consulta `/_localstack/init/ready` e só passa com o script `SUCCESSFUL` e nenhum `ERROR`; `tusd`, API e worker esperam por ele.
- As variáveis `MINIO_*` viram `S3_PORT`, `S3_ACCESS_KEY`, `S3_SECRET_KEY`, `S3_HOST` (host:porta do worker) e `S3_USE_SSL`. A porta do console some (o LocalStack community não tem console).
- Correção encontrada ao provar o fluxo: no backend S3 o `Upload.ID` do hook do `tusd` é `<chave>+<id do multipart>` (código-fonte do `s3store` v2.6.0); a API usava o ID como chave do objeto, e a leitura do sha256 e o `object_key` do evento apontariam para uma chave inexistente. A ACL do hook passa a ler `Storage.Key`.
- O CI ganha o job `s3-localstack` (upload real pelo `tusd` até o bucket, sha256 do objeto e regra de expiração).

## Consequências

- O LocalStack community não persiste entre recriações: uploads locais são descartáveis.
- Não é prova de conformidade com a AWS nem da expiração efetiva dos objetos; só a regra gravada com o prazo certo é verificada.
- A imagem é maior do que a do servidor anterior.

## Alternativas consideradas

- **Outra imagem de servidor S3:** não avaliada com manifesto conferido; mantém o risco de nova retirada.
- **S3 real na AWS para o ambiente local:** exige conta e credenciais, o que um clone limpo não tem.
