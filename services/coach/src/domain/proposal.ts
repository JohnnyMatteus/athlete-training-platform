import { z } from "zod";
import { IntensitySchema } from "./athlete-data.js";

const day = z.string().regex(/^\d{4}-\d{2}-\d{2}$/);

export const ChangeSchema = z
  .object({
    session_id: z.uuid(),
    action: z.enum(["move", "replace_intensity", "skip"]),
    new_date: day.optional(),
    new_intensity: IntensitySchema.optional(),
  })
  .superRefine((c, ctx) => {
    const field = { move: "new_date", replace_intensity: "new_intensity", skip: undefined }[c.action] as
      | "new_date"
      | "new_intensity"
      | undefined;
    if (field && c[field] === undefined) ctx.addIssue({ code: "custom", message: `${c.action} requires ${field}` });
  });
export type Change = z.infer<typeof ChangeSchema>;

export const ProposalDraftSchema = z.object({
  rationale: z.string().min(1).max(500),
  changes: z.array(ChangeSchema).min(1).max(7),
});
export type ProposalDraft = z.infer<typeof ProposalDraftSchema>;

export type ProposalStatus =
  | "pending_athlete"
  | "pending_coach"
  | "applied"
  | "rejected"
  | "expired";

export interface Proposal extends ProposalDraft {
  readonly id: string;
  readonly athleteId: string;
  readonly planId: string;
  readonly requestedBy: string;
  readonly status: ProposalStatus;
  readonly createdAt: string;
  readonly expiresAt: string;
  readonly decidedBy?: string;
  readonly decidedAt?: string;
}

export const GoalConstraintsSchema = z.object({
  event_date: day,
  event_format: z.string().min(1).max(60),
  weekly_hours: z.number().min(1).max(40),
});

/** What the model is allowed to say. Anything else is rejected before it reaches a user or a port. */
export const ModelOutputSchema = z.discriminatedUnion("outcome", [
  z.object({ outcome: z.literal("proposal"), message: z.string().max(600), proposal: ProposalDraftSchema }),
  z.object({ outcome: z.literal("no_change"), message: z.string().max(600) }),
  z.object({ outcome: z.literal("answer"), message: z.string().max(1500) }),
  z.object({ outcome: z.literal("goal"), message: z.string().max(600), constraints: GoalConstraintsSchema }),
]);
export type ModelOutput = z.infer<typeof ModelOutputSchema>;

export type Intent = "adjust_week" | "explain_session" | "weekly_summary" | "parse_goal";

export const ALLOWED_OUTCOMES: Record<Intent, ReadonlyArray<ModelOutput["outcome"]>> = {
  adjust_week: ["proposal", "no_change"],
  explain_session: ["answer"],
  weekly_summary: ["answer"],
  parse_goal: ["goal"],
};
