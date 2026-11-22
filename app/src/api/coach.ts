import { z } from 'zod';
import type { Http } from './http';

/*
 * Wire do serviço coach (services/coach/src/interfaces/http/app.ts, commit 66d7259). O coach ainda não
 * tem OpenAPI em contracts/: os schemas abaixo seguem o código e quebram o contrato do app se ele mudar.
 */
export const intentSchema = z.enum(['adjust_week', 'explain_session', 'weekly_summary', 'parse_goal']);
export type Intent = z.infer<typeof intentSchema>;

const changeSchema = z.object({
  session_id: z.uuid(),
  action: z.enum(['move', 'replace_intensity', 'skip']),
  new_date: z.string().optional(),
  new_intensity: z.string().optional(),
});

export const proposalSchema = z.object({
  id: z.uuid(),
  athleteId: z.uuid(),
  planId: z.string(),
  requestedBy: z.string(),
  status: z.enum(['pending_athlete', 'pending_coach', 'applied', 'rejected', 'expired']),
  createdAt: z.iso.datetime(),
  expiresAt: z.iso.datetime(),
  decidedBy: z.string().optional(),
  decidedAt: z.iso.datetime().optional(),
  rationale: z.string(),
  changes: z.array(changeSchema),
});
export type Proposal = z.infer<typeof proposalSchema>;

const violations = z.array(z.object({ rule: z.string(), message: z.string() }));

export const coachResultSchema = z.discriminatedUnion('kind', [
  z.object({ kind: z.literal('blocked_health'), message: z.string() }),
  z.object({ kind: z.enum(['answer', 'no_change']), message: z.string() }),
  z.object({ kind: z.literal('proposal'), message: z.string(), proposal: proposalSchema }),
  z.object({
    kind: z.literal('goal'),
    message: z.string(),
    constraints: z.object({ event_date: z.string(), event_format: z.string(), weekly_hours: z.number() }),
  }),
  z.object({ kind: z.literal('no_valid_proposal'), message: z.string(), violations }),
  z.object({ kind: z.literal('invalid_model_output'), message: z.string() }),
]);
export type CoachResult = z.infer<typeof coachResultSchema>;

export function createCoachApi(http: Http) {
  const id = (v: string) => encodeURIComponent(v);
  return {
    ask: (intent: Intent, message: string) =>
      http('/coach/messages', { method: 'POST', body: { intent, message }, schema: coachResultSchema }),
    proposals: () => http('/coach/proposals', { schema: z.object({ items: z.array(proposalSchema) }) }),
    approve: (pid: string) => http(`/coach/proposals/${id(pid)}/approve`, { method: 'POST', schema: proposalSchema }),
    reject: (pid: string) => http(`/coach/proposals/${id(pid)}/reject`, { method: 'POST', schema: proposalSchema }),
  };
}
export type CoachApi = ReturnType<typeof createCoachApi>;
