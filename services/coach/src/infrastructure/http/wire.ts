import { z } from "zod";

/**
 * Wire shapes of the other contexts (Portuguese field names) and their translation to the coach's own model.
 * This is the Anti-Corruption Layer (ADR 0002): nothing outside infrastructure/http sees these names.
 * Source contracts: planejamento.yaml, metricas.yaml; consumed subset in contracts/openapi/coach-ports.yaml.
 */
const day = z.string().regex(/^\d{4}-\d{2}-\d{2}$/);

export const WireProfile = z.object({ sexo: z.string(), fc_repouso: z.number().int(), fc_max: z.number().int() });
export const WireThresholds = z.object({
  itens: z.array(z.object({ tipo: z.enum(["ftp", "pace_limiar", "css"]), valor: z.number().positive(), vigente_desde: day })),
});
export const WireLoad = z.object({
  atleta_id: z.uuid(),
  dias: z.array(z.object({ dia: day, tss: z.number(), ctl: z.number(), atl: z.number(), tsb: z.number() })),
});
export const WireActivityLoads = z.object({
  itens: z.array(z.object({ atividade_id: z.uuid(), dia: day, tipo: z.string(), metodo: z.string(), tss: z.number() })),
});

const Modality = z.enum(["natacao", "ciclismo", "corrida", "forca"]);
const Step = z.object({
  tipo: z.enum(["aquecimento", "trabalho", "recuperacao", "desaquecimento"]),
  objetivo: z.object({ tipo: z.enum(["tempo", "distancia", "aberto"]), valor: z.number().optional() }),
  alvo: z.object({ tipo: z.string(), baixo: z.number(), alto: z.number() }).optional(),
});
const Workout = z.object({
  id: z.uuid(),
  data: day,
  modalidade: Modality,
  nome: z.string(),
  intensidade: z.enum(["descanso", "baixa", "moderada", "alta"]),
  duracao_s: z.number().int(),
  tss_planejado: z.number(),
  blocos: z.array(z.object({ repeticoes: z.number().int(), passos: z.array(Step) })),
});
export const WirePlanSummaries = z.object({
  itens: z.array(z.object({ id: z.uuid(), status: z.enum(["aguardando_aprovacao", "ativo", "rejeitado"]), inicio: day })),
});
export const WirePlan = z.object({
  id: z.uuid(),
  mesociclos: z.array(
    z.object({ microciclos: z.array(z.object({ semana: z.number().int(), inicio: day, treinos: z.array(Workout) })) }),
  ),
});
export const WireLinks = z.object({
  itens: z.array(
    z.object({
      atleta_id: z.uuid(),
      treinador_id: z.uuid(),
      status: z.enum(["pendente", "ativo", "revogado"]),
      categorias: z.array(z.enum(["treinos", "frequencia_cardiaca", "plano"])),
    }),
  ),
});
export const WireAdjustmentResult = z.object({
  aceito: z.boolean(),
  violacoes: z.array(z.object({ regra: z.string(), mensagem: z.string() })),
});

const SPORT = { natacao: "swim", ciclismo: "bike", corrida: "run", forca: "strength" } as const;
const INTENSITY = { descanso: "rest", baixa: "low", moderada: "moderate", alta: "high" } as const;
export const INTENSITY_TO_WIRE = { rest: "descanso", low: "baixa", moderate: "moderada", high: "alta" } as const;
const STEP = { aquecimento: "warmup", trabalho: "work", recuperacao: "recovery", desaquecimento: "cooldown" } as const;

export type WirePlanT = z.infer<typeof WirePlan>;

/** The microcycle that contains `today` (or `weekStart`), falling back to the next one that starts later. */
export function pickWeek(plan: WirePlanT, today: string, weekStart?: string) {
  const weeks = plan.mesociclos.flatMap((m) => m.microciclos).sort((a, b) => a.inicio.localeCompare(b.inicio));
  if (weekStart) return weeks.find((w) => w.inicio === weekStart);
  const current = [...weeks].reverse().find((w) => w.inicio <= today);
  return current ?? weeks[0];
}

export function toWeekSessions(week: NonNullable<ReturnType<typeof pickWeek>>) {
  return week.treinos.map((t) => ({
    id: t.id,
    date: t.data,
    sport: SPORT[t.modalidade],
    name: t.nome,
    intensity: INTENSITY[t.intensidade],
    duration_min: Math.round(t.duracao_s / 60),
    planned_tss: t.tss_planejado,
    steps: t.blocos.flatMap((b) =>
      b.passos.map((p) => ({
        type: STEP[p.tipo],
        objective: p.objetivo.valor === undefined ? p.objetivo.tipo : `${p.objetivo.tipo} ${p.objetivo.valor}`,
        target: p.alvo ? `${p.alvo.tipo} ${p.alvo.baixo}-${p.alvo.alto}` : "aberto",
      })),
    ),
  }));
}
