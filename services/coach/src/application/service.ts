import type { Principal, Subject } from "../domain/principal.js";
import type { Intent } from "../domain/proposal.js";
import { createCoachMcpServer } from "../interfaces/mcp/server.js";
import { forbidden } from "./errors.js";
import { buildCoachGraph, runCoach, type CoachResult, type GraphConfig } from "./graph.js";
import { McpToolClient } from "./mcp-client.js";
import type { AthleteDataPort, BudgetPort, Clock, LlmPort, PlanValidatorPort, ProposalStorePort } from "./ports.js";
import { CoachToolkit } from "./toolkit.js";

export interface CoachServiceDeps {
  data: AthleteDataPort;
  validator: PlanValidatorPort;
  proposals: ProposalStorePort;
  llm: LlmPort;
  budget: BudgetPort;
  clock: Clock;
  graph: GraphConfig;
  proposalTtlHours: number;
}

/** Resolves the subject of a request. A coach names the athlete; an athlete is always themself. */
export function resolveSubject(principal: Principal, requestedAthleteId?: string): Subject {
  if (principal.role === "athlete") {
    if (requestedAthleteId && requestedAthleteId !== principal.subject) throw forbidden("athlete can only act on own data");
    return { principal, athleteId: principal.subject };
  }
  if (!requestedAthleteId) throw forbidden("coach must name the athlete");
  return { principal, athleteId: requestedAthleteId };
}

export class CoachService {
  constructor(private readonly deps: CoachServiceDeps) {}

  toolkitFor(subject: Subject): CoachToolkit {
    return new CoachToolkit(subject, this.deps);
  }

  async ask(subject: Subject, input: { intent: Intent; message: string; sessionId?: string }): Promise<CoachResult> {
    const mcp = await McpToolClient.connect(createCoachMcpServer(this.toolkitFor(subject)));
    try {
      const graph = buildCoachGraph({ mcp, llm: this.deps.llm, budget: this.deps.budget, athleteId: subject.athleteId, config: this.deps.graph });
      return await runCoach(graph, input);
    } finally {
      await mcp.close();
    }
  }
}
