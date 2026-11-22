import { describe, expect, it, vi } from 'vitest';
import { EndpointUnavailableError } from '@/api/http';
import { toHealthWorkout } from '@/features/health/capgoHealthSource';
import type { HealthSourcePort, HealthWorkout } from '@/features/health/ports';
import { AnchorStore, HealthSyncService } from '@/features/health/syncService';
import { ConsentRepository } from '@/features/session/consent';
import { MemoryStore } from '@/platform/kv';

const w = (id: string, end: string): HealthWorkout => ({ externalId: id, originType: 'running', startUtc: end, endUtc: end, durationS: 60 });

function setup(pages: HealthWorkout[][]) {
  const kv = new MemoryStore();
  const consent = new ConsentRepository(kv);
  const anchors = new AnchorStore(kv);
  const read = vi.fn(async ({ pageAnchor }: { sinceUtc: string | null; pageAnchor?: string; limit: number }) => {
    const i = pageAnchor ? Number(pageAnchor) : 0;
    return { workouts: pages[i] ?? [], nextPageAnchor: i + 1 < pages.length ? String(i + 1) : undefined };
  });
  const source: HealthSourcePort = { platform: async () => 'healthkit', requestReadAuthorization: async () => true, readWorkouts: read };
  const send = vi.fn().mockResolvedValue(undefined);
  const svc = new HealthSyncService(source, anchors, { send }, consent, () => new Date('2026-11-22T10:00:00Z'));
  return { consent, anchors, read, send, svc };
}

describe('leitura incremental', () => {
  it('não lê sem consentimento de treinos', async () => {
    const { svc, read } = setup([]);
    expect(await svc.run()).toEqual({ status: 'sem_consentimento' });
    expect(read).not.toHaveBeenCalled();
  });

  it('pagina, entrega e persiste a âncora; a próxima leitura parte dela', async () => {
    const { svc, consent, anchors, read, send } = setup([[w('a', '2026-11-01T10:00:00.000Z')], [w('b', '2026-11-02T10:00:00.000Z')]]);
    await consent.accept(['treinos']);
    expect(await svc.run()).toEqual({ status: 'ok', sent: 2 });
    expect(send).toHaveBeenCalledTimes(2);
    expect((await anchors.load())?.sinceUtc).toBe('2026-11-02T10:00:00.000Z');
    await svc.run();
    expect(read.mock.calls.at(-1)![0].sinceUtc).toBe('2026-11-02T10:00:00.000Z');
  });

  it('servidor sem endpoint: não avança a âncora', async () => {
    const { svc, consent, anchors, send } = setup([[w('a', '2026-11-01T10:00:00.000Z')]]);
    await consent.accept(['treinos']);
    send.mockRejectedValue(new EndpointUnavailableError('/health/workouts'));
    expect(await svc.run()).toEqual({ status: 'servidor_indisponivel', sent: 0 });
    expect(await anchors.load()).toBeNull();
  });

  it('traduz o treino do plugin para o modelo do app', () => {
    const r = toHealthWorkout({ workoutType: 'cycling', duration: 3600.4, startDate: '2026-11-01T10:00:00Z', endDate: '2026-11-01T11:00:00Z', totalDistance: 30000, platformId: 'u1' });
    expect(r).toMatchObject({ externalId: 'u1', originType: 'cycling', durationS: 3600, distanceM: 30000 });
  });
});
