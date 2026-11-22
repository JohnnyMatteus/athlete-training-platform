import { z } from 'zod';
import type { KeyValueStore } from '@/platform/kv';
import { consentCategorySchema, type ConsentCategory, type ConsentPayload } from '@/api/schemas';

// Versão do termo apresentada ao atleta. O texto jurídico é do autor (ADR 0008); o do app é provisório.
export const CURRENT_TERM_VERSION = '2026-11';

export const CATEGORY_LABELS: Record<ConsentCategory, { titulo: string; detalhe: string }> = {
  treinos: { titulo: 'Treinos', detalhe: 'Tipo, data, duração e distância das atividades.' },
  frequencia_cardiaca: { titulo: 'Frequência cardíaca', detalhe: 'Amostras de FC durante os treinos.' },
  localizacao: { titulo: 'Localização', detalhe: 'Rotas (GPS) dos treinos ao ar livre.' },
  potencia: { titulo: 'Potência', detalhe: 'Potência e cadência do ciclocomputador.' },
};

const storedSchema = z.object({
  versao_termo: z.string(),
  categorias: z.array(consentCategorySchema),
  aceito_em: z.iso.datetime(),
});
type Stored = z.infer<typeof storedSchema>;

const KEY = 'consent.v1';

export class ConsentRepository {
  constructor(
    private readonly kv: KeyValueStore,
    private readonly now: () => Date = () => new Date(),
  ) {}

  // Termo de versão antiga não vale: o atleta precisa aceitar de novo.
  async current(): Promise<Stored | null> {
    const raw = await this.kv.get(KEY);
    if (!raw) return null;
    try {
      const parsed = storedSchema.parse(JSON.parse(raw));
      return parsed.versao_termo === CURRENT_TERM_VERSION && parsed.categorias.length > 0 ? parsed : null;
    } catch {
      return null;
    }
  }

  async accept(categorias: ConsentCategory[]): Promise<void> {
    if (categorias.length === 0) throw new Error('Escolha ao menos uma categoria.');
    const stored: Stored = {
      versao_termo: CURRENT_TERM_VERSION,
      categorias: [...new Set(categorias)],
      aceito_em: this.now().toISOString(),
    };
    await this.kv.set(KEY, JSON.stringify(stored));
  }

  async revoke(): Promise<void> {
    await this.kv.remove(KEY);
  }

  async has(categoria: ConsentCategory): Promise<boolean> {
    return (await this.current())?.categorias.includes(categoria) ?? false;
  }

  async payload(): Promise<ConsentPayload | null> {
    const c = await this.current();
    return c ? { aceito: true, versao_termo: c.versao_termo, categorias: c.categorias } : null;
  }
}
