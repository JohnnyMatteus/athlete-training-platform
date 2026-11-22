import { CoachError } from "../../application/errors.js";
import type { AthleteDataPort, PlanApplierPort, PlanValidatorPort, ValidationResult } from "../../application/ports.js";
import type { Plan } from "../../domain/athlete-data.js";
import type { Principal, Subject } from "../../domain/principal.js";
import type { Proposal, ProposalDraft } from "../../domain/proposal.js";
import type { Clock } from "../../application/ports.js";
import type { HttpJson } from "./client.js";
import {
  INTENSITY_TO_WIRE, pickWeek, toWeekSessions, WireActivityLoads, WireAdjustmentResult, WireLinks, WireLoad, WirePlan,
  WirePlanSummaries, WireProfile, WireThresholds,
} from "./wire.js";

const enc = encodeURIComponent;

/** Metrics context (perfil, limiares, carga) and Planning (planos, vínculos). Contract subset: contracts/openapi/coach-ports.yaml. */
export class HttpAthleteData implements AthleteDataPort {
  constructor(private readonly metrics: HttpJson, private readonly planning: HttpJson, private readonly clock: Clock) {}

  // A coach names the athlete in the query; for an athlete the token already says who it is.
  private who(s: Subject, first = false) {
    return s.principal.role === "coach" ? `${first ? "?" : "&"}atleta_id=${enc(s.athleteId)}` : "";
  }

  async getProfile(s: Subject) {
    const t = s.principal.token;
    const [profile, thresholds] = await Promise.all([
      this.metrics.request(t, "GET", `/metricas/perfil${this.who(s, true)}`, WireProfile),
      this.metrics.request(t, "GET", `/metricas/limiares${this.who(s, true)}`, WireThresholds),
    ]);
    return {
      resting_hr: profile.fc_repouso,
      max_hr: profile.fc_max,
      thresholds: thresholds.itens.map((x) => ({ type: x.tipo, value: x.valor, valid_since: x.vigente_desde })),
    };
  }

  async getLoadHistory(s: Subject, r: { from: string; to: string }) {
    const res = await this.metrics.request(s.principal.token, "GET", `/metricas/carga?de=${r.from}&ate=${r.to}${this.who(s)}`, WireLoad);
    return { items: res.dias.map((d) => ({ day: d.dia, tss: d.tss, ctl: d.ctl, atl: d.atl, tsb: d.tsb })) };
  }

  async getRecentActivities(s: Subject, limit: number) {
    const to = this.clock.now();
    const from = new Date(to.getTime() - 14 * 86_400_000).toISOString().slice(0, 10);
    const res = await this.metrics.request(s.principal.token, "GET", `/metricas/atividades?de=${from}&ate=${to.toISOString().slice(0, 10)}${this.who(s)}`, WireActivityLoads);
    const items = res.itens
      .map((a) => ({ id: a.atividade_id, day: a.dia, type: a.tipo, method: a.metodo, tss: a.tss }))
      .sort((a, b) => b.day.localeCompare(a.day))
      .slice(0, limit);
    return { items };
  }

  async getPlan(s: Subject, weekStart?: string): Promise<Plan> {
    const t = s.principal.token;
    const list = await this.planning.request(t, "GET", `/planos?atleta_id=${enc(s.athleteId)}`, WirePlanSummaries);
    const current = list.itens.filter((p) => p.status !== "rejeitado").sort((a, b) => b.inicio.localeCompare(a.inicio))[0];
    if (!current) throw new CoachError("not_found", "athlete has no plan");
    const plan = await this.planning.request(t, "GET", `/planos/${enc(current.id)}`, WirePlan);
    const week = pickWeek(plan, this.clock.now().toISOString().slice(0, 10), weekStart);
    if (!week) throw new CoachError("not_found", "plan has no such week");
    return { plan_id: plan.id, week_start: week.inicio, sessions: toWeekSessions(week) };
  }

  async getCoachLink(s: Subject) {
    const res = await this.planning.request(s.principal.token, "GET", "/vinculos", WireLinks);
    const link = res.itens.find((l) => l.atleta_id === s.athleteId && (s.principal.role === "athlete" || l.treinador_id === s.principal.subject));
    if (!link) return { coach_id: null, status: "none" as const, consent: { view_data: false, approve_plans: false } };
    const status = ({ pendente: "pending", ativo: "active", revogado: "revoked" } as const)[link.status];
    return {
      coach_id: link.treinador_id,
      status,
      consent: { view_data: link.categorias.includes("treinos"), approve_plans: link.categorias.includes("plano") },
    };
  }
}

function adjustmentBody(d: ProposalDraft) {
  return {
    motivo: d.rationale,
    origem: "coach",
    relato_dor: false,
    alteracoes: d.changes.map((c) => ({
      treino_id: c.session_id,
      acao: { move: "mover", replace_intensity: "alterar_intensidade", skip: "remover" }[c.action],
      ...(c.new_date ? { nova_data: c.new_date } : {}),
      ...(c.new_intensity ? { nova_intensidade: INTENSITY_TO_WIRE[c.new_intensity] } : {}),
    })),
  };
}

/**
 * Validation goes through its own route, POST /planos/{id}/ajustes/validacao (same body and answer, no effect).
 * It is NOT in the planning contract yet. The write route is never used for validation, with or without a flag:
 * a server that ignores unknown query parameters would apply the change. While the route is missing the
 * answer is 404 and validation fails closed: no proposal is stored.
 */
export class HttpPlanValidator implements PlanValidatorPort {
  constructor(private readonly planning: HttpJson) {}
  async validate(s: Subject, planId: string, draft: ProposalDraft): Promise<ValidationResult> {
    try {
      const res = await this.planning.request(s.principal.token, "POST", `/planos/${enc(planId)}/ajustes/validacao`, WireAdjustmentResult, adjustmentBody(draft));
      return { valid: res.aceito, violations: res.violacoes.map((v) => ({ rule: v.regra, message: v.mensagem })) };
    } catch (e) {
      if (e instanceof CoachError && e.code === "not_found") throw new CoachError("upstream", "planning validation route is not available; no proposal was made");
      throw e;
    }
  }
}

/** The only write into a plan. Called after explicit approval; the planning context validates again and may hold the plan for the coach. */
export class HttpPlanApplier implements PlanApplierPort {
  constructor(private readonly planning: HttpJson) {}
  async apply(approver: Principal, p: Proposal): Promise<void> {
    const res = await this.planning.request(approver.token, "POST", `/planos/${enc(p.planId)}/ajustes`, WireAdjustmentResult, adjustmentBody(p));
    if (!res.aceito) throw new CoachError("conflict", `planning rejected the adjustment: ${res.violacoes.map((v) => v.regra).join(", ")}`);
  }
}
