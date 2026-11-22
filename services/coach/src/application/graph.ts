import { Annotation, END, START, StateGraph } from "@langchain/langgraph";
import { ActivitiesSchema, LoadHistorySchema, PlanSchema, ProfileSchema } from "../domain/athlete-data.js";
import { containsMedicalAdvice, mentionsHealthIssue, STANDARD_HEALTH_RESPONSE } from "../domain/health-guard.js";
import { importedTexts, minimizeForLlm, type LlmContext, type RawContext } from "../domain/minimize.js";
import { ALLOWED_OUTCOMES, ModelOutputSchema, type GoalConstraintsSchema, type Intent, type Proposal } from "../domain/proposal.js";
import type { z } from "zod";
import { CoachError } from "./errors.js";
import { McpToolClient, ToolError } from "./mcp-client.js";
import type { BudgetPort, LlmPort } from "./ports.js";
import { buildUserPrompt, SYSTEM_PROMPT } from "./prompts.js";
import type { ProposeResult } from "./toolkit.js";

export interface GraphConfig {
  maxLlmCalls: number;
  maxOutputTokens: number;
  includeImportedText: boolean;
  loadDays: number;
}

export type CoachResult =
  | { kind: "blocked_health"; message: string }
  | { kind: "answer" | "no_change"; message: string }
  | { kind: "proposal"; message: string; proposal: Proposal }
  | { kind: "goal"; message: string; constraints: z.infer<typeof GoalConstraintsSchema> }
  | { kind: "no_valid_proposal"; message: string; violations: ReadonlyArray<{ rule: string; message: string }> }
  | { kind: "invalid_model_output"; message: string };

export interface GraphInput {
  intent: Intent;
  message: string;
  sessionId?: string;
}

const State = Annotation.Root({
  intent: Annotation<Intent>(),
  message: Annotation<string>(),
  sessionId: Annotation<string | undefined>(),
  raw: Annotation<RawContext | undefined>(),
  context: Annotation<LlmContext | undefined>(),
  sessionDetail: Annotation<unknown>(),
  calls: Annotation<number>(),
  violations: Annotation<ReadonlyArray<{ rule: string; message: string }>>(),
  pending: Annotation<{ draft: unknown; message: string } | null>(),
  result: Annotation<CoachResult | undefined>(),
});
type S = typeof State.State;

const MAX_INVALID_JSON = "Não consegui produzir uma resposta válida agora. Nenhuma alteração foi feita no seu plano.";
const NO_VALID = "Não consegui montar um ajuste que passe nas regras de planejamento. Nenhuma alteração foi feita no seu plano.";

function parseModelJson(text: string): unknown {
  const m = /\{[\s\S]*\}/.exec(text);
  if (!m) return undefined;
  try {
    return JSON.parse(m[0]);
  } catch {
    return undefined;
  }
}

async function tool<T>(mcp: McpToolClient, name: string, args: Record<string, unknown>, schema: z.ZodType<T>): Promise<T> {
  try {
    return schema.parse(await mcp.call(name, args));
  } catch (e) {
    if (e instanceof ToolError) {
      const body = safeJson(e.body);
      const code = (body?.error as CoachError["code"] | undefined) ?? "upstream";
      throw new CoachError(code, String(body?.message ?? e.body));
    }
    throw e;
  }
}
const safeJson = (s: string): Record<string, unknown> | undefined => {
  try {
    return JSON.parse(s) as Record<string, unknown>;
  } catch {
    return undefined;
  }
};

/**
 * screen -> gather -> draft -> (validate -> draft)* -> end.
 * Guardrails are nodes and edges, not prompt text: the health screen runs before any data or model call,
 * the model sees only the minimized context, and a proposal exists only if the validator port accepted it.
 */
