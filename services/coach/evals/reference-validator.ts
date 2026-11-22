import type { PlanValidatorPort, ValidationResult } from "../src/application/ports.js";
import type { Plan } from "../src/domain/athlete-data.js";
import type { Subject } from "../src/domain/principal.js";
import type { ProposalDraft } from "../src/domain/proposal.js";

/**
 * Test-only stand-in for the planning validator (design doc 7.3). It exists so the evals can measure
 * "proposals accepted by the validator" without the planning service. It is NOT the production validator.
 */
export class ReferenceValidator implements PlanValidatorPort {
  constructor(private readonly plan: () => Plan, private readonly tsb: () => number) {}

  async validate(_s: Subject, _planId: string, draft: ProposalDraft): Promise<ValidationResult> {
    const plan = this.plan();
    const sessions = plan.sessions.map((x) => ({ ...x }));
    const violations: Array<{ rule: string; message: string }> = [];
    const start = new Date(`${plan.week_start}T00:00:00Z`).getTime();
    const end = start + 6 * 86_400_000;

    for (const c of draft.changes) {
      const sess = sessions.find((x) => x.id === c.session_id);
      if (!sess) continue;
      if (c.action === "move" && c.new_date) {
        const t = new Date(`${c.new_date}T00:00:00Z`).getTime();
        if (t < start || t > end) violations.push({ rule: "fora_da_semana", message: "data fora da semana" });
        sess.date = c.new_date;
      }
      if (c.action === "replace_intensity" && c.new_intensity) {
        if (c.new_intensity === "high" && this.tsb() < -25) violations.push({ rule: "tsb_minimo", message: "TSB abaixo de -25" });
        sess.intensity = c.new_intensity;
      }
      if (c.action === "skip") sess.intensity = "rest";
    }

    const high = new Set(sessions.filter((x) => x.intensity === "high").map((x) => x.date));
    let run = 0;
    for (let d = start; d <= end; d += 86_400_000) {
      run = high.has(new Date(d).toISOString().slice(0, 10)) ? run + 1 : 0;
      if (run > 2) {
        violations.push({ rule: "dias_alta_consecutivos", message: "mais de 2 dias seguidos de alta intensidade" });
        break;
      }
    }
    const days = new Set(sessions.filter((x) => x.intensity !== "rest").map((x) => x.date));
    if (days.size >= 7) violations.push({ rule: "sem_descanso", message: "sem dia de descanso" });
    const dates = sessions.map((x) => x.date);
    if (new Set(dates.filter((d, i) => dates.indexOf(d) !== i)).size > 0 && draft.changes.some((c) => c.action === "move")) {
      violations.push({ rule: "choque_de_data", message: "duas sessões no mesmo dia" });
    }
    return { valid: violations.length === 0, violations };
  }
}
