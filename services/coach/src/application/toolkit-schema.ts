import { z } from "zod";
import type { Proposal } from "../domain/proposal.js";
import type { ProposeResult } from "./toolkit.js";

/** Shape of the propose_adjustment tool answer as seen by the graph. The proposal itself was produced by our own toolkit. */
export const ProposalSchemaLoose = z.custom<ProposeResult>((v) => {
  const r = v as { status?: string; proposal?: Proposal; violations?: unknown };
  return (r.status === "stored" && !!r.proposal) || (r.status === "rejected" && Array.isArray(r.violations));
});
