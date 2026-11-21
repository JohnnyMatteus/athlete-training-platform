# 0001. Seis contextos delimitados integrados por eventos (A-01)

- **Status:** Proposto
- **Data:** 2026-11-21
- **Id no design doc:** A-01 (seção 13; detalhe na seção 4.1)

## Contexto

A plataforma junta importação de arquivos e do HealthKit, um modelo canônico de atividades, cálculo de métricas, perfil do atleta, planejamento e envio ao relógio. São responsabilidades com ritmos e linguagens diferentes (Hyperf, Go, TypeScript, Swift; ADR 0007) e precisam evoluir sem que uma mudança de esquema numa parte quebre as outras.

## Decisão

O sistema é dividido em seis contextos delimitados: Importação, Atividades, Métricas, Atleta, Planejamento e Sincronização, cada um com seus agregados e eventos publicados (tabela da seção 4.1). A integração entre contextos acontece só por eventos (outbox → RabbitMQ) ou por consulta a read models. Nenhum contexto lê a tabela de outro.

## Alternativas consideradas

- O design doc não registra alternativas para esta decisão. A preencher pelo autor antes de aceitar.

## Consequências

- Cada contexto pode mudar o próprio esquema sem coordenar com os outros, desde que mantenha os eventos.
- Correções antigas se propagam por evento (ex.: `AtividadeCorrigida` dispara recálculo de carga, seção 6).
- Exige outbox em cada contexto que publica, consumidores idempotentes e read models para consultas entre contextos.
- Mais infraestrutura (RabbitMQ) e consistência eventual entre contextos.
