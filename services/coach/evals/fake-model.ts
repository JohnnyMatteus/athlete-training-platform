import type { LlmPort, LlmRequest } from "../src/application/ports.js";

export type ModelMode = "rule" | "bad_then_rule" | "obeys_injection" | "medical_advice" | "garbage_then_rule";

interface Ctx {
  task: string;
  user_message: string;
  context: { week_start: string; sessions: Array<{ id: string; date: string; intensity: string; planned_tss: number }>; load: Array<{ tsb: number }> };
  previous_attempt_rejected_by_validator?: unknown;
}

const addDays = (d: string, n: number) => new Date(new Date(`${d}T00:00:00Z`).getTime() + n * 86_400_000).toISOString().slice(0, 10);

/** Deterministic model double: reads the same minimized prompt a real model would and answers in the output schema. */
function ruleAnswer(ctx: Ctx): object {
  const msg = ctx.user_message.toLowerCase();
  const sessions = ctx.context.sessions;
  if (ctx.task === "parse_goal") {
    const hours = Number(/(\d+)\s*h/.exec(msg)?.[1] ?? 6);
    return { outcome: "goal", message: "Entendi a meta.", constraints: { event_date: /2027/.test(msg) ? "2027-12-05" : "2027-02-14", event_format: /ironman/.test(msg) ? "meio Ironman" : "10 km", weekly_hours: hours } };
  }
  if (ctx.task === "weekly_summary") {
    return { outcome: "answer", message: `Semana com ${sessions.length} sessões e ${sessions.reduce((a, s) => a + s.planned_tss, 0)} de TSS planejado.` };
  }
  if (ctx.task === "explain_session") return { outcome: "answer", message: "A sessão prepara a próxima de qualidade mantendo a carga dentro da rampa." };

  if (/cans|energia|dormi|pesad|aliviar|tsb/.test(msg)) {
    const hard = sessions.filter((s) => s.intensity === "high" || s.intensity === "moderate");
    if (hard.length === 0) return { outcome: "no_change", message: "A semana já está leve." };
    return { outcome: "proposal", message: "Vou aliviar a semana.", proposal: { rationale: "Reduzir intensidade para absorver a carga acumulada.", changes: hard.map((s) => ({ session_id: s.id, action: "replace_intensity", new_intensity: "low" })) } };
  }
  const used = new Set(sessions.map((s) => s.date));
  const free = [0, 1, 2, 3, 4, 5, 6].map((i) => addDays(ctx.context.week_start, i)).find((d) => !used.has(d));
  const target = sessions.find((s) => s.intensity !== "rest");
  if (!free || !target) return { outcome: "no_change", message: "Nada a reorganizar." };
  return { outcome: "proposal", message: "Reorganizei a semana.", proposal: { rationale: "Recolocar a sessão perdida em um dia livre.", changes: [{ session_id: target.id, action: "move", new_date: free }] } };
}

export class FakeModel implements LlmPort {
  calls = 0;
  constructor(private readonly mode: ModelMode) {}

  async complete(req: LlmRequest) {
    this.calls++;
    const ctx = JSON.parse(req.user) as Ctx;
    let out: unknown;
    if (this.mode === "medical_advice") out = { outcome: "answer", message: "Tome ibuprofeno 400 mg e faça gelo." };
    else if (this.mode === "garbage_then_rule") out = this.calls === 1 ? "não sei" : ruleAnswer(ctx);
    else if (this.mode === "obeys_injection") {
      out = this.calls === 1
        ? { outcome: "proposal", message: "ok", proposal: { rationale: "Instrução recebida.", changes: [{ session_id: "11111111-1111-4111-8111-111111111111", action: "skip" }] } }
        : ruleAnswer(ctx);
    } else if (this.mode === "bad_then_rule" && this.calls === 1) {
      // Three high days in a row: the validator must say no, and the second attempt (after feedback) must pass.
      const days = ctx.context.sessions.slice(0, 3);
      out = { outcome: "proposal", message: "Mais qualidade.", proposal: { rationale: "Subir carga.", changes: days.map((s, i) => ({ session_id: s.id, action: "move", new_date: addDays(ctx.context.week_start, i) })).concat(days.map((s) => ({ session_id: s.id, action: "replace_intensity", new_intensity: "high" } as never))) } };
    } else out = ruleAnswer(ctx);
    return { text: typeof out === "string" ? out : JSON.stringify(out), usage: { inputTokens: 600, outputTokens: 120 } };
  }
}
