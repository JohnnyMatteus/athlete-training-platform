import { z } from 'zod';
import type { Http } from './http';

// Espelho de contracts/openapi/atividades.yaml. A face /internal não é do app.
export const activityTypeSchema = z.enum([
  'natacao_piscina',
  'natacao_aguas_abertas',
  'ciclismo_indoor',
  'ciclismo_outdoor',
  'corrida',
  'forca',
  'outro',
]);

export const correctionSchema = z
  .object({ tipo: activityTypeSchema.optional(), inicio_utc: z.iso.datetime().optional() })
  .refine((v) => v.tipo !== undefined || v.inicio_utc !== undefined, 'Informe tipo ou inicio_utc');
export type ActivityCorrection = z.infer<typeof correctionSchema>;

export const correctedActivitySchema = z.object({
  atividade_id: z.uuid(),
  tipo: z.string(),
  inicio_utc: z.iso.datetime(),
  duracao_s: z.number().int(),
  distancia_m: z.number().nullable(),
});

export function createAtividadesApi(http: Http) {
  return {
    correct: (id: string, c: ActivityCorrection) =>
      http(`/atividades/${encodeURIComponent(id)}`, { method: 'PATCH', body: correctionSchema.parse(c), schema: correctedActivitySchema }),
  };
}
export type AtividadesApi = ReturnType<typeof createAtividadesApi>;
