import { Redis } from "ioredis";
import { ApprovalService } from "./application/approval.js";
import { CoachService } from "./application/service.js";
import type { Config } from "./config.js";
import { JwtAuthenticator } from "./infrastructure/auth/jwt.js";
import { HttpJson } from "./infrastructure/http/client.js";
import { HttpAthleteData, HttpPlanApplier, HttpPlanValidator } from "./infrastructure/http/adapters.js";
import { AnthropicLlm } from "./infrastructure/llm/anthropic.js";
import { RedisBudget, RedisProposalStore } from "./infrastructure/redis/stores.js";
import { buildApp } from "./interfaces/http/app.js";

export function compose(cfg: Config) {
  const redis = new Redis(cfg.redisUrl);
  const clock = { now: () => new Date() };
  const metrics = new HttpJson({ baseUrl: cfg.metricsUrl, timeoutMs: cfg.upstreamTimeoutMs });
  const planning = new HttpJson({ baseUrl: cfg.planningUrl, timeoutMs: cfg.upstreamTimeoutMs });
  const data = new HttpAthleteData(metrics, planning, clock);
  const validator = new HttpPlanValidator(planning);
  const proposals = new RedisProposalStore(redis, cfg.proposalTtlHours * 3600 * 2);

  const service = new CoachService({
    data, validator, proposals, clock,
    llm: new AnthropicLlm(cfg.anthropicApiKey, cfg.model, cfg.llmTimeoutMs),
    budget: new RedisBudget(redis, cfg.dailyTokenBudget, clock),
    graph: cfg.graph,
    proposalTtlHours: cfg.proposalTtlHours,
  });
  const approvals = new ApprovalService({ data, validator, applier: new HttpPlanApplier(planning), proposals, clock });
  const app = buildApp({ auth: new JwtAuthenticator(cfg.jwt), service, approvals });
  return { app, redis };
}
