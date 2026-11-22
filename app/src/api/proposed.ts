import { z } from 'zod';
import type { Http } from './http';

/*
 * Só o que NÃO tem contrato publicado em contracts/openapi: a entrega de treinos lidos
 * do HealthKit/Health Connect. Formato proposto; o contrato real, quando sair, manda. Enquanto o
 * servidor responder 404/405/501 a tela mostra "indisponível" e nada é inventado.
 */
export const healthBatchSchema = z.object({
  fonte: z.enum(['healthkit', 'health_connect']),
  treinos: z.array(
    z.object({
      id_externo: z.string(),
      tipo_origem: z.string(),
      inicio_utc: z.iso.datetime(),
      fim_utc: z.iso.datetime(),
      duracao_s: z.number().int().min(0),
      distancia_m: z.number().optional(),
      energia_kcal: z.number().optional(),
      origem: z.string().optional(),
    }),
  ),
});
export type HealthBatch = z.infer<typeof healthBatchSchema>;

export function createHealthIngestApi(http: Http) {
  return {
    sendBatch: (batch: HealthBatch) =>
      http('/health/workouts', { method: 'POST', body: healthBatchSchema.parse(batch), schema: z.object({}).loose(), proposed: true }),
  };
}
export type HealthIngestApi = ReturnType<typeof createHealthIngestApi>;
