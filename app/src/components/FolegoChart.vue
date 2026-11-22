<script setup lang="ts">
import { computed } from 'vue';
import type { LoadDay } from '@/api/metricas';
import { gapShapes } from '@/domain/gap';
import { dayMonth, num0 } from '@/domain/format';
import { summarize } from '@/domain/form';

const props = defineProps<{ days: LoadDay[] }>();

const W = 640;
const H = 220;
const PAD = { l: 36, r: 8, t: 8, b: 40 };

const model = computed(() => {
  const d = props.days;
  const n = Math.max(d.length - 1, 1);
  const vals = d.flatMap((p) => [p.ctl, p.atl]);
  const lo = Math.min(0, ...vals);
  const hi = Math.max(10, ...vals);
  const maxTss = Math.max(1, ...d.map((p) => p.tss));
  const x = (i: number) => PAD.l + (i / n) * (W - PAD.l - PAD.r);
  const y = (v: number) => PAD.t + (1 - (v - lo) / (hi - lo)) * (H - PAD.t - PAD.b);
  const line = (k: 'ctl' | 'atl') => d.map((p, i) => `${i ? 'L' : 'M'}${x(i).toFixed(1)},${y(p[k]).toFixed(1)}`).join(' ');
  const shapes = gapShapes(d.map((p, i) => ({ x: x(i), ctl: p.ctl, atl: p.atl }))).map((s) => ({
    fresh: s.fresh,
    d: s.vertices.map((v, i) => `${i ? 'L' : 'M'}${v.x.toFixed(1)},${y(v.y).toFixed(1)}`).join(' ') + 'Z',
  }));
  const bw = Math.max(1, ((W - PAD.l - PAD.r) / d.length) * 0.6);
  const bars = d.map((p, i) => {
    const h = (p.tss / maxTss) * 28;
    return { x: x(i) - bw / 2, y: H - PAD.b + 6 + (28 - h), w: bw, h };
  });
  return { ctl: line('ctl'), atl: line('atl'), shapes, bars, ticks: [lo, (lo + hi) / 2, hi].map((v) => ({ v, y: y(v) })) };
});

const last = computed(() => props.days[props.days.length - 1]);
const label = computed(() => (last.value ? summarize(last.value) : 'Sem dados'));
</script>

<template>
  <figure class="folego">
    <svg :viewBox="`0 0 ${W} ${H}`" role="img" :aria-label="label" preserveAspectRatio="xMidYMid meet">
      <defs>
        <pattern id="atp-hatch" width="6" height="6" patternUnits="userSpaceOnUse" patternTransform="rotate(45)">
          <line x1="0" y1="0" x2="0" y2="6" class="hatch-line" />
        </pattern>
      </defs>
      <g v-for="t in model.ticks" :key="t.v">
        <line :x1="PAD.l" :x2="W - PAD.r" :y1="t.y" :y2="t.y" class="grid" />
        <text :x="PAD.l - 4" :y="t.y + 4" class="axis num" text-anchor="end">{{ num0(t.v) }}</text>
      </g>
      <path v-for="(s, i) in model.shapes" :key="i" :d="s.d" :class="s.fresh ? 'gap-fresh' : 'gap-tired'" />
      <rect v-for="(b, i) in model.bars" :key="i" :x="b.x" :y="b.y" :width="b.w" :height="b.h" class="tss" />
      <path :d="model.ctl" class="line-ctl" />
      <path :d="model.atl" class="line-atl" />
      <text :x="PAD.l" :y="H - 2" class="axis num">{{ days[0] ? dayMonth(days[0].dia) : '' }}</text>
      <text :x="W - PAD.r" :y="H - 2" class="axis num" text-anchor="end">{{ last ? dayMonth(last.dia) : '' }}</text>
    </svg>
    <figcaption>
      <span class="key key-ctl">Forma (CTL)</span>
      <span class="key key-atl">Fadiga (ATL)</span>
      <span class="key key-fresh">Fôlego: área lisa, fresco</span>
      <span class="key key-tired">Fôlego: hachura, cansado</span>
    </figcaption>
    <details>
      <summary>Ver como tabela</summary>
      <table>
        <thead><tr><th>Dia</th><th>TSS</th><th>CTL</th><th>ATL</th><th>TSB</th></tr></thead>
        <tbody>
          <tr v-for="p in [...days].reverse().slice(0, 30)" :key="p.dia" class="num">
            <td>{{ dayMonth(p.dia) }}</td><td>{{ num0(p.tss) }}</td><td>{{ p.ctl.toFixed(1) }}</td><td>{{ p.atl.toFixed(1) }}</td><td>{{ p.tsb.toFixed(1) }}</td>
          </tr>
        </tbody>
      </table>
    </details>
  </figure>
</template>

<style scoped>
.folego { margin: 0; }
svg { width: 100%; height: auto; display: block; }
.grid { stroke: var(--atp-surface-variant); stroke-width: 1; }
.axis { fill: var(--atp-on-surface); font-size: 11px; opacity: 0.8; }
.line-ctl { fill: none; stroke: var(--atp-primary); stroke-width: 2; }
.line-atl { fill: none; stroke: var(--atp-accent); stroke-width: 2; stroke-dasharray: 5 3; }
.gap-fresh { fill: var(--atp-primary); fill-opacity: 0.18; stroke: var(--atp-gap-edge); stroke-width: 1; }
.gap-tired { fill: url(#atp-hatch); stroke: var(--atp-gap-edge); stroke-width: 1; }
.hatch-line { stroke: var(--atp-accent); stroke-width: 1.5; }
.tss { fill: var(--atp-surface-variant); }
figcaption { display: flex; flex-wrap: wrap; gap: 4px 16px; font-size: 13px; padding-top: 4px; }
.key::before { content: ''; display: inline-block; width: 14px; height: 0; margin-right: 6px; border-top: 2px solid; vertical-align: middle; }
.key-ctl::before { border-color: var(--atp-primary); }
.key-atl::before { border-color: var(--atp-accent); border-top-style: dashed; }
.key-fresh::before { height: 8px; border: 1px solid var(--atp-gap-edge); background: var(--atp-primary); opacity: 0.4; }
.key-tired::before { height: 8px; border: 1px solid var(--atp-gap-edge); background: repeating-linear-gradient(45deg, var(--atp-accent) 0 2px, transparent 2px 5px); }
details { margin-top: 8px; font-size: 13px; }
table { width: 100%; border-collapse: collapse; }
th, td { text-align: right; padding: 2px 6px; }
th:first-child, td:first-child { text-align: left; }
@media (prefers-reduced-motion: no-preference) {
  .gap-fresh, .gap-tired { animation: grow 300ms ease-out; transform-origin: center; }
  @keyframes grow { from { opacity: 0; } to { opacity: 1; } }
}
</style>
