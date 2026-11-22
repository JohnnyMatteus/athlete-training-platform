import type { Activities, CoachLink, LoadHistory, Plan, Profile } from "../domain/athlete-data.js";
import type { Proposal, ProposalDraft } from "../domain/proposal.js";
import type { Principal, Subject } from "../domain/principal.js";

/**
 * Read side of the other contexts, always over HTTP with the caller's token (ADR 0021).
 * Contract: contracts/openapi/coach-ports.yaml. The coach never opens another context's database.
 */
export interface AthleteDataPort {
  getProfile(s: Subject): Promise<Profile>;
  getLoadHistory(s: Subject, range: { from: string; to: string }): Promise<LoadHistory>;
  getRecentActivities(s: Subject, limit: number): Promise<Activities>;
  getPlan(s: Subject, weekStart?: string): Promise<Plan>;
  getCoachLink(s: Subject): Promise<CoachLink>;
}

export interface ValidationResult {
  readonly valid: boolean;
  readonly violations: ReadonlyArray<{ rule: string; message: string }>;
}

/** The deterministic validator lives in the planning context (ADR 0005); this is only its port. It must not persist anything (dry run). */
export interface PlanValidatorPort {
  validate(s: Subject, planId: string, draft: ProposalDraft): Promise<ValidationResult>;
}

/** The only write into a plan, called by the approval flow and never exposed as a tool. */
export interface PlanApplierPort {
  apply(approver: Principal, proposal: Proposal): Promise<void>;
}

export interface LlmRequest {
  readonly system: string;
  readonly user: string;
  readonly maxOutputTokens: number;
}
export interface LlmResponse {
  readonly text: string;
  readonly usage: { inputTokens: number; outputTokens: number };
}
export interface LlmPort {
  complete(req: LlmRequest): Promise<LlmResponse>;
}

/** Daily token cap per athlete (ADR 0023). reserve throws CoachError("budget_exceeded"). */
export interface BudgetPort {
  reserve(athleteId: string, tokens: number): Promise<void>;
  settle(athleteId: string, reserved: number, actual: number): Promise<void>;
}

export interface ProposalStorePort {
  save(p: Proposal): Promise<void>;
  get(id: string): Promise<Proposal | null>;
  listByAthlete(athleteId: string): Promise<Proposal[]>;
  /** Exclusive lock so two approvals of the same proposal cannot both apply it. */
  withLock<T>(id: string, fn: () => Promise<T>): Promise<T>;
}

export interface Clock {
  now(): Date;
}
