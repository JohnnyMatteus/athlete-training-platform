<script setup lang="ts">
import { computed, ref } from 'vue';
import { IonButton } from '@ionic/vue';
import PageShell from '@/components/PageShell.vue';
import RemoteState from '@/components/RemoteState.vue';
import { addDays, isoDay, MODALIDADE_LABEL, num0 } from '@/domain/format';
import { useAdherence, useCurrentPlan } from '@/queries';

const cur = useCurrentPlan();
const adherence = useAdherence(() => cur.plan.value?.id);
const anchor = ref(isoDay(new Date()).slice(0, 7));

const grid = computed(() => {
  const first = `${anchor.value}-01`;
  const lead = (new Date(`${first}T00:00:00Z`).getUTCDay() + 6) % 7;
  const start = addDays(first, -lead);
  return Array.from({ length: 42 }, (_, i) => {
    const date = addDays(start, i);
    const w = cur.workouts.value.find((x) => x.data === date && x.intensidade !== 'descanso');
    return { date, inMonth: date.startsWith(anchor.value), w, done: Boolean(w && adherence.done.value.has(w.id)) };
  });
});

function shift(n: number) {
  const [y, m] = anchor.value.split('-').map(Number) as [number, number];
  const d = new Date(Date.UTC(y, m - 1 + n, 1));
  anchor.value = d.toISOString().slice(0, 7);
}
const title = computed(() => {
  const [y, m] = anchor.value.split('-');
  return `${new Intl.DateTimeFormat('pt-BR', { month: 'long' }).format(new Date(Date.UTC(Number(y), Number(m) - 1, 1)))} de ${y}`;
});
</script>

<template>
  <PageShell title="Calendário" back="/mais">
    <div class="nav">
      <ion-button fill="clear" aria-label="Mês anterior" @click="shift(-1)">‹</ion-button>
      <strong>{{ title }}</strong>
      <ion-button fill="clear" aria-label="Próximo mês" @click="shift(1)">›</ion-button>
    </div>
    <RemoteState :kind="cur.kind.value" what="o plano de treino" empty-text="Sem plano: o calendário fica vazio.">
      <div class="grid">
        <span v-for="n in ['seg','ter','qua','qui','sex','sáb','dom']" :key="n" class="hd">{{ n }}</span>
        <div v-for="c in grid" :key="c.date" class="cell" :class="{ off: !c.inMonth }">
          <span class="num d">{{ c.date.slice(8) }}</span>
          <template v-if="c.w">
            <span class="m">{{ MODALIDADE_LABEL[c.w.modalidade] }}</span>
            <span class="num t">{{ num0(c.w.tss_planejado) }}</span>
            <span class="mark" :class="{ done: c.done }" />
          </template>
        </div>
      </div>
    </RemoteState>
  </PageShell>
</template>

<style scoped>
.nav { display: flex; align-items: center; justify-content: space-between; }
.grid { display: grid; grid-template-columns: repeat(7, minmax(0, 1fr)); gap: 2px; }
.hd { font-size: 11px; text-transform: uppercase; text-align: center; }
.cell { background: var(--atp-surface); min-height: 64px; padding: 2px 4px; font-size: 11px; display: flex; flex-direction: column; }
.cell.off { opacity: 0.45; }
.d { font-size: 12px; }
.mark { margin-top: auto; border-top: 2px dashed var(--atp-gap-edge); }
.mark.done { border-top: 2px solid var(--atp-primary); }
</style>
