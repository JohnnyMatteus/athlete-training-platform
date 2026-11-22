import { z } from "zod";

const day = z.string().regex(/^\d{4}-\d{2}-\d{2}$/);

export const ProfileSchema = z.object({
  resting_hr: z.number().int(),
  max_hr: z.number().int(),
  thresholds: z.array(
    z.object({
      type: z.enum(["ftp", "pace_limiar", "css"]),
      value: z.number().positive(),
      valid_since: day,
    }),
  ),
});
export type Profile = z.infer<typeof ProfileSchema>;

export const LoadHistorySchema = z.object({
  items: z.array(
    z.object({ day, tss: z.number(), ctl: z.number(), atl: z.number(), tsb: z.number() }),
  ),
});
export type LoadHistory = z.infer<typeof LoadHistorySchema>;

export const ActivitiesSchema = z.object({
  items: z.array(
    z.object({
      id: z.uuid(),
      day,
      type: z.string(),
      method: z.string(),
      tss: z.number(),
      // No contract exposes imported free text today; if one does, it is untrusted by construction (ADR 0023).
      title: z.string().nullish(),
      notes: z.string().nullish(),
    }),
  ),
});
export type Activities = z.infer<typeof ActivitiesSchema>;

export const IntensitySchema = z.enum(["rest", "low", "moderate", "high"]);
export type Intensity = z.infer<typeof IntensitySchema>;

export const PlanSchema = z.object({
  plan_id: z.uuid(),
  week_start: day,
  sessions: z.array(
    z.object({
      id: z.uuid(),
      date: day,
      sport: z.enum(["swim", "bike", "run", "strength"]),
      name: z.string(),
      intensity: IntensitySchema,
      duration_min: z.number().int().nonnegative(),
      planned_tss: z.number().nonnegative(),
      steps: z
        .array(
          z.object({
            type: z.enum(["warmup", "work", "recovery", "cooldown"]),
            objective: z.string(),
            target: z.string(),
          }),
        )
        .default([]),
    }),
  ),
});
export type Plan = z.infer<typeof PlanSchema>;
export type PlannedSession = Plan["sessions"][number];

export const CoachLinkSchema = z.object({
  coach_id: z.uuid().nullable(),
  status: z.enum(["none", "pending", "active", "revoked"]),
  // Derived from the link categories: "treinos" lets the coach read, "plano" lets the coach approve plan changes.
  consent: z.object({ view_data: z.boolean(), approve_plans: z.boolean() }),
});
export type CoachLink = z.infer<typeof CoachLinkSchema>;
