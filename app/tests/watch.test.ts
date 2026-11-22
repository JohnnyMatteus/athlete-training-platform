import { describe, expect, it, vi } from 'vitest';
import type { PlanWorkout } from '@/api/planejamento';
import { plannedWorkoutSchema } from '@/api/plannedWorkout';
import { scheduleFromPlan, toPlannedWorkout } from '@/features/watch/toPlannedWorkout';
import { WatchSyncService } from '@/features/watch/watchSync';
import fixture from '../../plugins/capacitor-workoutkit/fixtures/planned-workout.json';

const ID = '5b1f0c2e-7a64-4f43-9d0e-2f6f6f8a1c11';
const workout: PlanWorkout = {
  id: ID, data: '2026-11-24', modalidade: 'ciclismo', nome: 'Sweet spot', intensidade: 'alta', duracao_s: 4200, tss_planejado: 70,
  blocos: [
    { repeticoes: 1, passos: [{ tipo: 'aquecimento', objetivo: { tipo: 'tempo', valor: 600 } }] },
    { repeticoes: 3, passos: [
      { tipo: 'trabalho', objetivo: { tipo: 'tempo', valor: 600 }, alvo: { tipo: 'potencia_pct_ftp', baixo: 88, alto: 94 } },
      { tipo: 'recuperacao', objetivo: { tipo: 'tempo', valor: 180 }, alvo: { tipo: 'rpe', baixo: 2, alto: 3 } },
    ] },
    { repeticoes: 1, passos: [{ tipo: 'desaquecimento', objetivo: { tipo: 'aberto' } }] },
  ],
};

describe('plano do servidor para o DTO do plugin', () => {
  it('converte % do limiar em watts e separa aquecimento e desaquecimento', () => {
    const dto = toPlannedWorkout(workout, { ftpW: 250 });
    expect(plannedWorkoutSchema.parse(dto)).toBeTruthy();
    expect(dto.aquecimento?.objetivo).toEqual({ tipo: 'tempo', segundos: 600 });
    expect(dto.desaquecimento?.objetivo).toEqual({ tipo: 'aberto' });
    expect(dto.blocos).toHaveLength(1);
    expect(dto.blocos[0]!.passos[0]!.alvo).toEqual({ tipo: 'potencia_w', min: 220, max: 235 });
    expect(dto.blocos[0]!.passos[1]!.alvo).toBeUndefined(); // RPE não tem alvo mensurável
  });

  it('sem limiar o passo fica sem alvo, não com chute', () => {
    expect(toPlannedWorkout(workout, {}).blocos[0]!.passos[0]!.alvo).toBeUndefined();
  });

  it('só plano ativo e só dias de treino vão ao relógio', () => {
    const base = { id: ID, atleta_id: ID, versao_plano: 1, prova: 'x', data_prova: '2027-01-01', inicio: '2026-11-23', modo_treinador: true, ctl_inicial: 0, avisos: [],
      mesociclos: [{ fase: 'base' as const, microciclos: [{ semana: 1, inicio: '2026-11-23', descarga: false, tss_planejado: 70, ctl_projetado: 10, treinos: [workout, { ...workout, id: ID.replace('5', '6'), intensidade: 'descanso' as const, blocos: [] }] }] }] };
    expect(scheduleFromPlan({ ...base, status: 'aguardando_aprovacao' }, {})).toEqual([]);
    const items = scheduleFromPlan({ ...base, status: 'ativo' }, {});
    expect(items).toHaveLength(1);
    expect(items[0]!.data).toBe('2026-11-24T07:00');
  });

  it('o fixture compartilhado com o Swift cumpre o schema', () => {
    expect(plannedWorkoutSchema.parse(fixture).id).toBe(ID);
  });
});

describe('sincronização com o relógio', () => {
  const plugin = () => ({
    requestAuthorization: vi.fn().mockResolvedValue({ status: 'authorized' }),
    syncWindow: vi.fn().mockResolvedValue({ added: [ID], removed: [], deferred: [], failed: [] }),
    openInWorkoutApp: vi.fn(), exportFit: vi.fn().mockResolvedValue({ filename: 'a.fit', base64: 'AA==' }),
  });
  const dto = toPlannedWorkout(workout, {});

  it('recusa fora do iOS sem tocar no plugin', async () => {
    const p = plugin();
    const svc = new WatchSyncService(p as never, () => false, { share: vi.fn() });
    expect(await svc.sync([])).toEqual({ status: 'indisponivel_plataforma' });
    expect(p.requestAuthorization).not.toHaveBeenCalled();
  });

  it('envia a janela com a data local de hoje', async () => {
    const p = plugin();
    const svc = new WatchSyncService(p as never, () => true, { share: vi.fn() }, () => new Date(2026, 10, 22, 9));
    const out = await svc.sync([{ id: ID, data: '2026-11-24T07:00', plano: dto }]);
    expect(out.status).toBe('ok');
    expect(p.syncWindow.mock.calls[0]![0].today).toBe('2026-11-22');
  });

  it('cai para o .fit quando o Workout recusa', async () => {
    const p = plugin();
    p.openInWorkoutApp.mockRejectedValue(new Error('falhou'));
    const share = vi.fn();
    expect(await new WatchSyncService(p as never, () => true, { share }).openOrExport(dto)).toBe('fit_exportado');
    expect(share).toHaveBeenCalledWith('a.fit', 'AA==');
  });
});
