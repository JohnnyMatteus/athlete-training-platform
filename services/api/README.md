# services/api

API de importação (Hyperf 3, PHP 8.4, hexagonal). Cobre M1-04 (criar e consultar importação), M1-05 (hooks do tusd e outbox) e o lado da API do M1-11 (tipos zip, fit, tcx, gpx).

| Rota | Função |
| --- | --- |
| `POST /imports` | Valida metadados, sha256 e consentimento LGPD; cria o lote; devolve `import_id` e `upload_url` |
| `GET /imports/{id}` | Status do lote (só do atleta dono) |
| `POST /hooks/tus` | Hooks `pre-create` (autoriza o upload) e `post-finish` (confere sha256, grava `ImportacaoRecebida` ou `ImportacaoFalhou` na outbox) |

Autenticação: JWT RS256 validado pela API (ADR 0011). Comando `outbox:relay` publica a outbox no RabbitMQ com confirmação.

Contrato: `contracts/openapi/imports.yaml` e `contracts/events/importacao/` (um teste valida os eventos da API contra os esquemas). A API também consome `api.importacao-resultado` (`php bin/hyperf.php importacao:consume-results`) para atualizar o lote; `outbox:relay` publica a outbox.

Hook do tusd: o cliente envia `import_id` em `Upload-Metadata` e `Authorization: Bearer <jwt>`; o tusd precisa de `-hooks-http-forward-headers=Authorization` e `-hooks-enabled-events=pre-create,post-finish`. `object_key` do evento é o ID do upload. `TUSD_HOOK_SECRET` (opcional) exige `?token=` na URL do hook.

Pendências no contrato/migrações (fundação): coluna `consentimento jsonb NOT NULL DEFAULT '{}'` em `lote_importacao`; `consentimento` no corpo do `POST /imports` do OpenAPI e autenticação JWT no lugar de `ATHLETE_ID`.

Testes: `composer install --ignore-platform-req=ext-swoole && composer test` (unitários, sem Docker). `composer lint` roda php-cs-fixer e PHPStan nível 6.
