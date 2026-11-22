# 0017. Aderência é calculada em Planejamento a partir de MetricasCalculadas

- **Status:** Proposto
- **Data:** 2026-11-22
- **Id no design doc:** seção 6 (métrica "Aderência")

## Contexto

Aderência é TSS realizado / TSS planejado por sessão e semana. O plano mora em Planejamento; o realizado, em Métricas. Nenhum contexto lê o banco do outro.

## Decisão

- Planejamento consome `MetricasCalculadas` e guarda `tss_realizado` por atividade e dia (read model, inbox por `event_id`), usando os campos opcionais `dia` e `tipo_atividade` do evento. O mesmo evento atualiza o CTL/ATL atual do atleta.
- Sessão: o treino planejado casa com as atividades do mesmo dia e da mesma modalidade; o realizado é a soma. Atividade sem treino planejado entra só no total da semana. Aderência nula quando o planejado é zero.
- Caso de uso `CalculateAdherence` e `GET /planos/{id}/aderencia`.
- Correção de atividade reenvia `MetricasCalculadas`; o read model faz upsert por `atividade_id`.

## Alternativas consideradas

- Calcular em Métricas com o plano replicado: Métricas passaria a conhecer o plano.
- Consulta síncrona entre serviços: acopla disponibilidade.

## Consequências

- Eventos antigos sem `dia` (anteriores a esta versão) não entram na aderência.
