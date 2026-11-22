import type { z } from "zod";
import { CoachError } from "../../application/errors.js";

export interface HttpOptions {
  baseUrl: string;
  timeoutMs: number;
  fetchImpl?: typeof fetch;
}

/** Calls another context's HTTP API with the caller's own token and validates the body against the contract schema. */
export class HttpJson {
  constructor(private readonly opts: HttpOptions) {}

  async request<T>(token: string, method: "GET" | "POST", path: string, schema: z.ZodType<T>, body?: unknown): Promise<T> {
    const f = this.opts.fetchImpl ?? fetch;
    let res: Response;
    try {
      res = await f(`${this.opts.baseUrl}${path}`, {
        method,
        headers: {
          authorization: `Bearer ${token}`,
          accept: "application/json",
          ...(body === undefined ? {} : { "content-type": "application/json" }),
        },
        body: body === undefined ? undefined : JSON.stringify(body),
        signal: AbortSignal.timeout(this.opts.timeoutMs),
      });
    } catch {
      throw new CoachError("upstream", `${method} ${path} unreachable`);
    }
    if (res.status === 401 || res.status === 403) throw new CoachError("forbidden", `${method} ${path} denied`);
    if (res.status === 404) throw new CoachError("not_found", `${method} ${path} not found`);
    if (!res.ok) throw new CoachError("upstream", `${method} ${path} answered ${res.status}`);
    const parsed = schema.safeParse(await res.json().catch(() => undefined));
    if (!parsed.success) throw new CoachError("upstream", `${method} ${path} broke the contract`);
    return parsed.data;
  }
}
