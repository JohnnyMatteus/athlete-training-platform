import { mount } from '@vue/test-utils';
import { describe, expect, it } from 'vitest';
import RemoteState from '@/components/RemoteState.vue';
import FolegoChart from '@/components/FolegoChart.vue';
import { ConsentRepository, CURRENT_TERM_VERSION } from '@/features/session/consent';
import { MemoryStore } from '@/platform/kv';
import { MemoryTokenStore } from '@/platform/tokenStore';

describe('estados remotos', () => {
  it('endpoint ausente mostra indisponível e não renderiza o conteúdo', () => {
    const w = mount(RemoteState, { props: { kind: 'unavailable', what: 'as zonas' }, slots: { default: '<p id="dado">dado</p>' }, global: { stubs: { IonSpinner: true } } });
    expect(w.find('[data-testid=indisponivel]').text()).toContain('Nada é estimado');
    expect(w.find('#dado').exists()).toBe(false);
  });
});

describe('gráfico do fôlego', () => {
  it('descreve o último dia para leitores de tela e desenha as duas linhas', () => {
    const w = mount(FolegoChart, { props: { days: [
      { dia: '2026-11-01', tss: 50, ctl: 40, atl: 35, tsb: 5 },
      { dia: '2026-11-02', tss: 90, ctl: 41, atl: 45, tsb: -4 },
    ] } });
    expect(w.find('svg').attributes('aria-label')).toBe('Forma 41,0, fadiga 45,0, fôlego -4,0, faixa neutro');
    expect(w.findAll('path.gap-fresh, path.gap-tired').length).toBe(2);
  });
});

describe('consentimento e token', () => {
  it('termo de versão antiga não vale', async () => {
    const kv = new MemoryStore();
    await kv.set('consent.v1', JSON.stringify({ versao_termo: '2020-01', categorias: ['treinos'], aceito_em: '2026-01-01T00:00:00.000Z' }));
    expect(await new ConsentRepository(kv).current()).toBeNull();
    await new ConsentRepository(kv).accept(['treinos']);
    expect((await new ConsentRepository(kv).current())?.versao_termo).toBe(CURRENT_TERM_VERSION);
  });

  it('token na web fica só em memória, nunca em localStorage', async () => {
    const s = new MemoryTokenStore();
    await s.set('a.b.c');
    expect(await s.get()).toBe('a.b.c');
    expect(JSON.stringify({ ...localStorage })).not.toContain('a.b.c');
  });
});
