import type { Http } from './http';
import {
  createImportRequestSchema,
  createImportResponseSchema,
  importSchema,
  type CreateImportRequest,
} from './schemas';

export function createImportsApi(http: Http) {
  return {
    create: (req: CreateImportRequest) =>
      http('/imports', { method: 'POST', body: createImportRequestSchema.parse(req), schema: createImportResponseSchema }),
    get: (id: string) => http(`/imports/${encodeURIComponent(id)}`, { schema: importSchema }),
  };
}
export type ImportsApi = ReturnType<typeof createImportsApi>;
