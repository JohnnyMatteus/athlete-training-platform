import { readFileSync } from "node:fs";
import { createServer, type IncomingMessage, type Server } from "node:http";
import type { AddressInfo } from "node:net";
import { resolve } from "node:path";
import { parse } from "yaml";
import { afterAll, beforeAll, describe, expect, it } from "vitest";
import { HttpAthleteData, HttpPlanApplier, HttpPlanValidator } from "../src/infrastructure/http/adapters.js";
import { HttpJson } from "../src/infrastructure/http/client.js";
import { athlete, ATHLETE, coach, COACH, NOW, PLAN_ID, S_HARD, subjectOf } from "./support.js";

type Doc = { paths: Record<string, Record<string, any>>; components: { schemas: Record<string, any> } }; // eslint-disable-line @typescript-eslint/no-explicit-any
const load = (f: string): Doc => parse(readFileSync(resolve(import.meta.dirname, "../../../contracts/openapi", f), "utf8"));
const ports = load("coach-ports.yaml");
const planning = load("planejamento.yaml");
const metrics = load("metricas.yaml");

const template = (tpl: string) => new RegExp(`^${tpl.replace(/\{[^}]+\}/g, "[^/]+")}$`);

interface Seen { method: string; path: string; query: URLSearchParams; auth: string | undefined; body: unknown }
const seen: Seen[] = [];
let server: Server;
let base: string;

const readBody = async (req: IncomingMessage) => {
  const chunks: Buffer[] = [];
  for await (const c of req) chunks.push(c as Buffer);
  return chunks.length ? JSON.parse(Buffer.concat(chunks).toString()) : undefined;
};

beforeAll(async () => {
  // A stub that answers each route with the example published in coach-ports.yaml.
  server = createServer((req, res) => {
    void readBody(req).then((body) => {
      const url = new URL(req.url ?? "/", "http://x");
      seen.push({ method: req.method ?? "", path: url.pathname, query: url.searchParams, auth: req.headers.authorization, body });
      const route = Object.keys(ports.paths).find((p) => template(p).test(url.pathname));
      const example = route && ports.paths[route]?.[(req.method ?? "").toLowerCase()]?.responses?.["200"]?.content?.["application/json"]?.example;
      if (!example) return void res.writeHead(404).end();
      res.writeHead(200, { "content-type": "application/json" }).end(JSON.stringify(example));
    });
  });
  await new Promise<void>((ok) => server.listen(0, "127.0.0.1", ok));
  base = `http://127.0.0.1:${(server.address() as AddressInfo).port}`;
});
afterAll(() => void server.close());

const http = () => new HttpJson({ baseUrl: base, timeoutMs: 2000 });
const clock = { now: () => NOW };

describe("coach ports against the published examples", () => {
  const data = () => new HttpAthleteData(http(), http(), clock);

  it("translates the Portuguese wire model and forwards the caller's own token", async () => {
    seen.length = 0;
    const s = subjectOf(athlete());
    const profile = await data().getProfile(s);
    const load = await data().getLoadHistory(s, { from: "2026-11-01", to: "2026-11-22" });
    const acts = await data().getRecentActivities(s, 5);
    const plan = await data().getPlan(s);
    expect(profile.thresholds[0]).toEqual({ type: "ftp", value: 230, valid_since: "2026-09-01" });
    expect(load.items[0]?.tsb).toBe(-13.8);
    expect(acts.items[0]).toMatchObject({ type: "corrida", tss: 62.5 });
    expect(plan.plan_id).toBe(PLAN_ID);
    expect(plan.sessions.map((x) => [x.sport, x.intensity, x.duration_min])).toEqual([["run", "low", 50], ["bike", "high", 60]]);
    expect(plan.sessions[0]?.steps[0]).toEqual({ type: "warmup", objective: "tempo 600", target: "fc_pct_max 60-70" });
    expect(seen.every((r) => r.auth === `Bearer ${s.principal.token}`)).toBe(true);
    expect(seen.some((r) => r.query.has("atleta_id") && r.path.startsWith("/metricas"))).toBe(false);
  });

  it("a coach names the athlete on metrics calls", async () => {
    seen.length = 0;
    await data().getLoadHistory(subjectOf(coach()), { from: "2026-11-01", to: "2026-11-22" });
    expect(seen[0]?.query.get("atleta_id")).toBe(ATHLETE);
  });

  it("derives the link and consents from the categories", async () => {
    const link = await data().getCoachLink(subjectOf(coach(), ATHLETE));
    expect(link).toEqual({ coach_id: COACH, status: "active", consent: { view_data: true, approve_plans: true } });
    const none = await data().getCoachLink(subjectOf(coach("7d2a8c60-1f4b-4e0a-9b53-6a1e2f3c4d99"), ATHLETE));
    expect(none.status).toBe("none");
  });

  it("validates on its own no-effect route and applies on /ajustes, always as origem coach", async () => {
    seen.length = 0;
    const draft = { rationale: "cansaço", changes: [{ session_id: S_HARD, action: "replace_intensity" as const, new_intensity: "low" as const }] };
    const verdict = await new HttpPlanValidator(http()).validate(subjectOf(athlete()), PLAN_ID, draft);
    await new HttpPlanApplier(http()).apply(athlete(), { ...draft, id: "i", athleteId: ATHLETE, planId: PLAN_ID, requestedBy: ATHLETE, status: "pending_athlete", createdAt: "", expiresAt: "" });
    expect(verdict.valid).toBe(true);
    expect(seen[0]).toMatchObject({ method: "POST", path: `/planos/${PLAN_ID}/ajustes/validacao` });
    expect(seen[1]).toMatchObject({ method: "POST", path: `/planos/${PLAN_ID}/ajustes` });
    expect(seen[1]?.query.size).toBe(0);
    expect(seen[1]?.body).toMatchObject({ origem: "coach", alteracoes: [{ treino_id: S_HARD, acao: "alterar_intensidade", nova_intensidade: "baixa" }] });
  });

  it("every route the adapters called is declared in coach-ports.yaml", () => {
    for (const r of seen) {
      const route = Object.keys(ports.paths).find((p) => template(p).test(r.path));
      expect(route, r.path).toBeDefined();
      expect(ports.paths[route!]?.[r.method.toLowerCase()], `${r.method} ${r.path}`).toBeDefined();
    }
  });
});

