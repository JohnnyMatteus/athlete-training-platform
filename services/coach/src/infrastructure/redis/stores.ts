import { randomUUID } from "node:crypto";
import type { Redis } from "ioredis";
import { CoachError, conflict } from "../../application/errors.js";
import type { BudgetPort, Clock, ProposalStorePort } from "../../application/ports.js";
import type { Proposal } from "../../domain/proposal.js";

const DAY_S = 86_400;

export class RedisBudget implements BudgetPort {
  constructor(private readonly redis: Redis, private readonly dailyTokens: number, private readonly clock: Clock) {}

  private key(athleteId: string) {
    return `coach:budget:${athleteId}:${this.clock.now().toISOString().slice(0, 10)}`;
  }

  async reserve(athleteId: string, tokens: number): Promise<void> {
    const key = this.key(athleteId);
    const total = await this.redis.incrby(key, tokens);
    await this.redis.expire(key, 2 * DAY_S);
    if (total > this.dailyTokens) {
      await this.redis.decrby(key, tokens);
      throw new CoachError("budget_exceeded", "daily AI budget reached");
    }
  }

  async settle(athleteId: string, reserved: number, actual: number): Promise<void> {
    await this.redis.incrby(this.key(athleteId), actual - reserved);
  }
}

export class RedisProposalStore implements ProposalStorePort {
  constructor(private readonly redis: Redis, private readonly ttlSeconds: number) {}

  async save(p: Proposal): Promise<void> {
    await this.redis.multi().set(`coach:proposal:${p.id}`, JSON.stringify(p), "EX", this.ttlSeconds).sadd(`coach:athlete-proposals:${p.athleteId}`, p.id).expire(`coach:athlete-proposals:${p.athleteId}`, this.ttlSeconds).exec();
  }

  async get(id: string): Promise<Proposal | null> {
    const raw = await this.redis.get(`coach:proposal:${id}`);
    return raw ? (JSON.parse(raw) as Proposal) : null;
  }

  async listByAthlete(athleteId: string): Promise<Proposal[]> {
    const ids = await this.redis.smembers(`coach:athlete-proposals:${athleteId}`);
    const all = await Promise.all(ids.map((id) => this.get(id)));
    return all.filter((p): p is Proposal => p !== null).sort((a, b) => b.createdAt.localeCompare(a.createdAt));
  }

  async withLock<T>(id: string, fn: () => Promise<T>): Promise<T> {
    const key = `coach:lock:${id}`;
    const token = randomUUID();
    if ((await this.redis.set(key, token, "EX", 30, "NX")) !== "OK") throw conflict("proposal is being decided");
    try {
      return await fn();
    } finally {
      if ((await this.redis.get(key)) === token) await this.redis.del(key);
    }
  }
}
