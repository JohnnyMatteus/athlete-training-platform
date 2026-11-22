# 0016. Motor de periodização e validador: modelo e parâmetros

- **Status:** Proposto
- **Data:** 2026-11-22
- **Detalha:** ADR 0005 (design doc 7.2 e 7.3)

## Contexto

O design doc descreve o motor e as regras, sem fixar o modelo de carga planejada nem o algoritmo. Os números abaixo são escolhas do implementador; o autor confirma.

## Decisão

- Função pura: mesmas entradas (meta, disponibilidade, restrições, CTL/ATL iniciais, limiares, data de início) geram o mesmo plano. Sem relógio, sem aleatoriedade, sem IA.
- TSS planejado de um treino: `horas × IF² × 100`, com IF por intensidade: baixa 0,65, moderada 0,80, alta 0,95 (força: IF 0,60, só baixa/moderada).
- Fases por distância até a prova: polimento nas 2 últimas semanas (1 em prova curta), pico nas 3 anteriores, construção em 40% do restante, base no resto; descarga a cada 4 semanas (configurável 3) com 60% do volume.
- Progressão: meta semanal de TSS cresce até a rampa máxima de CTL (5/semana por padrão) projetada pelo CTL/ATL de 42 e 7 dias; o motor escolhe o maior TSS semanal que respeita a rampa, o TSB mínimo e a disponibilidade.
- Distribuição polarizada: 80% do tempo em baixa intensidade, 20% em moderada/alta, com no máximo 2 dias de alta seguidos e nunca dois dias de alta antes de um dia de descanso.
- Validador (regras do design doc 7.3) devolve violações `obrigatoria` ou `preferencial`; motor e ajustes usam o mesmo validador. Ajuste que gere violação obrigatória é recusado.
- Dor ou lesão relatada bloqueia geração e ajuste e devolve recomendação de procurar profissional; o sistema não interpreta o relato.

## Alternativas consideradas

- IF por zona de FC em vez de por intensidade: exige perfil completo que o atleta pode não ter.
- Solver (programação linear): mais flexível, menos explicável; descartado para manter auditoria simples.

## Consequências

- Regras e constantes com testes de unidade e propriedades (determinismo; plano gerado nunca viola regra obrigatória).
- Roteiro do núcleo em `docs/roteiros/validador-de-restricoes.md`.