const VALIDATION = "/planos/{plano_id}/ajustes/validacao";

describe("coach ports against the owner contracts", () => {
  const owners: Array<[string, Doc]> = [["metricas", metrics], ["planejamento", planning]];
  it("every route in coach-ports.yaml exists in the owner contract with the same method", () => {
    for (const [path, methods] of Object.entries(ports.paths)) {
      for (const method of Object.keys(methods)) {
        const found = owners.some(([, d]) => d.paths[path]?.[method]);
        expect(found, `${method.toUpperCase()} ${path}`).toBe(true);
      }
    }
  });

  it("enums the ACL maps are the owners' enums", () => {
    const s = planning.components.schemas;
    expect(s.Modality.enum).toEqual(["natacao", "ciclismo", "corrida", "forca"]);
    expect(s.Workout.properties.intensidade.enum).toEqual(["descanso", "baixa", "moderada", "alta"]);
    expect(s.Step.properties.tipo.enum).toEqual(["aquecimento", "trabalho", "recuperacao", "desaquecimento"]);
    expect(s.Link.properties.status.enum).toEqual(["pendente", "ativo", "revogado"]);
    expect(s.Link.properties.categorias.items.enum).toEqual(["treinos", "frequencia_cardiaca", "plano"]);
    const item = s.AdjustmentRequest.properties.alteracoes.items.properties;
    expect(item.acao.enum).toEqual(["mover", "alterar_intensidade", "remover"]);
    expect(s.AdjustmentRequest.properties.origem.enum).toContain("coach");
  });

  it("the no-effect validation route is the owner's: same body, AdjustmentResult", () => {
    const route = planning.paths[VALIDATION]?.post;
    expect(route.requestBody.content["application/json"].schema.$ref).toMatch(/AdjustmentRequest/);
    expect(route.responses["200"].content["application/json"].schema.$ref).toMatch(/AdjustmentResult/);
  });
});

describe("fail closed against a planning server that has only the current contract", () => {
  it("never writes before approval: validation 404s and no proposal is stored", async () => {
    let writes = 0;
    const owner = createServer((req, res) => {
      const url = new URL(req.url ?? "/", "http://x");
      // Like the real server today: unknown query parameters are ignored, and POST /ajustes validates AND applies.
      if (req.method === "POST" && url.pathname === `/planos/${PLAN_ID}/ajustes`) {
        writes++;
        return void res.writeHead(200, { "content-type": "application/json" }).end(JSON.stringify({ aceito: true, violacoes: [] }));
      }
      res.writeHead(404).end();
    });
    await new Promise<void>((ok) => owner.listen(0, "127.0.0.1", ok));
    const planningHttp = new HttpJson({ baseUrl: `http://127.0.0.1:${(owner.address() as AddressInfo).port}`, timeoutMs: 2000 });
    const { makeEnv } = await import("./support.js");
    const { CoachToolkit } = await import("../src/application/toolkit.js");
    const env = await makeEnv();
    const toolkit = new CoachToolkit(subjectOf(athlete()), {
      data: env.data, validator: new HttpPlanValidator(planningHttp), proposals: env.proposals, clock, proposalTtlHours: 48,
    });
    await expect(toolkit.proposeAdjustment({ rationale: "x", changes: [{ session_id: S_HARD, action: "skip" }] })).rejects.toMatchObject({ code: "upstream" });
    expect(writes).toBe(0);
    expect(await env.proposals.listByAthlete(ATHLETE)).toHaveLength(0);
    owner.close();
  });
});
