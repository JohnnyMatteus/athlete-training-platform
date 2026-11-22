<script setup lang="ts">
import { computed } from 'vue';
import FolegoChart from '@/components/FolegoChart.vue';
import PageShell from '@/components/PageShell.vue';
import RemoteState from '@/components/RemoteState.vue';
import TsbRule from '@/components/TsbRule.vue';
import WeekStrip from '@/components/WeekStrip.vue';
import { addDays, isoDay, MODALIDADE_LABEL, duration } from '@/domain/format';
import { useAdherence, useCurrentPlan, useLoad } from '@/queries';

const load = useLoad(42);
const cur = useCurrentPlan();
const adherence = useAdherence(() => cur.plan.value?.id);

const today = isoDay(new Date());
const weekStart = computed(() => {
  const dow = (new Date(`${today}T00:00:00Z`).getUTCDay() + 6) % 7;
  return addDays(today, -dow);
});
const last = computed(() => load.days.value[load.days.value.length - 1]);
const todays = computed(() => cur.workouts.value.filter((w) => w.data === today && w.intensidade !== 'descanso'));
</script>

<template>
  <PageShell title="Hoje">
    <section>
      <h2>Seu fôlego</h2>
      <RemoteState :kind="load.kind.value" what="a carga diária" empty-text="Sem carga calculada ainda. Importe treinos para começar.">
        <template v-if="last">
          <TsbRule :tsb="last.tsb" />
          <FolegoChart :days="load.days.value" />
        </template>
        <p v-else>Sem carga calculada ainda. Importe treinos para começar.</p>
      </RemoteState>
    </section>
    <section>
      <h2>A semana</h2>
      <RemoteState :kind="cur.kind.value" what="o plano de treino" empty-text="Você ainda não tem plano. Quando o planejamento gerar o primeiro, ele aparece aqui.">
        <template v-if="cur.plan.value">
          <p v-if="cur.plan.value.status === 'aguardando_aprovacao'" class="note">
            Plano aguardando aprovação do treinador. Ele só chega ao relógio depois disso.
          </p>
          <WeekStrip :week-start="weekStart" :workouts="cur.workouts.value" :done="adherence.done.value" />
          <p v-for="w in todays" :key="w.id" class="today">
            Hoje: <strong>{{ w.nome }}</strong> ({{ MODALIDADE_LABEL[w.modalidade] }}, <span class="num">{{ duration(w.duracao_s) }}</span>)
          </p>
          <p v-if="todays.length === 0">Hoje é dia de descanso no plano.</p>
        </template>
        <p v-else>Você ainda não tem plano. Quando o planejamento gerar o primeiro, ele aparece aqui.</p>
      </RemoteState>
    </section>
  </PageShell>
</template>

<style scoped>
section { margin-bottom: 24px; }
.note { color: var(--atp-warning); }
.today { margin-top: 12px; }
</style>
