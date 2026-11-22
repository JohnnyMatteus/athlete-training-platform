# Roteiro: tradutor WorkoutKit (`tradutor-workoutkit`)

Código: `plugins/capacitor-workoutkit/ios/WorkoutKitCore/Sources/WorkoutKitCore/Translator.swift` (design doc 4.4 e 8.2). Núcleo do modo par: o autor escreve o tradutor e as regras; a ponte e o mapeador são de apoio.

## A solução em uma página

`PlannedWorkoutDTO` (SI, português) entra; sai um modelo neutro já validado (`WorkoutBlueprint`). Só depois disso o mapeador cria `CustomWorkout`. Validar antes de tocar o WorkoutKit torna a regra testável sem Xcode e deixa o mesmo modelo alimentar o `.fit`.

Regras que o tradutor impõe: id em UUID (vira o id do `WorkoutPlan`, o que torna reagendar idempotente); aquecimento e desaquecimento só fora dos blocos; potência só em ciclismo; faixa com mínimo positivo e máximo maior ou igual; zonas de FC 1 a 5 e de potência 1 a 7; objetivo positivo; treino sem nenhum passo é recusado.

## Alternativas e trade-offs

- Validar no mapeador: acopla a regra ao iOS e impede teste em CI comum.
- Traduzir % do limiar no plugin: o plugin passaria a conhecer limiares; ficou no app (ADR 0034).
- `SwimBikeRunWorkout`: não coberto; multiesporte pede um DTO de sessões.

## Perguntas de entrevista

1. Por que o id do treino é o id do plano no relógio?
2. O que acontece se o WorkoutScheduler recusar o agendamento? Qual a ordem do fallback?
3. Como a janela deslizante evita agendar de novo o que já está certo?
4. Por que o `.fit` fecha o CRC em zero quando recalculado sobre o arquivo inteiro?
5. Onde a regra "potência só em ciclismo" deveria viver, e por quê?
