<script setup lang="ts">
import { computed } from 'vue';
import type { PlanWorkout } from '@/api/planejamento';
import { addDays } from '@/domain/format';
import { MODALIDADE_LABEL, dayMonth, num0 } from '@/domain/format';

const props = defineProps<{ weekStart: string; workouts: PlanWorkout[]; done: Set<string> }>();
const NAMES = ['seg', 'ter', 'qua', 'qui', 'sex', 'sáb', 'dom'];

const days = computed(() =>
  NAMES.map((name, i) => {
    const date = addDays(props.weekStart, i);
    const w = props.workouts.find((x) => x.data === date && x.intensidade !== 'descanso');
    return { name, date, w, done: Boolean(w && props.done.has(w.id)) };
  }),
);
</script>

<template>
  <ol class="week">
    <li v-for="d in days" :key="d.date" :class="{ rest: !d.w }">
      <span class="dn">{{ d.name }} <span class="num">{{ dayMonth(d.date) }}</span></span>
      <template v-if="d.w">
        <strong>{{ MODALIDADE_LABEL[d.w.modalidade] }}</strong>
        <span class="num">{{ num0(d.w.tss_planejado) }} TSS</span>
      </template>
      <span v-else>descanso</span>
      <span class="mark" :class="{ done: d.done }">{{ d.w ? (d.done ? 'cumprida' : 'a cumprir') : '' }}</span>
    </li>
  </ol>
</template>

<style scoped>
.week { list-style: none; margin: 0; padding: 0; display: grid; grid-template-columns: repeat(7, minmax(0, 1fr)); gap: 2px; }
li { display: flex; flex-direction: column; gap: 2px; padding: 6px 4px; background: var(--atp-surface); font-size: 12px; min-height: 84px; }
li.rest { opacity: 0.7; }
.dn { text-transform: uppercase; font-size: 11px; }
.mark { margin-top: auto; border-top: 2px dashed var(--atp-gap-edge); padding-top: 2px; font-size: 11px; }
.mark.done { border-top: 2px solid var(--atp-primary); }
@media (max-width: 420px) { .week { grid-template-columns: 1fr; } li { flex-direction: row; min-height: 0; gap: 8px; align-items: center; } .mark { margin-top: 0; border-top: 0; margin-left: auto; } }
</style>
