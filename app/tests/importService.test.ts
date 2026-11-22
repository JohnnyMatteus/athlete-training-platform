import { describe, expect, it, vi } from 'vitest';
import { ConsentRepository } from '@/features/session/consent';
import { ConsentRequiredError, ImportHistory, ImportService, PendingUploads, UnsupportedFileError, fileTypeOf } from '@/features/imports/importService';
import { sha256Hex } from '@/features/imports/sha256';
import type { ResumableUploader } from '@/features/imports/tusUploader';
import { MemoryStore } from '@/platform/kv';

const ID = '3c1d2e4f-0b1a-4c2d-8e3f-9a8b7c6d5e40';
const created = { import_id: ID, status: 'aguardando_upload', upload_url: 'http://tus.test/files/', upload_metadata: { import_id: ID } };

function setup(status = 'aguardando_upload') {
  const kv = new MemoryStore();
  const consent = new ConsentRepository(kv);
  const api = {
    create: vi.fn().mockResolvedValue(created),
    get: vi.fn().mockResolvedValue({ status }),
  };
  const start = vi.fn().mockResolvedValue({ resumed: false, done: new Promise<void>(() => undefined), abort: async () => undefined });
  const uploader: ResumableUploader = { start };
  const svc = new ImportService(api as never, uploader, consent, new ImportHistory(kv), new PendingUploads(kv), async () => 'tok');
  return { kv, consent, api, start, svc };
}
const file = () => new File(['abc'], 'export.zip', { lastModified: 1 });

describe('sha256 incremental', () => {
  it('bate com o vetor conhecido de "abc"', async () => {
    expect(await sha256Hex(file())).toBe('ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad');
  });
});

describe('importação', () => {
  it('reconhece só os formatos do contrato', () => {
    expect(fileTypeOf('A.GPX')).toBe('gpx');
    expect(() => fileTypeOf('x.pdf')).toThrow(UnsupportedFileError);
  });

  it('exige consentimento antes de qualquer chamada', async () => {
    const { svc, api } = setup();
    await expect(svc.start(file(), () => undefined)).rejects.toBeInstanceOf(ConsentRequiredError);
    expect(api.create).not.toHaveBeenCalled();
  });

  it('cria o lote com o sha256 e o consentimento e envia pelo tus', async () => {
    const { svc, api, start, consent } = setup();
    await consent.accept(['treinos', 'frequencia_cardiaca']);
    await svc.start(file(), () => undefined);
    expect(api.create.mock.calls[0]![0]).toMatchObject({
      nome_arquivo: 'export.zip', tipo_arquivo: 'zip', tamanho_bytes: 3,
      sha256: 'ba7816bf8f01cfea414140de5dae2223b00361a396177a9cb410ff61f20015ad',
      consentimento: { aceito: true, categorias: ['treinos', 'frequencia_cardiaca'] },
    });
    expect(start.mock.calls[0]![0]).toMatchObject({ endpoint: created.upload_url, metadata: { import_id: ID } });
  });

  it('retoma o lote pendente em vez de criar outro', async () => {
    const { svc, api, start, consent } = setup();
    await consent.accept(['treinos']);
    const f = file();
    await svc.start(f, () => undefined);
    await svc.start(f, () => undefined);
    expect(api.create).toHaveBeenCalledTimes(1);
    expect(start).toHaveBeenCalledTimes(2);
  });
});
