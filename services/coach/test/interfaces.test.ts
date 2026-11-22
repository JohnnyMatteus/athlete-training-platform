import { mkdtempSync, writeFileSync } from "node:fs";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { Client } from "@modelcontextprotocol/sdk/client/index.js";
import { StreamableHTTPClientTransport } from "@modelcontextprotocol/sdk/client/streamableHttp.js";
import { exportSPKI, generateKeyPair, SignJWT } from "jose";
import { describe, expect, it } from "vitest";
import { createCoachMcpServer } from "../src/interfaces/mcp/server.js";
import { McpToolClient } from "../src/application/mcp-client.js";
import { JwtAuthenticator, type Authenticator } from "../src/infrastructure/auth/jwt.js";
import { RedisBudget } from "../src/infrastructure/redis/stores.js";
import { buildApp } from "../src/interfaces/http/app.js";
import { athlete, ATHLETE, coach, makeEnv, NOW, OTHER_ATHLETE, S_HARD, subjectOf } from "./support.js";

const auth: Authenticator = {
  async authenticate(h) {
    if (h === "Bearer athlete") return athlete();
    if (h === "Bearer other") return athlete(OTHER_ATHLETE);
    if (h === "Bearer coach") return coach();
    return null;
  },
};
const json = { "content-type": "application/json" };

describe("MCP surface", () => {
  it("exposes read tools and propose_adjustment only: no plan write, no athlete id argument", async () => {
    const env = await makeEnv();
    const mcp = await McpToolClient.connect(createCoachMcpServer(env.service.toolkitFor(subjectOf(athlete()))));
    const { tools } = await mcp.client.listTools();
    expect(tools.map((t: { name: string }) => t.name).sort()).toEqual(
      ["explain_session", "get_athlete_profile", "get_load_history", "get_plan", "get_recent_activities", "propose_adjustment"],
    );
    expect(JSON.stringify(tools)).not.toMatch(/athlete_id|atleta_id/);
    await mcp.close();
  });

  it("propose_adjustment stores a pending proposal and leaves the plan untouched", async () => {
    const env = await makeEnv();
    const mcp = await McpToolClient.connect(createCoachMcpServer(env.service.toolkitFor(subjectOf(athlete()))));
    const res = await mcp.call<{ status: string }>("propose_adjustment", {
      rationale: "x", changes: [{ session_id: S_HARD, action: "skip" }],
    });
    expect(res.status).toBe("stored");
    expect(env.applier.applied).toHaveLength(0);
    await mcp.close();
  });

  it("works over Streamable HTTP with the caller's bearer token", async () => {
    const env = await makeEnv();
    const app = buildApp({ auth, service: env.service, approvals: env.approvals });
    await app.listen({ port: 0, host: "127.0.0.1" });
    const url = new URL(`http://127.0.0.1:${(app.server.address() as { port: number }).port}/mcp`);
    const client = new Client({ name: "t", version: "1" });
    await client.connect(new StreamableHTTPClientTransport(url, { requestInit: { headers: { authorization: "Bearer athlete" } } }));
    const res = await client.callTool({ name: "get_plan", arguments: {} });
    expect(JSON.stringify(res.content)).toContain(S_HARD);
    expect(env.data.calls.every((c) => c.token === athlete().token)).toBe(true);
    await client.close();
    await app.close();
  });
});

