import { computed, toValue, type MaybeRefOrGetter } from 'vue';
import { allWorkouts } from '@/api/planejamento';
import { addDays, isoDay } from '@/domain/format';
import { limitsFrom } from '@/features/watch/toPlannedWorkout';
import { remoteKind, useRemote, useServices } from '@/composables';

export function useLoad(days: MaybeRefOrGetter<number> = 90) {
  const s = useServices();
  const to = isoDay(new Date());
  const q = useRemote(['load', computed(() => toValue(days))], () => s.metricas.load({ de: addDays(to, -toValue(days)), ate: to }));
  return { q, kind: remoteKind(q), days: computed(() => q.data.value?.dias ?? []) };
}

// O plano mais recente: ativo primeiro; sem ativo, o último criado.
export function useCurrentPlan() {
  const s = useServices();
  const q = useRemote(['plan-current'], async () => {
    const { itens } = await s.planejamento.list();
    const pick = itens.find((p) => p.status === 'ativo') ?? itens[itens.length - 1];
    return pick ? s.planejamento.get(pick.id) : null;
  });
  const kind = remoteKind(q);
  const workouts = computed(() => (q.data.value ? allWorkouts(q.data.value) : []));
  return { q, kind, plan: computed(() => q.data.value ?? null), workouts };
}

export function useLimits() {
  const s = useServices();
  const t = useRemote(['thresholds'], () => s.metricas.thresholds());
  const p = useRemote(['profile'], () => s.metricas.profile());
  return { t, p, limits: computed(() => limitsFrom(t.data.value?.itens ?? [], p.data.value ?? null)) };
}

export function useAdherence(planId: () => string | undefined) {
  const s = useServices();
  const q = useRemote(
    ['adherence', computed(planId)],
    () => s.planejamento.adherence(planId() as string),
    computed(() => Boolean(planId())),
  );
  const done = computed(
    () => new Set((q.data.value?.semanas ?? []).flatMap((w) => w.sessoes.filter((x) => x.tss_realizado > 0).map((x) => x.treino_id))),
  );
  return { q, done };
}
