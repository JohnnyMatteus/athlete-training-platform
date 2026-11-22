import * as tus from 'tus-js-client';
import { assertTrustedUploadUrl, isTrustedUploadUrl } from './uploadOrigin';

export interface UploadRequest {
  file: Blob & { name?: string };
  endpoint: string;
  metadata: Record<string, string>;
  token: () => Promise<string | null>;
  onProgress: (sent: number, total: number) => void;
}

export interface UploadHandle {
  /** true quando o envio retomou um upload anterior do mesmo arquivo. */
  resumed: boolean;
  done: Promise<void>;
  abort(): Promise<void>;
}

// Porta do upload resumível (protocolo tus; servidor tusd, ADR 0010).
export interface ResumableUploader {
  start(req: UploadRequest): Promise<UploadHandle>;
}

export class TusUploader implements ResumableUploader {
  constructor(private readonly allowedOrigins: readonly string[]) {}

  async start(req: UploadRequest): Promise<UploadHandle> {
    assertTrustedUploadUrl(req.endpoint, this.allowedOrigins);
    let resolve!: () => void;
    let reject!: (e: Error) => void;
    const done = new Promise<void>((res, rej) => {
      resolve = res;
      reject = rej;
    });

    const upload = new tus.Upload(req.file, {
      endpoint: req.endpoint,
      metadata: { ...req.metadata, filename: req.file.name ?? 'arquivo' },
      chunkSize: 8 * 1024 * 1024,
      retryDelays: [0, 1000, 3000, 5000, 10000],
      storeFingerprintForResuming: true,
      removeFingerprintOnSuccess: true,
      // O token é lido a cada requisição: um JWT renovado durante um upload longo vale na hora.
      // Inclui a URL retomada do armazenamento e o Location devolvido pelo servidor: cada requisição é conferida.
      onBeforeRequest: async (r) => {
        if (!isTrustedUploadUrl(r.getURL(), this.allowedOrigins)) throw new Error(`origem de upload não confiável: ${r.getURL()}`);
        const t = await req.token();
        if (t) r.setHeader('Authorization', `Bearer ${t}`);
      },
      onProgress: req.onProgress,
      onSuccess: () => resolve(),
      onError: (e) => reject(e),
    });

    const previous = await upload.findPreviousUploads();
    const last = previous[0];
    if (last) upload.resumeFromPreviousUpload(last);
    upload.start();
    return { resumed: Boolean(last), done, abort: () => upload.abort(true) };
  }
}
