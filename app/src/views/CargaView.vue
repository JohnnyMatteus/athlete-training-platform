<script setup lang="ts">
import { ref } from 'vue';
import { IonSegment, IonSegmentButton, IonLabel } from '@ionic/vue';
import FolegoChart from '@/components/FolegoChart.vue';
import PageShell from '@/components/PageShell.vue';
import RemoteState from '@/components/RemoteState.vue';
import TsbRule from '@/components/TsbRule.vue';
import { num1 } from '@/domain/format';
import { useLoad } from '@/queries';

const span = ref(42);
const load = useLoad(span);
</script>

<template>
  <PageShell title="Carga">
    <ion-segment :value="String(span)" @ion-change="span = Number($event.detail.value)">
      <ion-segment-button value="42"><ion-label>6 semanas</ion-label></ion-segment-button>
      <ion-segment-button value="180"><ion-label>6 meses</ion-label></ion-segment-button>
      <ion-segment-button value="365"><ion-label>1 ano</ion-label></ion-segment-button>
    </ion-segment>
    <RemoteState :kind="load.kind.value" what="a carga diária" empty-text="Sem carga calculada neste período.">
      <template v-if="load.days.value.length">
        <p class="nums num">
          CTL {{ num1(load.days.value.at(-1)!.ctl) }} · ATL {{ num1(load.days.value.at(-1)!.atl) }}
        </p>
        <TsbRule :tsb="load.days.value.at(-1)!.tsb" />
        <FolegoChart :days="load.days.value" />
      </template>
      <p v-else>Sem carga calculada neste período.</p>
    </RemoteState>
  </PageShell>
</template>
