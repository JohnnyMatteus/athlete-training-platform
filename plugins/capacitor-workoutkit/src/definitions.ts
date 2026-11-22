// Contrato estável entre o app e o plugin (ADR 0006). Unidades em SI.
export type Goal =
  | { tipo: 'tempo'; segundos: number }
  | { tipo: 'distancia'; metros: number }
  | { tipo: 'aberto' };

export type Target =
  | { tipo: 'fc_bpm' | 'potencia_w' | 'velocidade_ms'; min: number; max: number }
  | { tipo: 'zona_fc' | 'zona_potencia'; zona: number };

export interface PlannedStep {
  tipo: 'aquecimento' | 'trabalho' | 'recuperacao' | 'desaquecimento';
  nome?: string;
  objetivo: Goal;
  alvo?: Target;
}

export interface PlannedWorkoutDTO {
  /** UUID: vira o id do WorkoutPlan no relógio, então reagendar o mesmo treino é idempotente. */
  id: string;
  nome: string;
  modalidade: 'corrida' | 'ciclismo' | 'natacao' | 'forca' | 'outro';
  ambiente?: 'indoor' | 'outdoor' | 'piscina' | 'aguas_abertas';
  aquecimento?: PlannedStep;
  blocos: { repeticoes: number; passos: PlannedStep[] }[];
  desaquecimento?: PlannedStep;
}

/** Data e hora locais do atleta, sem fuso: `2026-11-23T06:30`. */
export type LocalDateTime = string;

export interface WindowItem {
  plan: PlannedWorkoutDTO;
  date: LocalDateTime;
}

export interface SyncWindowResult {
  added: string[];
  removed: string[];
  /** Dentro da janela, mas sem vaga: entram na próxima abertura do app. */
  deferred: string[];
  /** Falhas de agendamento por id; o app oferece o fallback (.fit). */
  failed: { id: string; reason: string }[];
}

export interface WorkoutKitPlugin {
  requestAuthorization(): Promise<{ status: 'authorized' | 'denied' | 'notDetermined' }>;
  schedule(options: { plan: PlannedWorkoutDTO; date: LocalDateTime }): Promise<{ id: string }>;
  listScheduled(): Promise<{ items: { id: string; date: LocalDateTime; name: string }[] }>;
  remove(options: { id: string }): Promise<void>;
  removeAll(): Promise<void>;
  openInWorkoutApp(options: { plan: PlannedWorkoutDTO }): Promise<void>;
  /**
   * Extensão do design doc (ADR 0031): reconcilia o que está agendado com a fila do backend,
   * respeitando a janela deslizante e o limite de planos do WorkoutScheduler.
   */
  syncWindow(options: { items: WindowItem[]; today: string; windowDays?: number }): Promise<SyncWindowResult>;
  /** Fallback (.fit de treino estruturado), em base64, para compartilhar com outro app. */
  exportFit(options: { plan: PlannedWorkoutDTO }): Promise<{ filename: string; base64: string }>;
}
