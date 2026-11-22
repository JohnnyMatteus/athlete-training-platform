import { EndpointUnavailableError } from '@/api/http';
import type { HealthBatch, HealthIngestApi } from '@/api/proposed';
import type { ConsentRepository } from '@/features/session/consent';
import type { KeyValueStore } from '@/platform/kv';
import type { Anchor, HealthPlatform, HealthSourcePort, HealthWorkout, WorkoutSink } from './ports';

const ANCHOR_KEY = 'health.anchor.v1';

// Persiste até onde já entregamos ao backend. Só avança depois de a entrega confirmar.
export class AnchorStore {
  constructor(private readonly kv: KeyValueStore) {}
  async load(): Promise<Anchor | null> {
    try {
      const v = JSON.parse((await this.kv.get(ANCHOR_KEY)) ?? 'null') as Anchor | null;
      return v && typeof v.sinceUtc === 'string' ? v : null;
    } catch {
      return null;
    }
  }
  save(a: Anchor): Promise<void> {
    return this.kv.set(ANCHOR_KEY, JSON.stringify(a));
  }
  reset(): Promise<void> {
    return this.kv.remove(ANCHOR_KEY);
  }
}

export class HealthIngestSink implements WorkoutSink {
  constructor(private readonly api: HealthIngestApi) {}
  send(platform: HealthPlatform, workouts: HealthWorkout[]): Promise<void> {
    const batch: HealthBatch = {
      fonte: platform,
      treinos: workouts.map((w) => ({
        id_externo: w.externalId,
        tipo_origem: w.originType,
        inicio_utc: w.startUtc,
        fim_utc: w.endUtc,
        duracao_s: w.durationS,
        distancia_m: w.distanceM,
        energia_kcal: w.energyKcal,
        origem: w.origin,
      })),
    };
    return this.api.sendBatch(batch).then(() => undefined);
  }
}

export type SyncOutcome =
  | { status: 'sem_consentimento' }
  | { status: 'indisponivel_plataforma' }
  | { status: 'permissao_negada' }
  | { status: 'servidor_indisponivel'; sent: number }
  | { status: 'ok'; sent: number };

const PAGE = 100;

export class HealthSyncService {
  constructor(
    private readonly source: HealthSourcePort,
    private readonly anchors: AnchorStore,
    private readonly sink: WorkoutSink,
    private readonly consent: ConsentRepository,
    private readonly now: () => Date = () => new Date(),
  ) {}

  async run(): Promise<SyncOutcome> {
    if (!(await this.consent.has('treinos'))) return { status: 'sem_consentimento' };
    const platform = await this.source.platform();
    if (!platform) return { status: 'indisponivel_plataforma' };
    if (!(await this.source.requestReadAuthorization())) return { status: 'permissao_negada' };

    const since = (await this.anchors.load())?.sinceUtc ?? null;
    let pageAnchor: string | undefined;
    let sent = 0;

    do {
      const page = await this.source.readWorkouts({ sinceUtc: since, pageAnchor, limit: PAGE });
      // O início da janela é inclusivo: o último treino já entregue volta e o backend o deduplica.
      const fresh = page.workouts;
      if (fresh.length > 0) {
        try {
          await this.sink.send(platform, fresh);
        } catch (e) {
          if (e instanceof EndpointUnavailableError) return { status: 'servidor_indisponivel', sent };
          throw e;
        }
        sent += fresh.length;
        const latest = fresh.reduce((m, w) => (w.endUtc > m ? w.endUtc : m), since ?? '');
        await this.anchors.save({ sinceUtc: latest, updatedAt: this.now().toISOString() });
      }
      pageAnchor = page.nextPageAnchor;
    } while (pageAnchor);

    return { status: 'ok', sent };
  }
}
