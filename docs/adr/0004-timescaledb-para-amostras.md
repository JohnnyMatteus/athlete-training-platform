# 0004. TimescaleDB para amostras; Postgres comum para o resto (A-04)

- **Status:** Proposto
- **Data:** 2026-11-21
- **Id no design doc:** A-04 (seção 13; detalhe nas seções 5.1 e 5.2)

## Contexto

Cada atividade tem séries temporais (FC, potência, cadência, velocidade, altitude) com muitas amostras, inseridas em lote e consultadas por intervalo de tempo. O restante do modelo (atividades, carga diária, planos) é relacional comum.

## Decisão

As amostras ficam numa hypertable do TimescaleDB particionada por tempo (`SELECT create_hypertable('amostra', 'ts')`), alimentada por `COPY` em lote. As demais tabelas (`atividade`, `carga_diaria` e as dos outros contextos) ficam em Postgres comum, no mesmo servidor.

## Alternativas consideradas

- O design doc não registra alternativas para esta decisão. A preencher pelo autor antes de aceitar.

## Consequências

- Um único banco para operar, com extensão para séries temporais.
- Inserção em lote e consulta por intervalo eficientes nas amostras.
- Dependência da extensão TimescaleDB nos ambientes e nos testes de integração (Testcontainers com imagem Postgres/Timescale).
