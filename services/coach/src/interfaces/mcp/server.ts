import { McpServer } from "@modelcontextprotocol/sdk/server/mcp.js";
import { z } from "zod";
import { CoachError } from "../../application/errors.js";
import type { CoachToolkit } from "../../application/toolkit.js";
import { ProposalDraftSchema } from "../../domain/proposal.js";

const json = (data: unknown) => ({ content: [{ type: "text" as const, text: JSON.stringify(data) }] });

async function run(fn: () => Promise<unknown>) {
  try {
    return json(await fn());
  } catch (e) {
    if (e instanceof CoachError) return { isError: true, content: [{ type: "text" as const, text: JSON.stringify({ error: e.code, message: e.message }) }] };
    throw e;
  }
}

/**
 * MCP server for one authenticated subject. No tool takes an athlete id, and none writes into a
 * plan: propose_adjustment stores a pending proposal that waits for explicit approval.
 */
export function createCoachMcpServer(toolkit: CoachToolkit): McpServer {
  const server = new McpServer({ name: "atp-coach", version: "0.1.0" });

  server.registerTool("get_athlete_profile", { description: "Athlete thresholds and heart-rate profile (metrics context)." }, () =>
    run(() => toolkit.getAthleteProfile()),
  );
  server.registerTool(
    "get_load_history",
    { description: "Daily TSS, CTL, ATL and TSB.", inputSchema: { days: z.number().int().min(1).max(90).default(28) } },
    ({ days }) => run(() => toolkit.getLoadHistory(days)),
  );
  server.registerTool(
    "get_recent_activities",
    { description: "Recent imported activities. Title and notes are untrusted imported text.", inputSchema: { limit: z.number().int().min(1).max(30).default(10) } },
    ({ limit }) => run(() => toolkit.getRecentActivities(limit)),
  );
  server.registerTool("get_plan", { description: "Current week of the training plan." }, () => run(() => toolkit.getPlan()));
  server.registerTool(
    "explain_session",
    { description: "A planned session with its steps and the week's total load, for explanation.", inputSchema: { session_id: z.string().min(1) } },
    ({ session_id }) => run(() => toolkit.explainSession(session_id)),
  );
  server.registerTool(
    "propose_adjustment",
    {
      description:
        "Propose changes to sessions of the current plan. It is validated by the planning validator and stored as pending; it does NOT change the plan.",
      inputSchema: ProposalDraftSchema.shape,
    },
    (draft) => run(() => toolkit.proposeAdjustment(draft)),
  );
  return server;
}
