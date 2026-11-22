# capacitor-workoutkit

Plugin Capacitor em Swift que agenda treinos estruturados no app Workout do Apple Watch pelo WorkoutKit (iOS 17+, watchOS 10+). ADRs 0006 e 0034. **Não publicado no npm**: o app o consome por alias (`@atp/capacitor-workoutkit`); a publicação é decisão do autor.

## API

`requestAuthorization`, `schedule`, `listScheduled`, `remove`, `removeAll`, `openInWorkoutApp` (design doc 8.2) mais duas extensões:

- `syncWindow({ items, today, windowDays? })`: janela deslizante. Remove do relógio o que o plano cancelou ou moveu, agenda os próximos `windowDays` (padrão 7) até o limite de planos do WorkoutScheduler (lido do sistema; padrão 15) e devolve `added`, `removed`, `deferred` e `failed`. Rodar de novo sem mudança no plano não altera nada.
- `exportFit({ plan })`: fallback, gera o `.fit` de treino estruturado no aparelho (base64).

`PlannedWorkoutDTO` (`src/definitions.ts`) usa SI: segundos, metros, bpm, watts, m/s. O `id` é UUID e vira o id do `WorkoutPlan`. O fixture `fixtures/planned-workout.json` é validado nos dois lados (zod no app, `swift test` aqui).

## Estrutura

| Caminho | Papel |
| --- | --- |
| `ios/WorkoutKitCore` | Núcleo Swift sem Capacitor: DTO, `WorkoutTranslator` (DTO para modelo neutro validado), `SlidingWindowPlanner`, `FitWorkoutEncoder`. Testes com Swift Testing |
| `ios/Sources/WorkoutKitPlugin` | Ponte Capacitor e `WorkoutKitMapper` (modelo neutro para `CustomWorkout`) |
| `src/` | Definições TypeScript e stub web (recusa, não simula) |

`CustomWorkout` cobre treino de uma modalidade. `SwimBikeRunWorkout` (multiesporte) **não está implementado**.

## Testar

```bash
cd ios/WorkoutKitCore && swift test      # precisa de Xcode 16 (Swift Testing)
```

Teste manual no aparelho (iPhone com iOS 17+ e Apple Watch pareado):

1. Instale o app, abra Plano e toque "Enviar ao relógio"; aceite a autorização do WorkoutKit.
2. No Watch, abra o app Workout: os treinos dos próximos 7 dias aparecem como treinos personalizados, com alvos de FC, potência ou ritmo.
3. Mude o plano no servidor (mova um treino) e abra o app de novo: o antigo some, o novo aparece.
4. Force uma falha (negue a autorização) e confira o fallback: "Abrir no app Workout" e, se falhar, o `.fit` no compartilhamento.
5. Registre iOS, versão do watchOS e o resultado; há relatos de falhas do WorkoutScheduler em betas.

## O que não foi verificado

- O toolchain Swift deste ambiente não importa Foundation (módulo duplicado em `/Library/Developer/CommandLineTools/usr/include/swift`) e não tem Xcode, XCTest nem SwiftPM funcional. A lógica sem Foundation (tradutor, janela, `.fit`, CRC) foi compilada e exercitada com um harness avulso. `swift test`, a decodificação do JSON, a ponte Capacitor e o mapeador WorkoutKit **só serão compilados no CI macOS** (job `plugin-swift`), então os nomes exatos dos inicializadores do WorkoutKit podem exigir ajuste.
- O `.fit` segue o perfil público do FIT e fecha o CRC, mas não foi aberto no SDK da Garmin nem em um dispositivo; o passo de repetição usa `target_type=open` e `target_value=repetições`.
- Nenhum agendamento real foi feito. `WorkoutScheduler.schedule(_:at:)` não lança nem devolve resultado (doc da Apple): o plugin confere `isSupported` antes, `CustomWorkout.supportsActivity/supportsGoal/supportsAlert` ao montar o treino, e relê `scheduledWorkouts` depois de agendar (compara id e dia). O que não entrou volta em `failed` no `syncWindow` e rejeita em `schedule`. Essa conferência só compila no CI macOS e só se prova no aparelho (limite de 15 planos, autorização negada, beta do iOS).
- `openInWorkoutApp` **não está implementado no iOS**: `WorkoutPlan.openInWorkoutApp()` é do watchOS e não compila no iOS. O método rejeita com `UNIMPLEMENTED` (nunca resolve sem ter feito nada) e o app cai no `exportFit`. A alternativa da Apple no iOS 17, a prévia do WorkoutKit (modificador SwiftUI `workoutPreview`), não foi implementada nem exercitada em aparelho.