describe("HTTP API", () => {
  it("requires a token, except /health", async () => {
    const env = await makeEnv();
    const app = buildApp({ auth, service: env.service, approvals: env.approvals });
    expect((await app.inject({ url: "/health" })).statusCode).toBe(200);
    expect((await app.inject({ method: "POST", url: "/coach/messages", payload: {} })).statusCode).toBe(401);
  });

  it("asks, then approves explicitly; a stranger gets 404; budget maps to 429", async () => {
    const env = await makeEnv({ replies: [{ outcome: "proposal", message: "ok", proposal: { rationale: "r", changes: [{ session_id: S_HARD, action: "skip" }] } }] });
    const app = buildApp({ auth, service: env.service, approvals: env.approvals });
    const ask = await app.inject({ method: "POST", url: "/coach/messages", headers: { authorization: "Bearer athlete", ...json }, payload: { intent: "adjust_week", message: "reduza a semana" } });
    expect(ask.statusCode).toBe(200);
    const id = ask.json().proposal.id as string;
    expect(env.applier.applied).toHaveLength(0);

    const stranger = await app.inject({ method: "POST", url: `/coach/proposals/${id}/approve`, headers: { authorization: "Bearer other" } });
    expect(stranger.statusCode).toBe(404);
    const ok = await app.inject({ method: "POST", url: `/coach/proposals/${id}/approve`, headers: { authorization: "Bearer athlete" } });
    expect(ok.json().status).toBe("applied");

    const bad = await app.inject({ method: "POST", url: "/coach/messages", headers: { authorization: "Bearer athlete", ...json }, payload: { intent: "nope", message: "x" } });
    expect(bad.statusCode).toBe(400);
    const foreign = await app.inject({ method: "POST", url: "/coach/messages", headers: { authorization: "Bearer athlete", ...json }, payload: { intent: "weekly_summary", message: "x", athlete_id: OTHER_ATHLETE } });
    expect(foreign.statusCode).toBe(403);

    const poor = await makeEnv({ budget: 10 });
    const app2 = buildApp({ auth, service: poor.service, approvals: poor.approvals });
    const r = await app2.inject({ method: "POST", url: "/coach/messages", headers: { authorization: "Bearer athlete", ...json }, payload: { intent: "weekly_summary", message: "x" } });
    expect(r.statusCode).toBe(429);
  });
});

describe("JWT authenticator (ADR 0011 contract)", () => {
  async function setup() {
    const { publicKey, privateKey } = await generateKeyPair("RS256");
    const dir = mkdtempSync(join(tmpdir(), "coach-jwt-"));
    const file = join(dir, "pub.pem");
    writeFileSync(file, await exportSPKI(publicKey));
    const authn = new JwtAuthenticator({ publicKeyFile: file, issuer: "https://idp.example.test/", audience: "athlete-api" });
    const sign = (claims: Record<string, unknown>, o: { iss?: string; aud?: string; sub?: string } = {}) =>
      new SignJWT(claims).setProtectedHeader({ alg: "RS256" }).setIssuer(o.iss ?? "https://idp.example.test/").setAudience(o.aud ?? "athlete-api")
        .setSubject(o.sub ?? ATHLETE).setExpirationTime("5m").sign(privateKey);
    return { authn, sign };
  }

  it("maps sub and role; rejects wrong issuer, audience, subject and garbage", async () => {
    const { authn, sign } = await setup();
    expect(await authn.authenticate(`Bearer ${await sign({})}`)).toMatchObject({ subject: ATHLETE, role: "athlete" });
    expect(await authn.authenticate(`Bearer ${await sign({ roles: ["treinador"] })}`)).toMatchObject({ role: "coach" });
    expect(await authn.authenticate(`Bearer ${await sign({}, { iss: "https://evil/" })}`)).toBeNull();
    expect(await authn.authenticate(`Bearer ${await sign({}, { aud: "other" })}`)).toBeNull();
    expect(await authn.authenticate(`Bearer ${await sign({}, { sub: "not-a-uuid" })}`)).toBeNull();
    expect(await authn.authenticate(`Bearer ${await sign({ roles: ["admin"] })}`)).toBeNull();
    expect(await authn.authenticate("Bearer garbage")).toBeNull();
    expect(await authn.authenticate(undefined)).toBeNull();
  });
});

describe("daily budget", () => {
  it("is per athlete and per day, and refunds what was not used", async () => {
    const env = await makeEnv();
    let now = NOW;
    const budget = new RedisBudget(env.redis, 1000, { now: () => now });
    await budget.reserve("a", 600);
    await budget.settle("a", 600, 100);
    await budget.reserve("a", 800);
    await expect(budget.reserve("a", 200)).rejects.toMatchObject({ code: "budget_exceeded" });
    await budget.reserve("b", 900);
    now = new Date(NOW.getTime() + 86_400_000);
    await budget.reserve("a", 900);
  });

  it("proposal lock refuses a concurrent decision", async () => {
    const env = await makeEnv();
    await expect(env.proposals.withLock("x", () => env.proposals.withLock("x", async () => 1))).rejects.toMatchObject({ code: "conflict" });
  });
});
