import type { Principal } from "../domain/principal.js";
import type { Proposal } from "../domain/proposal.js";
import { conflict, forbidden, notFound } from "./errors.js";
import type { AthleteDataPort, Clock, PlanApplierPort, PlanValidatorPort, ProposalStorePort } from "./ports.js";

export interface ApprovalDeps {
  data: AthleteDataPort;
  validator: PlanValidatorPort;
  applier: PlanApplierPort;
  proposals: ProposalStorePort;
  clock: Clock;
}

/** Explicit approval before any plan change (ADR 0005, 0022). The model has no path into this class. */
export class ApprovalService {
  constructor(private readonly deps: ApprovalDeps) {}

  async list(principal: Principal, athleteId: string): Promise<Proposal[]> {
    await this.authorize(principal, athleteId, "read");
    return this.deps.proposals.listByAthlete(athleteId);
  }

  async approve(principal: Principal, proposalId: string): Promise<Proposal> {
    return this.deps.proposals.withLock(proposalId, async () => {
      const proposal = await this.load(proposalId);
      await this.authorize(principal, proposal.athleteId, proposal.status === "pending_coach" ? "approve_as_coach" : "approve_as_athlete");
      const open = await this.assertOpen(proposal);
      const subject = { principal, athleteId: proposal.athleteId };

      // The plan may have moved since the proposal was made, so the validator runs again right before the write.
      const plan = await this.deps.data.getPlan(subject);
      if (plan.plan_id !== proposal.planId) throw conflict("plan changed since the proposal");
      const verdict = await this.deps.validator.validate(subject, proposal.planId, proposal);
      if (!verdict.valid) throw conflict(`validator rejected at approval: ${verdict.violations.map((v) => v.rule).join(", ")}`);

      await this.deps.applier.apply(principal, open);
      const applied: Proposal = { ...open, status: "applied", decidedBy: principal.subject, decidedAt: this.deps.clock.now().toISOString() };
      await this.deps.proposals.save(applied);
      return applied;
    });
  }

  async reject(principal: Principal, proposalId: string): Promise<Proposal> {
    return this.deps.proposals.withLock(proposalId, async () => {
      const proposal = await this.load(proposalId);
      // The athlete owns the plan, so can always reject; a linked coach can reject what waits on them.
      await this.authorize(principal, proposal.athleteId, principal.role === "athlete" ? "approve_as_athlete" : "approve_as_coach");
      const open = await this.assertOpen(proposal);
      const rejected: Proposal = { ...open, status: "rejected", decidedBy: principal.subject, decidedAt: this.deps.clock.now().toISOString() };
      await this.deps.proposals.save(rejected);
      return rejected;
    });
  }

  private async load(id: string): Promise<Proposal> {
    const p = await this.deps.proposals.get(id);
    if (!p) throw notFound("proposal not found");
    return p;
  }

  private async assertOpen(p: Proposal): Promise<Proposal> {
    if (p.status !== "pending_athlete" && p.status !== "pending_coach") throw conflict(`proposal is ${p.status}`);
    if (new Date(p.expiresAt) <= this.deps.clock.now()) {
      await this.deps.proposals.save({ ...p, status: "expired" });
      throw conflict("proposal expired");
    }
    return p;
  }

  private async authorize(
    principal: Principal,
    athleteId: string,
    action: "read" | "approve_as_athlete" | "approve_as_coach",
  ): Promise<void> {
    // Unknown proposals and foreign athletes look the same to the caller (404), like the API (ADR 0011).
    if (principal.role === "athlete") {
      if (principal.subject !== athleteId) throw notFound("proposal not found");
      if (action === "approve_as_coach") throw forbidden("this proposal waits for the linked coach");
      return;
    }
    if (action === "approve_as_athlete") throw forbidden("only the athlete can approve this proposal");
    const link = await this.deps.data.getCoachLink({ principal, athleteId });
    const linked = link.status === "active" && link.coach_id === principal.subject;
    if (!linked) throw notFound("proposal not found");
    if (action === "read" && !link.consent.view_data) throw forbidden("athlete has not consented to data sharing");
    if (action === "approve_as_coach" && !link.consent.approve_plans) throw forbidden("athlete has not consented to plan approval by the coach");
  }
}
