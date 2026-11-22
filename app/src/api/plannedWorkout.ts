import { z } from 'zod';
import type { PlannedWorkoutDTO } from '@atp/capacitor-workoutkit';

// Contrato estável app <-> plugin (ADR 0006). Unidades em SI: segundos, metros, bpm, watts, m/s.
const goalSchema = z.discriminatedUnion('tipo', [
  z.object({ tipo: z.literal('tempo'), segundos: z.number().int().positive() }),
  z.object({ tipo: z.literal('distancia'), metros: z.number().positive() }),
  z.object({ tipo: z.literal('aberto') }),
]);

const targetSchema = z.discriminatedUnion('tipo', [
  z.object({ tipo: z.literal('fc_bpm'), min: z.number().positive(), max: z.number().positive() }),
  z.object({ tipo: z.literal('potencia_w'), min: z.number().positive(), max: z.number().positive() }),
  z.object({ tipo: z.literal('velocidade_ms'), min: z.number().positive(), max: z.number().positive() }),
  z.object({ tipo: z.literal('zona_fc'), zona: z.number().int().min(1).max(5) }),
  z.object({ tipo: z.literal('zona_potencia'), zona: z.number().int().min(1).max(7) }),
]);

const stepSchema = z.object({
  tipo: z.enum(['aquecimento', 'trabalho', 'recuperacao', 'desaquecimento']),
  nome: z.string().optional(),
  objetivo: goalSchema,
  alvo: targetSchema.optional(),
});

export const plannedWorkoutSchema = z.object({
  id: z.uuid(),
  nome: z.string().min(1),
  modalidade: z.enum(['corrida', 'ciclismo', 'natacao', 'forca', 'outro']),
  ambiente: z.enum(['indoor', 'outdoor', 'piscina', 'aguas_abertas']).optional(),
  aquecimento: stepSchema.optional(),
  blocos: z.array(z.object({ repeticoes: z.number().int().min(1), passos: z.array(stepSchema).min(1) })),
  desaquecimento: stepSchema.optional(),
}) satisfies z.ZodType<PlannedWorkoutDTO>;

export interface ScheduledItem {
  id: string;
  /** Data e hora locais, `YYYY-MM-DDTHH:mm`. */
  data: string;
  plano: PlannedWorkoutDTO;
}
