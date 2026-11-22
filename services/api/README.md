# services/api

API de importação (Hyperf 3, PHP 8.4, hexagonal). Cobre M1-04 (criar e consultar importação), M1-05 (hooks do tusd e outbox) e o lado da API do M1-11 (tipos zip, fit, tcx, gpx).

| Rota | Função |
| --- | --- |
| `POST /imports` | Valida metadados, sha256 e consentimento LGPD; cria o lote; devolve `import_id` e `upload_url` |
| `GET /imports/{id}` | Status do lote (só do atleta dono) |
| `POST /hooks/tusd?token=...` | Hooks `pre-create` (autoriza o upload) e `post-finish` (confere sha256, grava `ImportacaoRecebida` ou `ImportacaoFalhou` na outbox) |

Autenticação: JWT RS256 validado pela API (ADR 0009). Comando `outbox:relay` publica a outbox no RabbitMQ com confirmação.

Flags do tusd no compose: `-hooks-http=http://api:9501/hooks/tusd?token=$TUSD_HOOK_SECRET -hooks-enabled-events=pre-create,post-finish -hooks-http-forward-headers=Authorization -s3-bucket=imports -s3-endpoint=http://minio:9000`. O cliente tus envia o metadado `import_id` e o cabeçalho `Authorization`.

Testes: `composer install --ignore-platform-req=ext-swoole && composer test` (unitários, sem Docker). `composer stan` roda PHPStan nível 6.
