<script setup lang="ts">
import { computed } from 'vue';
import { readForm } from '@/domain/form';
import { num1 } from '@/domain/format';

const props = defineProps<{ tsb: number }>();
const reading = computed(() => readForm(props.tsb));
// Faixas desenhadas de -40 a +35; o marcador fica preso nas bordas.
const pos = computed(() => Math.min(100, Math.max(0, ((props.tsb + 40) / 75) * 100)));
</script>

<template>
  <div class="rule">
    <p class="head">
      <span class="num value">TSB {{ num1(tsb) }}</span> · <strong>{{ reading.label }}</strong>
    </p>
    <div class="bar" aria-hidden="true">
      <span v-for="n in 5" :key="n" class="seg" :class="{ on: ['sobrecarga', 'produtivo', 'neutro', 'fresco', 'perdendo_forma'][n - 1] === reading.band }" />
      <span class="marker" :style="{ left: pos + '%' }" />
    </div>
    <p class="action">{{ reading.action }}</p>
  </div>
</template>

<style scoped>
.head { margin: 0 0 6px; }
.value { font-size: 22px; }
.bar { position: relative; display: flex; height: 10px; gap: 2px; }
.seg { flex: 1; background: var(--atp-surface-variant); }
.seg.on { background: var(--atp-primary); }
.marker { position: absolute; top: -4px; width: 2px; height: 18px; background: var(--atp-on-surface); }
.action { margin: 8px 0 0; }
</style>
