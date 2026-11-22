import { describe, expect, it, vi } from 'vitest';
import { z } from 'zod';
import { ApiError, ContractError, EndpointUnavailableError, NetworkError, createHttp } from '@/api/http';

const json = (status: number, body: unknown) =>
  new Response(JSON.stringify(body), { status, headers: { 'Content-Type': 'application/json' } });
const make = (f: typeof fetch, onUnauthorized = vi.fn()) =>
  createHttp({ baseUrl: 'http://x.test', token: async () => 'tok', fetch: f, onUnauthorized });

describe('cliente HTTP', () => {
  it('envia o Bearer e valida a resposta', async () => {
    const f = vi.fn().mockResolvedValue(json(200, { a: 1 }));
    const r = await make(f)('/p', { schema: z.object({ a: z.number() }), query: { de: '2026-01-01' } });
    expect(r).toEqual({ a: 1 });
    const [url, init] = f.mock.calls[0]!;
    expect(String(url)).toBe('http://x.test/p?de=2026-01-01');
    expect(init.headers.Authorization).toBe('Bearer tok');
  });

  it('endpoint proposto ausente vira indisponível, publicado vira ApiError', async () => {
    const f = vi.fn().mockResolvedValue(json(404, { type: 't', title: 'x', status: 404 }));
    await expect(make(f)('/p', { schema: z.object({}), proposed: true })).rejects.toBeInstanceOf(EndpointUnavailableError);
    await expect(make(f)('/p', { schema: z.object({}) })).rejects.toBeInstanceOf(ApiError);
  });

  it('resposta fora do contrato, falha de rede e 401', async () => {
    await expect(make(vi.fn().mockResolvedValue(json(200, { a: 'x' })))('/p', { schema: z.object({ a: z.number() }) })).rejects.toBeInstanceOf(ContractError);
    await expect(make(vi.fn().mockRejectedValue(new TypeError('down')))('/p', { schema: z.object({}) })).rejects.toBeInstanceOf(NetworkError);
    const unauthorized = vi.fn();
    await expect(make(vi.fn().mockResolvedValue(json(401, { type: 't', title: 'x', status: 401 })), unauthorized)('/p', { schema: z.object({}) })).rejects.toBeInstanceOf(ApiError);
    expect(unauthorized).toHaveBeenCalled();
  });
});
