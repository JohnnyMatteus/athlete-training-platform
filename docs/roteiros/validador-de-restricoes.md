# Roteiro: motor de periodização e validador de restrições (`validador-de-restricoes`)

Código: `services/planning/src/Domain/Plan/` (`ConstraintValidator`, `PeriodizationEngine`, `WorkoutBuilder`, `Adjustment`, `LoadModel`). Design doc, seções 7.2 e 7.3; ADR 0005 e ADR 0016. Domínio puro: sem I/O, sem relógio, sem aleatoriedade.

## A solução em uma página

1. **O validador é uma função do plano.** `validate(Plan)` simula CTL/ATL dia a dia com `LoadModel::series` (constantes de 42 e 7 dias, `TSB(t) = CTL(t-1) - ATL(t-1)`) e devolve violações `obrigatoria` ou `preferencial`. Nada de estado, nada de IA: o motor e os ajustes passam pelo mesmo código.
2. **Regras** (valores padrão do design doc, todos configuráveis em `Constraints`):
   - rampa de CTL por semana acima de 5: obrigatória, comparando o CTL do fim da semana com o do fim da anterior;
   - TSB antes de sessão de alta intensidade abaixo de -25: obrigatória, olhando o TSB do dia anterior à sessão;
   - mais de 2 dias seguidos com sessão de alta intensidade: obrigatória;
   - semana completa sem dia livre: obrigatória (a semana da prova é parcial e não conta);
   - volume do dia ou da semana acima da disponibilidade declarada: obrigatória (dia sem disponibilidade tem limite zero, então mover treino para um dia indisponível é pego aqui);
   - musculação no mesmo dia de corrida moderada ou alta: preferencial, só aviso;
   - relato de dor ou lesão: obrigatória e sem interpretação; o chamador bloqueia a geração ou o ajuste e devolve a recomendação de procurar um profissional.
3. **TSS planejado** de um treino: `horas × IF² × 100`, com IF por intensidade (baixa 0,65, moderada 0,80, alta 0,95; força 0,60 e 0,70). É o mesmo formato do TSS realizado, o que torna a aderência uma razão entre grandezas comparáveis.
4. **Motor.** Fases pela distância até a prova (polimento, pico, construção, base), descarga a cada 4 semanas com 60% da carga. A meta de TSS da semana vem de inverter o filtro de 42 dias: dado o CTL de hoje e a rampa desejada (80% ou 90% do limite), qual TSS diário médio leva a esse CTL em 7 dias. Depois a semana é montada (dias disponíveis, um dia livre, longão no dia mais longo, sessões de qualidade espaçadas, rodízio de modalidades) e **validada**; se violar regra obrigatória, o volume encolhe em passos de 10% e, no limite, as sessões de qualidade saem. Se nem assim passar, o motor falha com `ConstraintsNotSatisfiable` em vez de entregar plano inválido.
5. **Ajustes.** `Adjustment::apply` aplica a proposta a uma cópia; `ConstraintValidator` decide. Só ajuste sem violação obrigatória vira nova versão. `POST /planos/{id}/ajustes/validacao` roda o mesmo caminho sem gravar: a classe `ValidateAdjustment` nem recebe repositório de escrita, outbox ou transação, então "somente leitura" é garantido pela estrutura e não por uma flag.

## Decisões e alternativas

| Decisão | Alternativa | Por quê |
| --- | --- | --- |
| Validar o plano inteiro a cada semana montada | Validar só a semana nova | A rampa e o TSB dependem do que veio antes; o custo é quadrático em semanas, irrelevante para 104 semanas no máximo. |
| Encolher o volume até passar | Solver de otimização | Explicável e determinístico; o solver acharia plano melhor, mas ninguém auditaria o porquê. |
| Falhar quando não há plano válido | Entregar o melhor esforço | Plano arriscado é o risco nº 1 do design doc (seção 15). |
| IDs de treino por UUIDv5 do plano e da posição | UUID aleatório | Mesmas entradas, mesmo plano, inclusive os identificadores. |
| Dor bloqueia, não é classificada | Pontuar gravidade | O sistema não diagnostica (design doc, seção 2). |
| Musculação com corrida de limiar só como aviso | Regra obrigatória | O design doc diz "preferencial". |
| Modalidade fora da lista do dia não é validada no ajuste | Regra nova | Não está no design doc 7.3; limitação conhecida, registrada aqui. |

## Como foi testado

- `ConstraintValidatorTest`: uma regra por teste, com valores calculados à mão (TSB de -40 contra limite de -25; 60 contra 45 minutos; 3 dias de alta contra 2; 7 dias com sessão contra descanso mínimo de 1).
- `PeriodizationEngineTest`: determinismo (mesma entrada, mesmo documento), plano sem violação obrigatória em quatro cenários (CTL 0, 38,5, 85 e prova em 3 semanas), fases e descarga, rampa dentro do limite configurado, distribuição majoritariamente de baixa intensidade, blocos somando a duração do treino.
- `PlanUseCasesTest`: ajuste aceito gera versão nova e eventos; ajuste recusado não grava nem publica; `/ajustes/validacao` com zero escrita, zero transação e zero outbox; dor bloqueia geração e ajuste.
- `FitWorkoutTranslatorTest`: estrutura, CRC e valores decodificados. O arquivo também foi lido com a biblioteca `fitparse` (Python) numa checagem manual, que reconheceu `file_id`, `workout` e `workout_step`.

## Perguntas de entrevista

1. Por que o validador é código e não prompt? O que o Coach IA pode e não pode fazer (ADR 0005)?
2. Como a rampa de CTL por semana se relaciona com a constante de 42 dias? Dê o TSS diário que leva um CTL de 40 a 45 em 7 dias.
3. Por que o TSB mínimo olha o dia anterior à sessão?
4. O motor encolhe o volume até passar. Que plano ele nunca consegue gerar e o que acontece então?
5. Por que a rota de validação é separada da de escrita, em vez de `?dry_run=true`?
6. Que regra do design doc está fora do validador hoje (modalidade fora da disponibilidade do dia) e como você a adicionaria sem quebrar o contrato?
7. Como você testaria que o motor é determinístico, e o que quebraria isso (relógio, `uniqid`, ordem de arrays)?
