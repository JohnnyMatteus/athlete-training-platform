import { randomUUID } from "node:crypto";
import type { Activities, CoachLink, LoadHistory, Plan, PlannedSession, Profile } from "../domain/athlete-data.js";
import type { Proposal, ProposalDraft } from "../domain/proposal.js";
import type { Subject } from "../domain/principal.js";
import { forbidden, notFound } from "./errors.js";
import type { AthleteDataPort, Clock, PlanValidatorPort, ProposalStorePort } from "./ports.js";

export type ProposeResult =
  | { status: "stored"; proposal: Proposal }
  | { status: "rejected"; violations: ReadonlyArray<{ rule: string; message: string }> }

export interface ToolkitDeps {
  data: AthleteDataPort;
  validator: PlanValidatorPort;
  proposals: ProposalStorePort;
  clock: Clock;
  proposalTtlHours: number;
}

/**
 * The operations behind the MCP tools, bound to one subject. The LLM never chooses the athlete:
 * the id and the token come from the authenticated request. There is no write into the plan here;
 * proposeAdjustment only stores a pending proposal (ADR 0005).
 */
export class CoachToolkit {
  private link: Promise<CoachLink> | undefined;

  constructor(
    private readonly subject: Subject,
    private readonly deps: ToolkitDeps,
  ) {}

  /** A coach reads an athlete only through an active link with the view_data consent. */
  async coachLink(): Promise<CoachLink> {
    this.link ??= this.deps.data.getCoachLink(this.subject);
    return this.link;
  }

  private async assertCanRead(): Promise<void> {
    const { principal, athleteId } = this.subject;
    if (principal.role === "athlete") {
      if (principal.subject !== athleteId) throw forbidden("athlete can only access own data");
      return;
    }
    const link = await this.coachLink();
    if (link.status !== "active" || link.coach_id !== principal.subject || !link.consent.view_data) {
      throw forbidden("no active consented link with this athlete");
    }
  }

  async getAthleteProfile(): Promise<Profile> {
    await this.assertCanRead();
    return this.deps.data.getProfile(this.subject);
  }

  async getLoadHistory(days: number): Promise<LoadHistory> {
    await this.assertCanRead();
    const to = this.deps.clock.now();
    const from = new Date(to.getTime() - days * 86_400_000);
    return this.deps.data.getLoadHistory(this.subject, { from: iso(from), to: iso(to) });
  }

  async getRecentActivities(limit: number): Promise<Activities> {
    await this.assertCanRead();
    return this.deps.data.getRecentActivities(this.subject, limit);
  }

  async getPlan(weekStart?: string): Promise<Plan> {
    await this.assertCanRead();
    return this.deps.data.getPlan(this.subject, weekStart);
  }

  async explainSession(sessionId: string): Promise<{ session: PlannedSession; week_start: string; week_total_tss: number }> {
    const plan = await this.getPlan();
    const session = plan.sessions.find((x) => x.id === sessionId);
    if (!session) throw notFound(`session ${sessionId} is not in the current plan`);
    return {
      session,
      week_start: plan.week_start,
      week_total_tss: plan.sessions.reduce((sum, x) => sum + x.planned_tss, 0),
    };
  }

  async proposeAdjustment(draft: ProposalDraft): Promise<ProposeResult> {
    await this.assertCanRead();
    const plan = await this.deps.data.getPlan(this.subject);

    const known = new Set(plan.sessions.map((x) => x.id));
    const unknown = draft.changes.find((c) => !known.has(c.session_id));
    if (unknown) {
      return { status: "rejected", violations: [{ rule: "unknown_session", message: `session ${unknown.session_id} is not in the plan` }] };
    }

    const verdict = await this.deps.validator.validate(this.subject, plan.plan_id, draft);
    if (!verdict.valid) return { status: "rejected", violations: verdict.violations };

    const { principal } = this.subject;
    const now = this.deps.clock.now();
    const proposal: Proposal = {
      ...draft,
      id: randomUUID(),
      athleteId: this.subject.athleteId,
      planId: plan.plan_id,
      requestedBy: principal.subject,
      // Whoever asked is who approves. With coach mode on, the planning context then holds the plan for the coach (ADR 0024).
      status: principal.role === "coach" ? "pending_coach" : "pending_athlete",
      createdAt: now.toISOString(),
      expiresAt: new Date(now.getTime() + this.deps.proposalTtlHours * 3_600_000).toISOString(),
    };
    await this.deps.proposals.save(proposal);
    return { status: "stored", proposal };
  }
}

const iso = (d: Date) => d.toISOString().slice(0, 10);
