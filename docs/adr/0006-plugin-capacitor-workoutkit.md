# 0006. Plugin Capacitor próprio em Swift para o WorkoutKit, publicado como open source (A-06)

- **Status:** Proposto
- **Data:** 2026-11-21
- **Id no design doc:** A-06 (seção 13; detalhe nas seções 3 e 8.2)

## Contexto

Os plugins Capacitor de saúde leem treinos, mas a escrita é limitada (ex.: `@capgo/capacitor-health` lista workouts como somente leitura). O WorkoutKit (iOS 17+/watchOS 10+) cria `CustomWorkout` e agenda no app Workout via `WorkoutScheduler`, que tem autorização própria e limites de quantidade e de janela (referência de terceiros: até 15 planos, ±7 dias). Há relatos de falhas de sincronização em versões beta do iOS.

## Decisão

Leitura de saúde pelo plugin existente; envio de treinos por um plugin próprio em Swift, `capacitor-workoutkit`, com a API TypeScript da seção 8.2 (`requestAuthorization`, `schedule`, `listScheduled`, `remove`, `removeAll`, `openInWorkoutApp`). `PlannedWorkoutDTO` é o contrato estável entre app e plugin; o `WorkoutKitPlanTranslator` (lado Swift) monta `CustomWorkout` ou `SwimBikeRunWorkout`. O backend mantém a fila e o app agenda só a janela permitida (janela deslizante). Fallback: `openInWorkoutApp()` ou export .fit. O plugin é publicado como pacote npm próprio, com README e exemplo.

## Alternativas consideradas

- Usar só o plugin de saúde existente: não escreve treinos (seção 3).

## Consequências

- Contribuição open source concreta e contrato estável entre app e plugin.
- Testes unitários do tradutor em XCTest; o agendamento real exige iPhone e Apple Watch físicos e um roteiro de teste manual.
- Build iOS exige Mac + Xcode, conta Apple Developer e CI em runner macOS.
