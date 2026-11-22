# 0009. Contratos versionados em `contracts/` e um banco por contexto

- **Status:** Proposto
- **Data:** 2026-11-21

## Contexto

API (PHP) e worker (Go) trocam eventos e precisam concordar no formato (ponto em aberto 4 das issues do M1). A ADR 0001 proíbe que um contexto leia a tabela de outro, e o ponto em aberto 1 pergunta quem é dono das tabelas de Atividades.

## Decisão

- Eventos descritos em JSON Schema 2020-12 autocontido (`contracts/events/`), com AsyncAPI para topologia e OpenAPI 3.1 para HTTP. O CI valida esquemas, exemplos e especificações.
- Dois bancos no mesmo servidor Postgres: `importacao` (lote e outbox da API) e `atividades` (atividade, amostra, outbox e inbox do worker). Migrações com golang-migrate em `infra/db/migrations/<banco>/`.
- A API atualiza o lote consumindo `ImportacaoConcluida` e `ImportacaoFalhou`; o worker nunca escreve no banco `importacao`.

## Consequências

- A fronteira entre contextos é imposta pela infraestrutura, não só por convenção.
- Status do lote é eventualmente consistente com o processamento.
- Um servidor só em desenvolvimento; separar fisicamente depois não muda código.

## Alternativas consideradas

- Um banco com schemas separados: mais simples, mas permite consulta cruzada por engano.
- Protobuf/Avro com registry: mais rigoroso, peso desproporcional ao M1.
- Worker escrevendo direto no lote: acopla os contextos.
