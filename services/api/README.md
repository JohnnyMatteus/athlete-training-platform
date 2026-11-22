# services/api

API de importação (Hyperf 3, PHP 8.4, hexagonal). Cobre M1-04 (criar e consultar importação), M1-05 (hooks do tusd e outbox) e o lado da API do M1-11 (tipos zip, fit, tcx, gpx).

| Rota | Função |
| --- | --- |
| `POST /imports` | Valida metadados, sha256 e consentimento LGPD; cria o lote; devolve `import_id` e `upload_url` |
| `GET /imports/{id}` | Status do lote (só do atleta dono) |
| `POST /hooks/tus` | Hooks `pre-create` (autoriza o upload) e `post-finish` (confere sha256, grava `ImportacaoRecebida` ou `ImportacaoFalhou` na outbox) |

Autenticação: JWT RS256 validado pela API (ADR 0011). Comando `outbox:relay` publica a outbox no RabbitMQ com confirmação.

Contrato: `contracts/openapi/imports.yaml` e `contracts/events/importacao/` (um teste valida os eventos da API contra os esquemas). A API também consome `api.importacao-resultado` (`php bin/hyperf.php importacao:consume-results`) para atualizar o lote; `outbox:relay` publica a outbox.

Hook do tusd: o cliente envia `import_id` em `Upload-Metadata` e `Authorization: Bearer <jwt>`; o tusd precisa de `-hooks-http-forward-headers=Authorization` e `-hooks-enabled-events=pre-create,post-finish`. `object_key` do evento é a chave do objeto no bucket (`Storage.Key` do hook; o `Upload.ID` do tusd leva o sufixo do multipart e não serve de chave). `TUSD_HOOK_SECRET` é obrigatório: o hook só é aceito com `?token=<valor>` na URL (comparação em tempo constante); com o segredo vazio a API recusa todos os hooks com 403. O `post-finish` só aceita o bucket `UPLOADS_BUCKET`; outro bucket é recusado sem ler o objeto e sem alterar o lote. A API (9501) e o tusd (8080) ficam publicados só em `127.0.0.1` no compose.

Contrato e migrações: coluna `consentimento` em `lote_importacao` (migração 000003) e `consentimento` obrigatório no `POST /imports`; autenticação JWT no lugar do `ATHLETE_ID` (ADR 0011).

Testes: `composer install --ignore-platform-req=ext-swoole && composer test` (unitários, sem Docker). `composer lint` roda php-cs-fixer e PHPStan nível 6.

Observabilidade (ADR 0018): `TraceMiddleware` abre o span de cada requisição, a outbox grava o `traceparent` (coluna nova, migração 000004) e o relay o publica como cabeçalho AMQP; o hook `post-finish` do tusd retoma o trace guardado no lote. `OTEL_EXPORTER_OTLP_ENDPOINT` liga a exportação. Workers (`outbox:relay`, `importacao:consume-results`) rodam como serviços próprios no compose, com `worker:health` como healthcheck (banco, broker e atraso da outbox; `WORKER_HEALTH_MAX_LAG`).
