# 0010. Servidor tus separado (tusd) com backend S3/MinIO

- **Status:** Proposto
- **Data:** 2026-11-21

## Contexto

O upload do export.zip (centenas de MB a GB) precisa ser resumível (design doc 3). Ponto em aberto 3: servidor tus separado ou dentro do Hyperf.

## Decisão

Usar o `tusd` oficial com backend S3 apontando para o MinIO. A API participa por hooks HTTP: `pre-create` valida o lote e `post-finish` confere o sha256 e grava `ImportacaoRecebida` na outbox (contrato em `contracts/openapi/imports.yaml`).

## Consequências

- Nenhum byte do arquivo passa pela API; Swoole não segura conexões longas de upload.
- Mais um container e um contrato de hook a manter.
- Troca de MinIO por S3 real é só configuração.

## Alternativas consideradas

- Implementar tus no Hyperf: código próprio para um protocolo com servidor de referência maduro.
- Upload multipart com presigned URL: não retoma de onde parou sem lógica no cliente.
