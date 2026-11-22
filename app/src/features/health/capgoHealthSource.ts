import { Capacitor } from '@capacitor/core';
import { Health, type Workout } from '@capgo/capacitor-health';
import type { HealthPlatform, HealthSourcePort, HealthWorkout, WorkoutPage } from './ports';

type HealthApi = Pick<typeof Health, 'isAvailable' | 'requestAuthorization' | 'checkAuthorization' | 'queryWorkouts'>;

export function toHealthWorkout(w: Workout): HealthWorkout {
  const startMs = Date.parse(w.startDate);
  return {
    // Sem platformId a deduplicação cai na chave natural do backend; o id sintético só serve para o lote.
    externalId: w.platformId ?? `${w.sourceId ?? 'fonte'}:${startMs}:${Math.round(w.duration)}`,
    originType: w.workoutType,
    startUtc: new Date(w.startDate).toISOString(),
    endUtc: new Date(w.endDate).toISOString(),
    durationS: Math.round(w.duration),
    distanceM: w.totalDistance,
    energyKcal: w.totalEnergyBurned,
    origin: w.sourceName,
  };
}

// Adaptador do @capgo/capacitor-health: HealthKit no iOS, Health Connect no Android (sem Google Fit).
export class CapgoHealthSource implements HealthSourcePort {
  constructor(private readonly api: HealthApi = Health) {}

  async platform(): Promise<HealthPlatform | null> {
    if (!Capacitor.isNativePlatform()) return null;
    const { available } = await this.api.isAvailable();
    if (!available) return null;
    return Capacitor.getPlatform() === 'ios' ? 'healthkit' : 'health_connect';
  }

  async requestReadAuthorization(): Promise<boolean> {
    // requestHistoryAccess: sem ele o Health Connect limita a leitura aos últimos ~30 dias.
    const s = await this.api.requestAuthorization({ read: ['workouts'], requestHistoryAccess: true });
    return s.readAuthorized.includes('workouts');
  }

  async readWorkouts(q: { sinceUtc: string | null; pageAnchor?: string; limit: number }): Promise<WorkoutPage> {
    const r = await this.api.queryWorkouts({
      startDate: q.sinceUtc ?? undefined,
      anchor: q.pageAnchor,
      limit: q.limit,
      ascending: true,
    });
    return { workouts: r.workouts.map(toHealthWorkout), nextPageAnchor: r.anchor ?? undefined };
  }
}
