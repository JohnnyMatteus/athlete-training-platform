# 0022. Proposta, validador por porta e aprovação explícita antes de mudar o plano

- **Status:** Proposto
- **Data:** 2026-11-22
- **Detalha:** ADR 0005

## Contexto

A IA só propõe; o validador decide (ADR 0005). O validador é do Planejamento (ADR 0016) e não deve ser duplicado no Coach.

## Decisão

- Saída do modelo validada por esquema Zod (`ModelOutputSchema`); o desfecho precisa caber na intenção pedida (um resumo não pode virar proposta). Mudanças só citam sessões existentes no plano atual (verificado em código) e usam ações fechadas: `move`, `replace_intensity`, `skip`.
- `propose_adjustment` chama o validador pela porta `PlanValidatorPort` (rota de validação sem efeito; falha fechada se indisponível) e, se aceito, grava uma proposta `pending_*` no Redis com validade (padrão 48 h). Se rejeitado, as violações voltam ao modelo; no máximo `COACH_MAX_LLM_CALLS` chamadas (padrão 3).
- Só o `ApprovalService` escreve no plano, por `PlanApplierPort`, e só pelas rotas HTTP `approve`. Nenhuma ferramenta MCP chega a ele. Antes de escrever, o plano é relido e o validador roda de novo; a aprovação é exclusiva (trava no Redis) e uma proposta aplicada não se aplica duas vezes.
- Quem pediu aprova: proposta do atleta, o atleta; proposta do treinador, o treinador. O atleta pode rejeitar qualquer proposta do próprio plano.
- Com modo treinador ligado no plano, o Planejamento devolve o plano a `aguardando_aprovacao` após aplicar o ajuste e o treinador decide ali, pela aprovação que já existe (`/planos/{id}/aprovacao`).

## Alternativas consideradas

- O design doc não registra alternativas. Possível: o Coach aplicar direto e o Planejamento ser o único a validar (sem aprovação explícita, contra o ADR 0005).

## Consequências

- Duas etapas de consentimento em coach mode (aprovar a proposta, depois o treinador aprovar o plano). Decisão do autor se isso é o desejado.
- Proposta expirada some do uso, não do histórico até o TTL do Redis.