export function buildCoachGraph(deps: { mcp: McpToolClient; llm: LlmPort; budget: BudgetPort; athleteId: string; config: GraphConfig }) {
  const { mcp, llm, budget, athleteId, config } = deps;

  const screen = (s: S): Partial<S> =>
    mentionsHealthIssue(s.message) ? { result: { kind: "blocked_health", message: STANDARD_HEALTH_RESPONSE } } : {};

  const gather = async (s: S): Promise<Partial<S>> => {
    const [profile, load, activities, plan] = await Promise.all([
      tool(mcp, "get_athlete_profile", {}, ProfileSchema),
      tool(mcp, "get_load_history", { days: config.loadDays }, LoadHistorySchema),
      tool(mcp, "get_recent_activities", { limit: 10 }, ActivitiesSchema),
      tool(mcp, "get_plan", {}, PlanSchema),
    ]);
    const raw: RawContext = { profile, load, activities, plan };
    const flagged = importedTexts(activities).some(mentionsHealthIssue);
    if (flagged && s.intent === "adjust_week") {
      return { raw, result: { kind: "blocked_health", message: STANDARD_HEALTH_RESPONSE } };
    }
    const sessionDetail = s.intent === "explain_session" && s.sessionId ? await mcp.call("explain_session", { session_id: s.sessionId }).catch((e: unknown) => {
      if (e instanceof ToolError) throw new CoachError("not_found", "session not found in the current plan");
      throw e;
    }) : undefined;
    return {
      raw,
      sessionDetail,
      context: minimizeForLlm(raw, { includeImportedText: config.includeImportedText, loadDays: config.loadDays }),
    };
  };

  const draft = async (s: S): Promise<Partial<S>> => {
    const system = SYSTEM_PROMPT;
    const user = buildUserPrompt({ intent: s.intent, message: s.message, context: s.context!, violations: s.violations, toolData: s.sessionDetail });
    const reserved = Math.ceil((system.length + user.length) / 3) + config.maxOutputTokens;
    await budget.reserve(athleteId, reserved);
    let res;
    try {
      res = await llm.complete({ system, user, maxOutputTokens: config.maxOutputTokens });
    } catch (e) {
      await budget.settle(athleteId, reserved, 0);
      throw e;
    }
    await budget.settle(athleteId, reserved, res.usage.inputTokens + res.usage.outputTokens);

    const calls = s.calls + 1;
    const parsed = ModelOutputSchema.safeParse(parseModelJson(res.text));
    if (!parsed.success || !ALLOWED_OUTCOMES[s.intent].includes(parsed.data.outcome)) {
      return calls >= config.maxLlmCalls
        ? { calls, result: { kind: "invalid_model_output", message: MAX_INVALID_JSON } }
        : { calls, pending: null };
    }
    const out = parsed.data;
    const text = out.outcome === "proposal" ? `${out.message} ${out.proposal.rationale}` : out.message;
    if (containsMedicalAdvice(text) || mentionsHealthIssue(text)) {
      return { calls, result: { kind: "blocked_health", message: STANDARD_HEALTH_RESPONSE } };
    }
    switch (out.outcome) {
      case "proposal":
        return { calls, pending: { draft: out.proposal, message: out.message } };
      case "goal":
        return { calls, result: { kind: "goal", message: out.message, constraints: out.constraints } };
      default:
        return { calls, result: { kind: out.outcome, message: out.message } };
    }
  };

  const validate = async (s: S): Promise<Partial<S>> => {
    const res = await mcp.call<ProposeResult>("propose_adjustment", s.pending!.draft as Record<string, unknown>);
    if (res.status === "stored") return { result: { kind: "proposal", message: s.pending!.message, proposal: res.proposal } };
    return s.calls >= config.maxLlmCalls
      ? { violations: res.violations, pending: null, result: { kind: "no_valid_proposal", message: NO_VALID, violations: res.violations } }
      : { violations: res.violations, pending: null };
  };

  const done = (s: S) => (s.result ? END : undefined);

  return new StateGraph(State)
    .addNode("screen", screen)
    .addNode("gather", gather)
    .addNode("draft", draft)
    .addNode("validate", validate)
    .addEdge(START, "screen")
    .addConditionalEdges("screen", (s) => done(s) ?? "gather", ["gather", END])
    .addConditionalEdges("gather", (s) => done(s) ?? "draft", ["draft", END])
    .addConditionalEdges("draft", (s) => done(s) ?? (s.pending ? "validate" : "draft"), ["validate", "draft", END])
    .addConditionalEdges("validate", (s) => done(s) ?? "draft", ["draft", END])
    .compile();
}

export async function runCoach(graph: ReturnType<typeof buildCoachGraph>, input: GraphInput): Promise<CoachResult> {
  const out = await graph.invoke({
    intent: input.intent,
    message: input.message,
    sessionId: input.sessionId,
    calls: 0,
    violations: [],
  });
  return out.result ?? { kind: "invalid_model_output", message: MAX_INVALID_JSON };
}
