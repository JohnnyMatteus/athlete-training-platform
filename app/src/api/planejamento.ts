import { z } from 'zod';
import type { Http } from './http';

// Espelho de contracts/openapi/planejamento.yaml (commit 5af3b2f da feat/planejamento).
export const modalitySchema = z.enum(['natacao', 'ciclismo', 'corrida', 'forca']);
export type Modality = z.infer<typeof modalitySchema>;

export const intensitySchema = z.enum(['descanso', 'baixa', 'moderada', 'alta']);

const stepSchema = z.object({
  tipo: z.enum(['aquecimento', 'trabalho', 'recuperacao', 'desaquecimento']),
  objetivo: z.object({ tipo: z.enum(['tempo', 'distancia', 'aberto']), valor: z.number().optional() }),
  alvo: z
    .object({
      tipo: z.enum(['potencia_pct_ftp', 'pace_pct_limiar', 'css_pct', 'fc_pct_max', 'rpe']),
      baixo: z.number(),
      alto: z.number(),
    })
    .optional(),
});
export type PlanStep = z.infer<typeof stepSchema>;

export const workoutSchema = z.object({
  id: z.uuid(),
  data: z.iso.date(),
  modalidade: modalitySchema,
  nome: z.string(),
  intensidade: intensitySchema,
  duracao_s: z.number().int(),
  tss_planejado: z.number(),
  blocos: z.array(z.object({ repeticoes: z.number().int().min(1), passos: z.array(stepSchema) })),
});
export type PlanWorkout = z.infer<typeof workoutSchema>;

const violationSchema = z.object({
  regra: z.string(),
  severidade: z.enum(['obrigatoria', 'preferencial']),
  mensagem: z.string(),
  semana: z.number().int().optional(),
  data: z.iso.date().optional(),
});

export const planSummarySchema = z.object({
  id: z.uuid(),
  atleta_id: z.uuid(),
  status: z.enum(['aguardando_aprovacao', 'ativo', 'rejeitado']),
  versao_plano: z.number().int(),
  prova: z.string(),
  data_prova: z.iso.date(),
  inicio: z.iso.date(),
});
export type PlanSummary = z.infer<typeof planSummarySchema>;

export const planSchema = planSummarySchema.extend({
  modo_treinador: z.boolean(),
  treinador_id: z.uuid().nullish(),
  ctl_inicial: z.number(),
  mesociclos: z.array(
    z.object({
      fase: z.enum(['base', 'construcao', 'pico', 'polimento', 'transicao']),
      microciclos: z.array(
        z.object({
          semana: z.number().int(),
          inicio: z.iso.date(),
          descarga: z.boolean(),
          tss_planejado: z.number(),
          ctl_projetado: z.number(),
          treinos: z.array(workoutSchema),
        }),
      ),
    }),
  ),
  avisos: z.array(violationSchema),
});
export type Plan = z.infer<typeof planSchema>;

export const adherenceSchema = z.object({
  plano_id: z.uuid(),
  aderencia_geral: z.number().nullable(),
  semanas: z.array(
    z.object({
      semana: z.number().int(),
      tss_planejado: z.number(),
      tss_realizado: z.number(),
      aderencia: z.number().nullable(),
      sessoes: z.array(
        z.object({
          treino_id: z.uuid(),
          data: z.iso.date(),
          tss_planejado: z.number(),
          tss_realizado: z.number(),
          aderencia: z.number().nullable(),
        }),
      ),
    }),
  ),
});
export type Adherence = z.infer<typeof adherenceSchema>;

export function createPlanejamentoApi(http: Http) {
  return {
    list: () => http('/planos', { schema: z.object({ itens: z.array(planSummarySchema) }) }),
    get: (id: string) => http(`/planos/${encodeURIComponent(id)}`, { schema: planSchema }),
    adherence: (id: string) => http(`/planos/${encodeURIComponent(id)}/aderencia`, { schema: adherenceSchema }),
  };
}
export type PlanejamentoApi = ReturnType<typeof createPlanejamentoApi>;

export function allWorkouts(plan: Plan): PlanWorkout[] {
  return plan.mesociclos.flatMap((m) => m.microciclos.flatMap((c) => c.treinos));
}
