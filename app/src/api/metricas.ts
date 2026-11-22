import { z } from 'zod';
import type { Http } from './http';

// Espelho de contracts/openapi/metricas.yaml (commit 5af3b2f da feat/planejamento).
export const loadDaySchema = z.object({
  dia: z.iso.date(),
  tss: z.number(),
  ctl: z.number(),
  atl: z.number(),
  tsb: z.number(),
});
export type LoadDay = z.infer<typeof loadDaySchema>;

export const activityLoadSchema = z.object({
  atividade_id: z.uuid(),
  dia: z.iso.date(),
  tipo: z.string(),
  metodo: z.enum(['tss_potencia', 'rtss', 'stss', 'trimp', 'indisponivel']),
  tss: z.number(),
  np: z.number().nullish(),
  fator_intensidade: z.number().nullish(),
  descartes_fc: z.number().int(),
  descartes_potencia: z.number().int(),
});
export type ActivityLoad = z.infer<typeof activityLoadSchema>;

export const thresholdSchema = z.object({
  tipo: z.enum(['ftp', 'pace_limiar', 'css']),
  valor: z.number().positive(),
  vigente_desde: z.iso.date(),
});
export type Threshold = z.infer<typeof thresholdSchema>;

export const profileSchema = z.object({
  sexo: z.enum(['M', 'F']),
  fc_repouso: z.number().int().min(30).max(120),
  fc_max: z.number().int().min(100).max(230),
});
export type Profile = z.infer<typeof profileSchema>;

export type DateRange = { de?: string; ate?: string };

export function createMetricasApi(http: Http) {
  return {
    load: (r: DateRange) =>
      http('/metricas/carga', { schema: z.object({ atleta_id: z.uuid(), dias: z.array(loadDaySchema) }), query: r }),
    activityLoads: (r: DateRange) =>
      http('/metricas/atividades', { schema: z.object({ itens: z.array(activityLoadSchema) }), query: r }),
    thresholds: () => http('/metricas/limiares', { schema: z.object({ itens: z.array(thresholdSchema) }) }),
    putThreshold: (t: Threshold) => http('/metricas/limiares', { method: 'PUT', body: thresholdSchema.parse(t), schema: thresholdSchema }),
    profile: () => http('/metricas/perfil', { schema: profileSchema }),
    putProfile: (p: Profile) => http('/metricas/perfil', { method: 'PUT', body: profileSchema.parse(p), schema: profileSchema }),
  };
}
export type MetricasApi = ReturnType<typeof createMetricasApi>;
