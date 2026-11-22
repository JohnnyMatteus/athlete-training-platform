<script setup lang="ts">
import { computed } from 'vue';
import { IonItem, IonLabel, IonList } from '@ionic/vue';
import PageShell from '@/components/PageShell.vue';
import RemoteState from '@/components/RemoteState.vue';
import { addDays, dayMonth, isoDay, num0 } from '@/domain/format';
import { remoteKind, useRemote, useServices } from '@/composables';

const s = useServices();
const to = isoDay(new Date());
const q = useRemote(['activities'], () => s.metricas.activityLoads({ de: addDays(to, -60), ate: to }));
const kind = remoteKind(q);
const items = computed(() => [...(q.data.value?.itens ?? [])].sort((a, b) => b.dia.localeCompare(a.dia)));

const METODO: Record<string, string> = {
  tss_potencia: 'TSS por potência',
  rtss: 'rTSS (corrida)',
  stss: 'sTSS (natação)',
  trimp: 'TRIMP (só FC)',
  indisponivel: 'sem dado para calcular',
};
</script>

<template>
  <PageShell title="Atividades">
    <p class="hint">Carga de cada atividade nos últimos 60 dias. Duração e distância ainda não têm endpoint de leitura para o app.</p>
    <RemoteState :kind="kind" what="as atividades" empty-text="Nenhuma atividade importada ainda. Use Importar para trazer seus treinos.">
      <ion-list v-if="items.length">
        <ion-item v-for="a in items" :key="a.atividade_id">
          <ion-label>
            <h3>{{ dayMonth(a.dia) }} · {{ a.tipo }}</h3>
            <p>{{ METODO[a.metodo] }}<span v-if="a.descartes_fc + a.descartes_potencia"> · {{ a.descartes_fc + a.descartes_potencia }} amostras descartadas</span></p>
          </ion-label>
          <span slot="end" class="num">{{ num0(a.tss) }} TSS</span>
        </ion-item>
      </ion-list>
      <p v-else>Nenhuma atividade importada ainda. Use Importar para trazer seus treinos.</p>
    </RemoteState>
  </PageShell>
</template>

<style scoped>.hint { opacity: 0.85; }</style>
