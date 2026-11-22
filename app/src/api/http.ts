import type { z } from 'zod';
import { problemSchema, type Problem } from './schemas';

export class ApiError extends Error {
  constructor(
    readonly status: number,
    readonly problem: Problem | null,
  ) {
    super(problem?.title ?? `HTTP ${status}`);
  }
}

// A API ainda não publica este endpoint (404, 405 ou 501 num caminho só proposto).
export class EndpointUnavailableError extends Error {
  constructor(readonly path: string) {
    super(`Endpoint indisponível: ${path}`);
  }
}

export class NetworkError extends Error {}

export class ContractError extends Error {
  constructor(
    readonly path: string,
    readonly issues: string,
  ) {
    super(`Resposta fora do contrato em ${path}`);
  }
}

export interface HttpDeps {
  baseUrl: string;
  token: () => Promise<string | null>;
  onUnauthorized?: () => void;
  fetch?: typeof fetch;
}

export interface RequestOptions<S extends z.ZodType> {
  method?: 'GET' | 'POST' | 'PUT' | 'PATCH';
  body?: unknown;
  schema: S;
  query?: Record<string, string | undefined>;
  // true: endpoint fora de contracts/openapi; ausência vira EndpointUnavailableError.
  proposed?: boolean;
  signal?: AbortSignal;
}

const ABSENT = new Set([404, 405, 501]);

export function createHttp(deps: HttpDeps) {
  const doFetch = deps.fetch ?? fetch;

  return async function request<S extends z.ZodType>(path: string, opts: RequestOptions<S>): Promise<z.infer<S>> {
    const url = new URL(path, deps.baseUrl);
    for (const [k, v] of Object.entries(opts.query ?? {})) if (v !== undefined) url.searchParams.set(k, v);

    const headers: Record<string, string> = { Accept: 'application/json, application/problem+json' };
    const token = await deps.token();
    if (token) headers.Authorization = `Bearer ${token}`;
    if (opts.body !== undefined) headers['Content-Type'] = 'application/json';

    let res: Response;
    try {
      res = await doFetch(url, {
        method: opts.method ?? 'GET',
        headers,
        body: opts.body === undefined ? undefined : JSON.stringify(opts.body),
        signal: opts.signal,
      });
    } catch (e) {
      throw new NetworkError(e instanceof Error ? e.message : 'falha de rede');
    }

    if (res.status === 401) deps.onUnauthorized?.();
    if (opts.proposed && ABSENT.has(res.status)) throw new EndpointUnavailableError(path);
    if (!res.ok) {
      const parsed = problemSchema.safeParse(await res.json().catch(() => null));
      throw new ApiError(res.status, parsed.success ? parsed.data : null);
    }

    const parsed = opts.schema.safeParse(await res.json());
    if (!parsed.success) throw new ContractError(path, parsed.error.message);
    return parsed.data;
  };
}

export type Http = ReturnType<typeof createHttp>;
