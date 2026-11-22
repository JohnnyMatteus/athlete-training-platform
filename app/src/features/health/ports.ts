// Tipos do domínio do app. Nada do plugin de saúde atravessa estas portas.
export interface HealthWorkout {
  externalId: string;
  originType: string;
  startUtc: string;
  endUtc: string;
  durationS: number;
  distanceM?: number;
  energyKcal?: number;
  origin?: string;
}

export type HealthPlatform = 'healthkit' | 'health_connect';

export interface WorkoutPage {
  workouts: HealthWorkout[];
  /** Cursor da próxima página dentro da mesma leitura; ausente quando acabou. */
  nextPageAnchor?: string;
}

export interface HealthSourcePort {
  platform(): Promise<HealthPlatform | null>;
  requestReadAuthorization(): Promise<boolean>;
  readWorkouts(query: { sinceUtc: string | null; pageAnchor?: string; limit: number }): Promise<WorkoutPage>;
}

export interface WorkoutSink {
  send(platform: HealthPlatform, workouts: HealthWorkout[]): Promise<void>;
}

export interface Anchor {
  /** Fim (UTC) do último treino entregue ao backend. */
  sinceUtc: string;
  updatedAt: string;
}
