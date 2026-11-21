# 0005. Motor de periodização determinístico; a IA só propõe, o validador decide (A-05)

- **Status:** Proposto
- **Data:** 2026-11-21
- **Id no design doc:** A-05 (seção 13; detalhe na seção 7)

## Contexto

Um plano de treino inadequado pode ser arriscado para o atleta (seção 15). O sistema não substitui treinador nem orientação médica (seção 2). Uma IA generativa que inventa carga não é verificável.

## Decisão

O plano é gerado por um motor de periodização determinístico (regras verificáveis e testáveis, seção 7.2). A IA (Coach, seção 7.4) interpreta metas, explica sessões e propõe ajustes, mas não tem ferramenta de escrita direta: `propose_adjustment` devolve uma proposta que passa pelo validador de restrições (código, não prompt; seção 7.3) e, se o modo treinador estiver ativo, pela aprovação do treinador. Menção a dor, lesão ou sintoma bloqueia ajustes de intensidade e gera resposta padrão de procurar profissional.

## Alternativas consideradas

- O design doc não registra alternativas para esta decisão. A preencher pelo autor antes de aceitar.

## Consequências

- Toda carga planejada é auditável e reproduzível; as regras do validador têm testes de unidade (seção 11).
- Os evals do Coach medem a % de propostas aceitas pelo validador, 0 conselhos médicos e recusa correta quando há dor (seção 11).
- O motor e o validador concentram muita regra de domínio; o validador é núcleo do autor (modo par, seção 14).
