import type { Activities, LoadHistory, Plan, Profile } from "./athlete-data.js";
import { sanitizeImportedText } from "./untrusted.js";

/** Everything the LLM may see. Built from an allowlist: no ids, no heart-rate absolutes, no sex, no raw samples (ADR 0008). */
export interface LlmContext {
  readonly week_start: string;
  readonly thresholds_present: string[];
  readonly load: Array<{ day: string; tss: number; ctl: number; atl: number; tsb: number }>;
  readonly recent_activities: Array<{ type: string; day: string; tss: number; imported_note: string }>;
  readonly sessions: Array<{ id: string; date: string; sport: string; name: string; intensity: string; duration_min: number; planned_tss: number }>;
}

export interface RawContext {
  profile: Profile;
  load: LoadHistory;
  activities: Activities;
  plan: Plan;
}

export function minimizeForLlm(raw: RawContext, opts: { includeImportedText: boolean; loadDays: number }): LlmContext {
  return {
    week_start: raw.plan.week_start,
    thresholds_present: [...new Set(raw.profile.thresholds.map((t) => t.type))],
    load: raw.load.items.slice(-opts.loadDays).map(({ day, tss, ctl, atl, tsb }) => ({ day, tss, ctl, atl, tsb })),
    recent_activities: raw.activities.items.slice(0, 10).map((a) => ({
      type: a.type,
      day: a.day,
      tss: a.tss,
      imported_note: opts.includeImportedText ? sanitizeImportedText(`${a.title ?? ""} ${a.notes ?? ""}`) : "",
    })),
    sessions: raw.plan.sessions.map(({ id, date, sport, name, intensity, duration_min, planned_tss }) => ({
      id, date, sport, name: sanitizeImportedText(name), intensity, duration_min, planned_tss,
    })),
  };
}

/** Text fields from imports that mention pain or symptoms are a health signal even though they are data. */
export function importedTexts(activities: Activities): string[] {
  return activities.items.flatMap((a) => [a.title ?? "", a.notes ?? ""]).filter(Boolean);
}
