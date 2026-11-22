# 0034. App e plugin: janela deslizante no plugin e tradução de alvos no app

- **Status:** Proposto
- **Data:** 2026-11-22
- **Id no design doc:** A-06, seção 8.2, M5

## Contexto

O plano do servidor usa alvos em % do limiar; o WorkoutKit precisa de valores absolutos. O WorkoutScheduler limita quantidade e janela (referência de terceiros: 15 planos, ±7 dias).

## Decisão

- O app traduz `Workout` do plano em `PlannedWorkoutDTO` (SI) com os limiares e o perfil vigentes (`toPlannedWorkout.ts`). Sem limiar, ou com RPE, o passo fica sem alvo (mesma regra do export .fit do servidor). Só plano `ativo` vai ao relógio.
- O plugin ganha `syncWindow` (extensão da API do design doc): reconcilia o agendado com a fila, dentro de ±7 dias e do máximo, sem reagendar o que já está certo. O id do `WorkoutPlan` é o UUID do treino, então reagendar é idempotente.
- Fallback: `openInWorkoutApp()` não existe no iOS (é do watchOS), então o plugin a rejeita sempre e o app cai direto em `exportFit`, que gera o .fit no aparelho; a prévia do WorkoutKit (`workoutPreview`) fica como evolução, não exercitada em aparelho (também existe `GET /treinos/{id}/export.fit` no servidor).
- Horário padrão do treino: 07:00 local, pois o contrato só tem a data.

## Premissas que o autor confirma

- `pace_pct_limiar` e `css_pct` são % da velocidade de limiar.
- Os limites 15 e ±7 vêm de terceiros; o plugin lê `maxAllowedScheduledWorkoutCount` do sistema quando disponível.
- Multiesporte (`SwimBikeRunWorkout`) não está implementado.

## Consequências

- Duplicação intencional do tradutor .fit (servidor e Swift): o do aparelho funciona offline.
- O plugin só é verificável por XCTest/Swift Testing no núcleo; o agendamento real exige iPhone e Apple Watch.
