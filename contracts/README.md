# contracts

Fonte da verdade dos contratos entre serviços. Mudar um contrato aqui exige atualizar quem publica e quem consome no mesmo PR.

| Caminho | Conteúdo |
| --- | --- |
| `events/importacao/*.v1.schema.json` | JSON Schema (2020-12) dos eventos de Importação |
| `events/importacao/examples/*.json` | Exemplo válido de cada evento (o CI valida contra o esquema) |
| `events/atividades/*.v1.schema.json` | `AtividadeCorrigida` (envelope sem `import_id`; publicado na exchange `importacao`) |
| `events/metricas/*.v1.schema.json` | `MetricasCalculadas` (exchange `metricas`, envelope sem `import_id`) |
| `events/planejamento/*.v1.schema.json` | `PlanoGerado`, `TreinoPlanejado`, `PlanoAjustado` (exchange `planejamento`) |
| `events/atleta/*.v1.schema.json` | `VinculoTreinadorConsentido`, `VinculoTreinadorRevogado` (exchange `atleta`) |
| `events/metricas/limiar-atualizado.v1.schema.json` | `LimiarAtualizado` (exchange `metricas`) |
| `openapi/coach-ports.yaml` | Subconjunto de métricas e planejamento consumido pelo Coach IA, com a rota `/ajustes/validacao`, ainda sem implementação no Planejamento (ADR 0021) |
| `asyncapi.yaml` | Canais e operações dos eventos no RabbitMQ |
| `openapi/imports.yaml` | API HTTP de importação (`POST /imports`, `GET /imports/{id}`, hook do tusd) |
| `openapi/planejamento.yaml` | API de Planejamento: planos, ajustes, aprovação, aderência, export .fit, vínculos (porta 8120) |
| `openapi/metricas.yaml` | Leitura de carga e atividades; escrita de perfil e limiares (porta 8121) |
| `openapi/atividades.yaml` | Amostras (interno) e correção de atividade, no binário do worker de ingestão (porta 8122) |

## Envelope dos eventos

Todo evento é um JSON com `event_id` (uuid), `tipo`, `versao`, `ocorrido_em` (UTC), `dados` (payload do tipo) e, nos eventos de Importação, `import_id`. Os esquemas são autocontidos (sem `$ref`) para facilitar validação em PHP e Go.

Propriedades AMQP da mensagem: `content_type: application/json`, `delivery_mode: 2` (persistente), `message_id = event_id`, `type = tipo`.

## Topologia (definida em `infra/rabbitmq/definitions.json`)

| Evento | Exchange | Routing key | Publicado por | Fila consumidora |
| --- | --- | --- | --- | --- |
| ImportacaoRecebida | `importacao` (topic) | `importacao.recebida` | API | `ingest.importacao-recebida` |
| TreinoImportado | `importacao` | `treino.importado` | worker | `metrics.treino-importado` |
| ImportacaoConcluida | `importacao` | `importacao.concluida` | worker | `api.importacao-resultado` |
| ImportacaoFalhou | `importacao` | `importacao.falhou` | API ou worker | `api.importacao-resultado` |
| AtividadeCorrigida | `importacao` | `atividade.corrigida` | worker de ingestão (`PATCH /atividades/{id}`) | `metrics.atividade-corrigida` |
| MetricasCalculadas | `metricas` (topic) | `metricas.calculadas` | serviço de métricas | `planning.metricas-calculadas` |
| LimiarAtualizado | `metricas` | `limiar.atualizado` | serviço de métricas | `planning.limiar-atualizado` |
| PlanoGerado | `planejamento` (topic) | `plano.gerado` | Planejamento | (nenhuma ainda) |
| TreinoPlanejado | `planejamento` | `treino.planejado` | Planejamento | (M5: sincronização) |
| PlanoAjustado | `planejamento` | `plano.ajustado` | Planejamento | (nenhuma ainda) |
| VinculoTreinadorConsentido | `atleta` (topic) | `vinculo.consentido` | Planejamento | `metrics.vinculo` |
| VinculoTreinadorRevogado | `atleta` | `vinculo.revogado` | Planejamento | `metrics.vinculo` |

Filas são quorum, duráveis, com `x-dead-letter-exchange=importacao.dlx`. Consumidor que não consegue processar rejeita sem requeue (`basic.nack`, `requeue=false`) e a mensagem cai em `importacao.dlq`.

## Versionamento

- Campo novo opcional: mesma versão, o esquema ganha a propriedade.
- Mudança incompatível: novo arquivo `*.v2.schema.json`; o publicador emite as duas versões durante a migração.
- Consumidores rejeitam (DLQ) versões desconhecidas em vez de adivinhar.

## Validação local

```bash
make contracts-check
```
