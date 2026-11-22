import { describe, expect, it } from "vitest";
import { athlete, ATHLETE, coach, COACH, makeEnv, OTHER_ATHLETE, S_HARD, subjectOf } from "./support.js";

const draft = { rationale: "cansaço", changes: [{ session_id: S_HARD, action: "replace_intensity" as const, new_intensity: "low" as const }] };

async function pending(env: Awaited<ReturnType<typeof makeEnv>>, who = athlete(), athleteId = ATHLETE) {
  const r = await env.service.toolkitFor(subjectOf(who, athleteId)).proposeAdjustment(draft);
  if (r.status !== "stored") throw new Error("not stored");
  return r.proposal;
}
const activeLink = (env: Awaited<ReturnType<typeof makeEnv>>, consent = { view_data: true, approve_plans: true }) => {
  env.data.link = { coach_id: COACH, status: "active", consent };
};

describe("approval", () => {
  it("applies only after the athlete approves, with the athlete's token", async () => {
    const env = await makeEnv();
    const p = await pending(env);
    expect(env.applier.applied).toHaveLength(0);
    const done = await env.approvals.approve(athlete(), p.id);
    expect(done.status).toBe("applied");
    expect(env.applier.applied[0]?.approver.token).toBe(athlete().token);
  });

  it("cannot be approved twice", async () => {
    const env = await makeEnv();
    const p = await pending(env);
    await env.approvals.approve(athlete(), p.id);
    await expect(env.approvals.approve(athlete(), p.id)).rejects.toMatchObject({ code: "conflict" });
    expect(env.applier.applied).toHaveLength(1);
  });

  it("revalidates right before the write and refuses when the validator now objects", async () => {
    const env = await makeEnv();
    const p = await pending(env);
    env.validator.verdicts = [{ valid: false, violations: [{ rule: "tsb_minimo", message: "x" }] }];
    await expect(env.approvals.approve(athlete(), p.id)).rejects.toMatchObject({ code: "conflict" });
    expect(env.applier.applied).toHaveLength(0);
  });

  it("another athlete sees 404, not 403", async () => {
    const env = await makeEnv();
    const p = await pending(env);
    await expect(env.approvals.approve(athlete(OTHER_ATHLETE), p.id)).rejects.toMatchObject({ code: "not_found" });
  });

  it("expired proposals cannot be approved", async () => {
    const env = await makeEnv();
    const p = await pending(env);
    await env.proposals.save({ ...p, expiresAt: "2026-11-21T00:00:00Z" });
    await expect(env.approvals.approve(athlete(), p.id)).rejects.toMatchObject({ code: "conflict" });
    expect((await env.proposals.get(p.id))?.status).toBe("expired");
  });

  it("reject leaves the plan alone", async () => {
    const env = await makeEnv();
    const p = await pending(env);
    expect((await env.approvals.reject(athlete(), p.id)).status).toBe("rejected");
    expect(env.applier.applied).toHaveLength(0);
  });
});

describe("coach mode", () => {
  it("a coach reads and proposes only with an active, consented link", async () => {
    const env = await makeEnv();
    const kit = { getPlan: () => env.service.toolkitFor(subjectOf(coach())).getPlan(), getAthleteProfile: () => env.service.toolkitFor(subjectOf(coach())).getAthleteProfile() };
    await expect(kit.getPlan()).rejects.toMatchObject({ code: "forbidden" });
    env.data.link = { coach_id: COACH, status: "pending", consent: { view_data: true, approve_plans: true } };
    await expect(kit.getPlan()).rejects.toMatchObject({ code: "forbidden" });
    env.data.link = { coach_id: COACH, status: "revoked", consent: { view_data: true, approve_plans: true } };
    await expect(kit.getAthleteProfile()).rejects.toMatchObject({ code: "forbidden" });
    activeLink(env, { view_data: false, approve_plans: true });
    await expect(kit.getPlan()).rejects.toMatchObject({ code: "forbidden" });
    activeLink(env);
    await expect(kit.getPlan()).resolves.toBeDefined();
  });

  it("a link to a different coach grants nothing", async () => {
    const env = await makeEnv();
    env.data.link = { coach_id: "7d2a8c60-1f4b-4e0a-9b53-6a1e2f3c4d99", status: "active", consent: { view_data: true, approve_plans: true } };
    await expect(env.service.toolkitFor(subjectOf(coach())).getPlan()).rejects.toMatchObject({ code: "forbidden" });
  });

  it("a coach-requested proposal waits for the coach, not the athlete", async () => {
    const env = await makeEnv();
    activeLink(env);
    const p = await pending(env, coach());
    expect(p.status).toBe("pending_coach");
    await expect(env.approvals.approve(athlete(), p.id)).rejects.toMatchObject({ code: "forbidden" });
    const done = await env.approvals.approve(coach(), p.id);
    expect(done.status).toBe("applied");
    expect(env.applier.applied[0]?.approver.role).toBe("coach");
  });

  it("revoking the link or the plan consent stops the coach from approving", async () => {
    const env = await makeEnv();
    activeLink(env);
    const p = await pending(env, coach());
    env.data.link = { coach_id: COACH, status: "revoked", consent: { view_data: false, approve_plans: false } };
    await expect(env.approvals.approve(coach(), p.id)).rejects.toMatchObject({ code: "not_found" });
    activeLink(env, { view_data: true, approve_plans: false });
    await expect(env.approvals.approve(coach(), p.id)).rejects.toMatchObject({ code: "forbidden" });
    expect(env.applier.applied).toHaveLength(0);
  });

  it("a coach cannot approve an athlete's own proposal", async () => {
    const env = await makeEnv();
    activeLink(env);
    const p = await pending(env);
    await expect(env.approvals.approve(coach(), p.id)).rejects.toMatchObject({ code: "forbidden" });
  });

  it("the athlete can reject a coach-requested proposal", async () => {
    const env = await makeEnv();
    activeLink(env);
    const p = await pending(env, coach());
    expect((await env.approvals.reject(athlete(), p.id)).status).toBe("rejected");
  });

  it("coach AI calls use the coach's token against the athlete named in the request", async () => {
    const env = await makeEnv({ replies: [{ outcome: "answer", message: "Resumo da semana." }] });
    activeLink(env);
    await env.service.ask(subjectOf(coach()), { intent: "weekly_summary", message: "resuma" });
    const reads = env.data.calls.filter((c) => c.method !== "link");
    expect(reads.length).toBeGreaterThan(0);
    expect(reads.every((c) => c.token === coach().token && c.athleteId === ATHLETE)).toBe(true);
  });
});
