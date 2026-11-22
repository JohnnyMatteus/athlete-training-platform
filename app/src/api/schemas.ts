import { z } from 'zod';

// Espelho de contracts/openapi/imports.yaml.
export const fileTypeSchema = z.enum(['zip', 'fit', 'tcx', 'gpx']);
export type FileType = z.infer<typeof fileTypeSchema>;

export const consentCategorySchema = z.enum(['treinos', 'frequencia_cardiaca', 'localizacao', 'potencia']);
export type ConsentCategory = z.infer<typeof consentCategorySchema>;

export const consentSchema = z.object({
  aceito: z.literal(true),
  versao_termo: z.string().min(1).max(32),
  categorias: z.array(consentCategorySchema).min(1),
});
export type ConsentPayload = z.infer<typeof consentSchema>;

export const createImportRequestSchema = z.object({
  nome_arquivo: z.string().min(1).max(255),
  tamanho_bytes: z.number().int().min(1),
  tipo_arquivo: fileTypeSchema,
  sha256: z.string().regex(/^[0-9a-f]{64}$/),
  consentimento: consentSchema,
});
export type CreateImportRequest = z.infer<typeof createImportRequestSchema>;

export const importStatusSchema = z.enum(['aguardando_upload', 'recebido', 'processando', 'concluido', 'falhou']);
export type ImportStatus = z.infer<typeof importStatusSchema>;

export const createImportResponseSchema = z.object({
  import_id: z.uuid(),
  status: importStatusSchema,
  upload_url: z.url(),
  upload_metadata: z.object({ import_id: z.uuid() }),
});

export const importSchema = z.object({
  import_id: z.uuid(),
  nome_arquivo: z.string(),
  tamanho_bytes: z.number().int(),
  tipo_arquivo: fileTypeSchema,
  sha256: z.string(),
  status: importStatusSchema,
  versao_formato: z.string().nullish(),
  importadas: z.number().int().min(0),
  duplicadas: z.number().int().min(0),
  erros: z.number().int().min(0),
  motivo_falha: z.string().nullish(),
  criado_em: z.iso.datetime(),
  atualizado_em: z.iso.datetime(),
});
export type ImportBatch = z.infer<typeof importSchema>;

export const problemSchema = z.object({
  type: z.string(),
  title: z.string(),
  status: z.number().int(),
  detail: z.string().optional(),
  errors: z.array(z.object({ campo: z.string(), mensagem: z.string() })).optional(),
});
export type Problem = z.infer<typeof problemSchema>;

export const TERMINAL_STATUSES: readonly ImportStatus[] = ['concluido', 'falhou'];
