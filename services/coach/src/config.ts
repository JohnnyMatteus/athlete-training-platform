import { z } from "zod";

const Env = z.object({
  COACH_PORT: z.coerce.number().int().default(8130),
  REDIS_URL: z.string().min(1),
  JWT_PUBLIC_KEY_FILE: z.string().min(1),
  JWT_ISSUER: z.string().min(1),
  JWT_AUDIENCE: z.string().min(1),
  METRICS_URL: z.url(),
  PLANNING_URL: z.url(),
  UPSTREAM_TIMEOUT_MS: z.coerce.number().int().default(5000),
  ANTHROPIC_API_KEY: z.string().min(1),
  COACH_LLM_MODEL: z.string().min(1),
  COACH_LLM_TIMEOUT_MS: z.coerce.number().int().default(30000),
  COACH_DAILY_TOKEN_BUDGET: z.coerce.number().int().positive().default(60000),
  COACH_MAX_LLM_CALLS: z.coerce.number().int().min(1).max(5).default(3),
  COACH_MAX_OUTPUT_TOKENS: z.coerce.number().int().min(100).max(4000).default(800),
  COACH_INCLUDE_IMPORTED_TEXT: z.enum(["true", "false"]).default("false"),
  COACH_LOAD_DAYS: z.coerce.number().int().min(7).max(90).default(28),
  COACH_PROPOSAL_TTL_HOURS: z.coerce.number().int().min(1).max(168).default(48),
});

export type Config = ReturnType<typeof loadConfig>;

export function loadConfig(env: NodeJS.ProcessEnv) {
  const e = Env.parse(env);
  return {
    port: e.COACH_PORT,
    redisUrl: e.REDIS_URL,
    jwt: { publicKeyFile: e.JWT_PUBLIC_KEY_FILE, issuer: e.JWT_ISSUER, audience: e.JWT_AUDIENCE },
    metricsUrl: e.METRICS_URL,
    planningUrl: e.PLANNING_URL,
    upstreamTimeoutMs: e.UPSTREAM_TIMEOUT_MS,
    anthropicApiKey: e.ANTHROPIC_API_KEY,
    model: e.COACH_LLM_MODEL,
    llmTimeoutMs: e.COACH_LLM_TIMEOUT_MS,
    dailyTokenBudget: e.COACH_DAILY_TOKEN_BUDGET,
    proposalTtlHours: e.COACH_PROPOSAL_TTL_HOURS,
    graph: {
      maxLlmCalls: e.COACH_MAX_LLM_CALLS,
      maxOutputTokens: e.COACH_MAX_OUTPUT_TOKENS,
      includeImportedText: e.COACH_INCLUDE_IMPORTED_TEXT === "true",
      loadDays: e.COACH_LOAD_DAYS,
    },
  };
}
