import type { ImportsApi } from '@/api/imports';
import { fileTypeSchema, type FileType, type ImportBatch } from '@/api/schemas';
import type { ConsentRepository } from '@/features/session/consent';
import type { KeyValueStore } from '@/platform/kv';
import { sha256Hex } from './sha256';
import { assertTrustedUploadUrl } from './uploadOrigin';
import type { ResumableUploader, UploadHandle } from './tusUploader';

export class ConsentRequiredError extends Error {
  constructor() {
    super('Aceite o termo de consentimento antes de importar.');
  }
}
export class UnsupportedFileError extends Error {
  constructor(name: string) {
    super(`Formato não suportado: ${name}. Use .zip (export do app Saúde), .fit, .tcx ou .gpx.`);
  }
}

export function fileTypeOf(name: string): FileType {
  const ext = name.split('.').pop()?.toLowerCase();
  const parsed = fileTypeSchema.safeParse(ext);
  if (!parsed.success) throw new UnsupportedFileError(name);
  return parsed.data;
}

export type ImportPhase = 'hash' | 'criando' | 'enviando' | 'enviado';

export interface ImportProgress {
  phase: ImportPhase;
  fraction: number;
  resumed?: boolean;
  importId?: string;
}

const HISTORY_KEY = 'imports.history.v1';
const HISTORY_MAX = 20;

// O contrato não tem listagem de lotes: o app guarda os ids dos próprios envios para acompanhá-los.
export class ImportHistory {
  constructor(private readonly kv: KeyValueStore) {}
  async ids(): Promise<string[]> {
    try {
      const v: unknown = JSON.parse((await this.kv.get(HISTORY_KEY)) ?? '[]');
      return Array.isArray(v) ? v.filter((x): x is string => typeof x === 'string') : [];
    } catch {
      return [];
    }
  }
  async add(id: string): Promise<void> {
    const next = [id, ...(await this.ids()).filter((x) => x !== id)].slice(0, HISTORY_MAX);
    await this.kv.set(HISTORY_KEY, JSON.stringify(next));
  }
}

interface Pending {
  import_id: string;
  upload_url: string;
  upload_metadata: Record<string, string>;
}

// Um envio interrompido volta ao mesmo lote (e ao mesmo upload tus) em vez de criar outro.
export class PendingUploads {
  constructor(private readonly kv: KeyValueStore) {}
  private key(f: File) {
    return `imports.pending.${f.name}.${f.size}.${f.lastModified}`;
  }
  async get(f: File): Promise<Pending | null> {
    try {
      return JSON.parse((await this.kv.get(this.key(f))) ?? 'null') as Pending | null;
    } catch {
      return null;
    }
  }
  set(f: File, p: Pending): Promise<void> {
    return this.kv.set(this.key(f), JSON.stringify(p));
  }
  clear(f: File): Promise<void> {
    return this.kv.remove(this.key(f));
  }
}

export class ImportService {
  constructor(
    private readonly api: ImportsApi,
    private readonly uploader: ResumableUploader,
    private readonly consent: ConsentRepository,
    private readonly history: ImportHistory,
    private readonly pending: PendingUploads,
    private readonly token: () => Promise<string | null>,
    private readonly allowedUploadOrigins: readonly string[],
  ) {}

  async start(file: File, onProgress: (p: ImportProgress) => void): Promise<{ importId: string; handle: UploadHandle }> {
    const tipo = fileTypeOf(file.name);
    const consentimento = await this.consent.payload();
    if (!consentimento) throw new ConsentRequiredError();

    const resumable = await this.findResumable(file);
    let target: Pending;
    if (resumable) {
      target = resumable;
    } else {
      onProgress({ phase: 'hash', fraction: 0 });
      const sha256 = await sha256Hex(file, (f) => onProgress({ phase: 'hash', fraction: f }));
      onProgress({ phase: 'criando', fraction: 0 });
      target = await this.api.create({
        nome_arquivo: file.name,
        tamanho_bytes: file.size,
        tipo_arquivo: tipo,
        sha256,
        consentimento,
      });
      await this.pending.set(file, target);
      await this.history.add(target.import_id);
    }
    const created = target;
    // Antes de qualquer envio: a URL do upload vem do servidor, não é confiável por si só.
    assertTrustedUploadUrl(created.upload_url, this.allowedUploadOrigins);

    const handle = await this.uploader.start({
      file,
      endpoint: created.upload_url,
      metadata: created.upload_metadata,
      token: this.token,
      onProgress: (sent, total) =>
        onProgress({ phase: 'enviando', fraction: total ? sent / total : 0, importId: created.import_id }),
    });
    handle.done.then(
      async () => {
        await this.pending.clear(file);
        onProgress({ phase: 'enviado', fraction: 1, importId: created.import_id, resumed: handle.resumed });
      },
      () => undefined,
    );
    return { importId: created.import_id, handle };
  }

  private async findResumable(file: File): Promise<Pending | null> {
    const p = await this.pending.get(file);
    if (!p) return null;
    try {
      if ((await this.api.get(p.import_id)).status === 'aguardando_upload') return p;
    } catch {
      // lote sumiu ou não é mais nosso: começa do zero
    }
    await this.pending.clear(file);
    return null;
  }

  get(id: string): Promise<ImportBatch> {
    return this.api.get(id);
  }
}
