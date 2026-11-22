import { readFileSync } from 'node:fs';
import { join } from 'node:path';
import { parse } from 'yaml';
import { describe, expect, it } from 'vitest';
import type { z } from 'zod';
import { correctedActivitySchema, activityTypeSchema } from '@/api/atividades';
import { activityLoadSchema, loadDaySchema, profileSchema, thresholdSchema } from '@/api/metricas';
import { adherenceSchema, planSchema, planSummarySchema, workoutSchema } from '@/api/planejamento';
import { createImportRequestSchema, importSchema, createImportResponseSchema } from '@/api/schemas';

type Spec = { components: { schemas: Record<string, Record<string, unknown>> }; paths: Record<string, Record<string, Record<string, unknown>>> };
const load = (f: string) =>
  parse(readFileSync(join(process.cwd(), '..', 'contracts', 'openapi', `${f}.yaml`), 'utf8')) as Spec;

const planejamento = load('planejamento');
const metricas = load('metricas');
const atividades = load('atividades');
const imports = load('imports');

type OaObject = { properties?: Record<string, { enum?: unknown[] }>; required?: string[] };
const ok = (spec: Spec, path: string, method: string): OaObject => {
  const r = spec.paths[path]?.[method] as { responses: Record<string, { content?: Record<string, { schema: OaObject }> }> };
  return r.responses['200']!.content!['application/json']!.schema;
};

// Cada propriedade do contrato precisa existir no zod, e as obrigatórias não podem ser opcionais lá.
function matches(zodObj: z.ZodObject, oa: OaObject) {
  const shape = zodObj.shape as Record<string, z.ZodType>;
  expect(Object.keys(shape).sort()).toEqual(Object.keys(oa.properties ?? {}).sort());
  for (const key of oa.required ?? []) expect(shape[key]!.safeParse(undefined).success, `${key} deve ser obrigatório`).toBe(false);
}

describe('zod contra contracts/openapi', () => {
  it('métricas: carga, atividades, limiares, perfil', () => {
    const load = ok(metricas, '/metricas/carga', 'get').properties!.dias as unknown as { items: OaObject };
    matches(loadDaySchema, load.items);
    const acts = ok(metricas, '/metricas/atividades', 'get').properties!.itens as unknown as { items: OaObject };
    matches(activityLoadSchema, acts.items);
    expect(activityLoadSchema.shape.metodo.options).toEqual(acts.items.properties!.metodo!.enum);
    matches(thresholdSchema, metricas.components.schemas.Threshold as OaObject);
    expect(thresholdSchema.shape.tipo.options).toEqual((metricas.components.schemas.Threshold as OaObject).properties!.tipo!.enum);
    matches(profileSchema, metricas.components.schemas.Profile as OaObject);
  });

  it('planejamento: plano, treino, aderência', () => {
    const s = planejamento.components.schemas as Record<string, OaObject>;
    matches(planSummarySchema, s.PlanSummary!);
    matches(workoutSchema, s.Workout!);
    expect(workoutSchema.shape.modalidade.options).toEqual((planejamento.components.schemas.Modality as { enum: string[] }).enum);
    const plan = (s.Plan as unknown as { allOf: OaObject[] }).allOf[1]!;
    expect(Object.keys(planSchema.shape)).toEqual(expect.arrayContaining(Object.keys(plan.properties!)));
    matches(adherenceSchema, s.Adherence!);
  });

  it('atividades: correção', () => {
    const patch = atividades.paths['/atividades/{atividade_id}']!.patch as {
      requestBody: { content: Record<string, { schema: OaObject }> };
      responses: Record<string, { content: Record<string, { schema: OaObject }> }>;
    };
    const body = patch.requestBody.content['application/json']!.schema;
    expect(activityTypeSchema.options).toEqual(body.properties!.tipo!.enum);
    matches(correctedActivitySchema, patch.responses['200']!.content['application/json']!.schema);
  });

  it('importação: requisição, resposta e lote', () => {
    const s = imports.components.schemas as Record<string, OaObject>;
    matches(createImportRequestSchema, s.CreateImportRequest!);
    matches(createImportResponseSchema, s.CreateImportResponse!);
    matches(importSchema, s.Import!);
  });
});
