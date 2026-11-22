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

## Exceção registrada: leitura das amostras

`TreinoImportado` traz só o resumo da atividade, e NP, pace normalizado e descartes exigem as amostras (hypertable `amostra`, banco `atividades`). Sem endpoint nem evento com as amostras, o serviço lê `atividades.amostra` por uma porta (`SampleSource`) com um adaptador somente leitura, usando um usuário próprio de apenas `SELECT` (`ATIVIDADES_READONLY_URL`). Esta é a única leitura cruzada e vive num adaptador isolado.

Quando o contexto de Atividades expuser as amostras (endpoint interno paginado ou URI de objeto no evento), troca-se o adaptador sem tocar em domínio ou caso de uso.

## Consequências

- A fronteira de escrita fica imposta pela infraestrutura; a de leitura tem uma exceção rastreável.
- Limiares e perfil ainda não têm origem: a API de Planejamento os alimentará (M4); até lá entram por migração, seed ou SQL. Sem limiar, a atividade cai no TRIMP ou fica `indisponivel`.
- `AtividadeCorrigida` ainda não tem publicador: o contrato existe e o consumidor está pronto.

## Alternativas consideradas

- `carga_diaria` no banco `atividades`: mais simples, mas o serviço de métricas passaria a escrever no banco do worker e inverteria a regra da ADR 0009.
- Amostras no evento: payload de milhares de pontos por treino, inviável no RabbitMQ.
- Réplica das amostras no banco `metricas`: duplica o maior volume de dados do sistema por uma leitura por atividade.
