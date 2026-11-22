# 0012. Banco `metricas` é dono de `carga_diaria`

- **Status:** Proposto
- **Data:** 2026-11-22

## Contexto

O design doc (5.2) lista `carga_diaria` junto de `atividade` e `amostra`, mas a ADR 0009 define um banco por contexto e a ADR 0001 proíbe um contexto de ler a tabela de outro. O serviço de métricas calcula TSS, CTL, ATL e TSB e precisa de limiares e perfil do atleta, que também não pertencem a Atividades.

## Decisão

- Terceiro banco no mesmo servidor, `metricas`, com migrações em `infra/db/migrations/metricas/`. Ele é dono de `carga_diaria`, `carga_atividade` (carga, método e descartes por atividade), `limiar` (FTP, pace de limiar e CSS versionados por `vigente_desde`), `perfil_atleta`, `outbox` e `evento_processado` (inbox).
- O serviço consome `TreinoImportado` e `AtividadeCorrigida`, grava as tabelas numa transação com a inbox e o outbox, e publica `MetricasCalculadas` na exchange `metricas`.
- O worker de ingestão e a API nunca escrevem nem leem o banco `metricas`; quem precisar de carga consome `MetricasCalculadas`.
- Dia da carga é o dia UTC do início da atividade.

## Amostras (atualizado pelo ADR 0015)

A exceção original (leitura somente leitura de `atividades.amostra`) foi substituída: o contexto de Atividades expõe as amostras por um endpoint interno paginado, e o serviço de métricas as lê por um cliente HTTP atrás da porta `SampleSource`. Nenhum serviço lê o banco de outro; `ATIVIDADES_READONLY_URL` deixa de existir.

## Consequências

- A fronteira de escrita e a de leitura ficam impostas pela infraestrutura.
- Limiares e perfil têm origem: `PUT /metricas/limiares` e `PUT /metricas/perfil` (contrato `openapi/metricas.yaml`) gravam nas tabelas deste banco e publicam `LimiarAtualizado`. Sem limiar, a atividade cai no TRIMP ou fica `indisponivel`.
- `AtividadeCorrigida` é publicado pelo worker de ingestão (`PATCH /atividades/{id}`, ADR 0015).

## Alternativas consideradas

- `carga_diaria` no banco `atividades`: mais simples, mas o serviço de métricas passaria a escrever no banco do worker e inverteria a regra da ADR 0009.
- Amostras no evento: payload de milhares de pontos por treino, inviável no RabbitMQ.
- Réplica das amostras no banco `metricas`: duplica o maior volume de dados do sistema por uma leitura por atividade.
