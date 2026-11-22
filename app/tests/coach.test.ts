import { describe, expect, it, vi } from 'vitest';
import { ApiError, createHttp } from '@/api/http';
import { createCoachApi } from '@/api/coach';

const ID = '3c1d2e4f-0b1a-4c2d-8e3f-9a8b7c6d5e40';
const proposal = {
  id: ID, athleteId: ID, planId: 'p', requestedBy: ID, status: 'pending_athlete',
  createdAt: '2026-11-22T10:00:00.000Z', expiresAt: '2026-11-23T10:00:00.000Z', rationale: 'descanso',
  changes: [{ session_id: ID, action: 'move', new_date: '2026-11-25' }],
};
const api = (res: Response) =>
  createCoachApi(createHttp({ baseUrl: 'http://coach.test', token: async () => 't', fetch: vi.fn().mockResolvedValue(res) as never }));
const json = (status: number, b: unknown) => new Response(JSON.stringify(b), { status });

describe('coach', () => {
  it('lê o resultado com proposta e a lista', async () => {
    const r = await api(json(200, { kind: 'proposal', message: 'ok', proposal })).ask('adjust_week', 'perdi treinos');
    expect(r.kind === 'proposal' && r.proposal.status).toBe('pending_athlete');
    expect((await api(json(200, { items: [proposal] })).proposals()).items).toHaveLength(1);
  });

  it('erro do coach vira ApiError com a mensagem', async () => {
    const e = await api(json(409, { error: 'conflict', message: 'proposal expired' })).approve(ID).catch((x) => x);
    expect(e).toBeInstanceOf(ApiError);
    expect(e.detail).toBe('proposal expired');
  });
});
