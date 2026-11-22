import RedisMock from "ioredis-mock";
import type { Redis } from "ioredis";
import { ApprovalService } from "../src/application/approval.js";
import type { GraphConfig } from "../src/application/graph.js";
import type { AthleteDataPort, LlmPort, LlmRequest, PlanApplierPort, PlanValidatorPort, ValidationResult } from "../src/application/ports.js";
import { CoachService } from "../src/application/service.js";
import type { Activities, CoachLink, LoadHistory, Plan, Profile } from "../src/domain/athlete-data.js";
import type { Principal, Subject } from "../src/domain/principal.js";
import type { Proposal, ProposalDraft } from "../src/domain/proposal.js";
import { RedisBudget, RedisProposalStore } from "../src/infrastructure/redis/stores.js";

export const ATHLETE = "2f0f0a2e-6a53-4c3a-9d8e-1b7e6c5d4a01";
export const OTHER_ATHLETE = "2f0f0a2e-6a53-4c3a-9d8e-1b7e6c5d4a99";
export const COACH = "7d2a8c60-1f4b-4e0a-9b53-6a1e2f3c4d55";
export const PLAN_ID = "5b0e0f5a-7a41-4c36-a6a7-3b9d3f6a1c10";
export const S_EASY = "0a1b2c3d-1111-4222-8333-444455556666";
export const S_HARD = "0a1b2c3d-1111-4222-8333-444455557777";

export const athlete = (sub = ATHLETE): Principal => ({ subject: sub, role: "athlete", token: `token-of-${sub}` });
export const coach = (sub = COACH): Principal => ({ subject: sub, role: "coach", token: `token-of-${sub}` });
export const subjectOf = (principal: Principal, athleteId = ATHLETE): Subject => ({ principal, athleteId });

export const basePlan = (): Plan => ({
  plan_id: PLAN_ID,
  week_start: "2026-11-23",
  sessions: [
    { id: S_EASY, date: "2026-11-24", sport: "run", name: "Rodagem leve", intensity: "low", duration_min: 50, planned_tss: 45, steps: [] },
    { id: S_HARD, date: "2026-11-26", sport: "bike", name: "Limiar", intensity: "high", duration_min: 60, planned_tss: 70, steps: [] },
  ],
});

export class FakeData implements AthleteDataPort {
  calls: Array<{ method: string; token: string; athleteId: string }> = [];
  plan: Plan = basePlan();
  activities: Activities = { items: [{ id: "9c1a7f4e-2b3d-4a55-8f0e-0d6b1c2a3e44", day: "2026-11-20", type: "corrida", method: "rtss", tss: 60 }] };
  link: CoachLink = { coach_id: null, status: "none", consent: { view_data: false, approve_plans: false } };
  private rec(method: string, s: Subject) {
    this.calls.push({ method, token: s.principal.token, athleteId: s.athleteId });
  }
  async getProfile(s: Subject): Promise<Profile> {
    this.rec("profile", s);
    return { resting_hr: 52, max_hr: 188, thresholds: [{ type: "ftp", value: 230, valid_since: "2026-09-01" }] };
  }
  async getLoadHistory(s: Subject): Promise<LoadHistory> {
    this.rec("load", s);
    return { items: [{ day: "2026-11-20", tss: 62, ctl: 41, atl: 55, tsb: -14 }] };
  }
  async getRecentActivities(s: Subject): Promise<Activities> {
    this.rec("activities", s);
    return this.activities;
  }
  async getPlan(s: Subject): Promise<Plan> {
    this.rec("plan", s);
    return this.plan;
  }
  async getCoachLink(s: Subject): Promise<CoachLink> {
    this.rec("link", s);
    return this.link;
  }
}

export class FakeValidator implements PlanValidatorPort {
  verdicts: ValidationResult[] = [];
  calls = 0;
  async validate(_s: Subject, _id: string, _d: ProposalDraft): Promise<ValidationResult> {
    this.calls++;
    return this.verdicts.shift() ?? { valid: true, violations: [] };
  }
}

export class FakeApplier implements PlanApplierPort {
  applied: Array<{ approver: Principal; proposal: Proposal }> = [];
  async apply(approver: Principal, proposal: Proposal): Promise<void> {
    this.applied.push({ approver, proposal });
  }
}

export class ScriptedLlm implements LlmPort {
  requests: LlmRequest[] = [];
  constructor(public replies: Array<string | object>) {}
  async complete(req: LlmRequest) {
    this.requests.push(req);
    const next = this.replies.shift() ?? { outcome: "answer", message: "ok" };
    return { text: typeof next === "string" ? next : JSON.stringify(next), usage: { inputTokens: 500, outputTokens: 100 } };
  }
}

export const NOW = new Date("2026-11-22T12:00:00Z");
export const graphConfig: GraphConfig = { maxLlmCalls: 3, maxOutputTokens: 400, includeImportedText: false, loadDays: 28 };

export async function makeEnv(opts: { replies?: Array<string | object>; budget?: number; graph?: Partial<GraphConfig> } = {}) {
  const redis = new RedisMock() as unknown as Redis;
  await redis.flushall();
  const clock = { now: () => NOW };
  const data = new FakeData();
  const validator = new FakeValidator();
  const applier = new FakeApplier();
  const llm = new ScriptedLlm(opts.replies ?? []);
  const proposals = new RedisProposalStore(redis, 3600);
  const service = new CoachService({
    data, validator, proposals, llm, clock,
    budget: new RedisBudget(redis, opts.budget ?? 100_000, clock),
    graph: { ...graphConfig, ...opts.graph },
    proposalTtlHours: 48,
  });
  const approvals = new ApprovalService({ data, validator, applier, proposals, clock });
  return { redis, data, validator, applier, llm, proposals, service, approvals, clock };
}
