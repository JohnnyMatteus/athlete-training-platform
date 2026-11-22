import type { PlannedStep, PlannedWorkoutDTO, Target } from '@atp/capacitor-workoutkit';
import type { Plan, PlanStep, PlanWorkout } from '@/api/planejamento';
import type { Profile, Threshold } from '@/api/metricas';
import type { ScheduledItem } from '@/api/plannedWorkout';

export interface Limits {
  ftpW?: number;
  paceLimiarSKm?: number;
  cssMs?: number;
  fcMax?: number;
}

// Limiar vigente mais recente de cada tipo, mais a FC máxima do perfil.
export function limitsFrom(thresholds: Threshold[], profile?: Profile | null): Limits {
  const latest = (tipo: Threshold['tipo']) =>
    thresholds.filter((t) => t.tipo === tipo).sort((a, b) => b.vigente_desde.localeCompare(a.vigente_desde))[0]?.valor;
  return { ftpW: latest('ftp'), paceLimiarSKm: latest('pace_limiar'), cssMs: latest('css'), fcMax: profile?.fc_max };
}

const round = (v: number) => Math.round(v * 100) / 100;

/*
 * O plano fala em % do limiar; o relógio precisa de valores absolutos. Sem o limiar (ou com RPE, que não
 * tem alvo mensurável) o passo fica sem alvo, a mesma regra do export .fit do servidor.
 * Premissa a confirmar com o autor: "pace_pct_limiar" e "css_pct" são % da VELOCIDADE de limiar.
 */
function targetOf(step: PlanStep, workout: PlanWorkout, l: Limits): Target | undefined {
  const a = step.alvo;
  if (!a) return undefined;
  const range = (thr: number) => ({ min: round((thr * a.baixo) / 100), max: round((thr * a.alto) / 100) });
  switch (a.tipo) {
    case 'potencia_pct_ftp':
      return l.ftpW && workout.modalidade === 'ciclismo' ? { tipo: 'potencia_w', ...range(l.ftpW) } : undefined;
    case 'pace_pct_limiar':
      return l.paceLimiarSKm ? { tipo: 'velocidade_ms', ...range(1000 / l.paceLimiarSKm) } : undefined;
    case 'css_pct':
      return l.cssMs ? { tipo: 'velocidade_ms', ...range(l.cssMs) } : undefined;
    case 'fc_pct_max':
      return l.fcMax ? { tipo: 'fc_bpm', ...range(l.fcMax) } : undefined;
    case 'rpe':
      return undefined;
  }
}

function stepOf(s: PlanStep, w: PlanWorkout, l: Limits, tipo = s.tipo): PlannedStep {
  const valor = s.objetivo.valor;
  const objetivo =
    s.objetivo.tipo === 'tempo' && valor
      ? ({ tipo: 'tempo', segundos: Math.round(valor) } as const)
      : s.objetivo.tipo === 'distancia' && valor
        ? ({ tipo: 'distancia', metros: valor } as const)
        : ({ tipo: 'aberto' } as const);
  const alvo = targetOf(s, w, l);
  return { tipo, objetivo, ...(alvo ? { alvo } : {}) };
}

export function toPlannedWorkout(w: PlanWorkout, l: Limits): PlannedWorkoutDTO {
  const blocos = w.blocos.map((b) => ({ repeticoes: b.repeticoes, passos: [...b.passos] }));
  let aquecimento: PlanStep | undefined;
  let desaquecimento: PlanStep | undefined;

  const first = blocos[0];
  if (first && first.repeticoes === 1 && first.passos[0]?.tipo === 'aquecimento') aquecimento = first.passos.shift();
  const last = blocos[blocos.length - 1];
  if (last && last.repeticoes === 1 && last.passos[last.passos.length - 1]?.tipo === 'desaquecimento') {
    desaquecimento = last.passos.pop();
  }

  const modalidadeOk = w.modalidade;
  return {
    id: w.id,
    nome: w.nome,
    modalidade: modalidadeOk,
    ...(aquecimento ? { aquecimento: stepOf(aquecimento, w, l) } : {}),
    blocos: blocos
      .filter((b) => b.passos.length > 0)
      .map((b) => ({
        repeticoes: b.repeticoes,
        // Aquecimento/desaquecimento no meio do treino não existem no WorkoutKit: viram recuperação.
        passos: b.passos.map((s) =>
          stepOf(s, w, l, s.tipo === 'aquecimento' || s.tipo === 'desaquecimento' ? 'recuperacao' : s.tipo),
        ),
      })),
    ...(desaquecimento ? { desaquecimento: stepOf(desaquecimento, w, l) } : {}),
  };
}

// Só plano ativo vai ao relógio: plano aguardando aprovação do treinador não sai do servidor (ADR 0005).
export function scheduleFromPlan(plan: Plan, l: Limits, time = '07:00'): ScheduledItem[] {
  if (plan.status !== 'ativo') return [];
  return plan.mesociclos
    .flatMap((m) => m.microciclos.flatMap((c) => c.treinos))
    .filter((w) => w.intensidade !== 'descanso' && w.blocos.length > 0)
    .map((w) => ({ id: w.id, data: `${w.data}T${time}`, plano: toPlannedWorkout(w, l) }));
}
