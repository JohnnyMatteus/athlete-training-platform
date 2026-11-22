<script setup lang="ts">
import { computed, reactive } from 'vue';
import { useMutation, useQueryClient } from '@tanstack/vue-query';
import { IonButton, IonInput, IonItem, IonList, IonSelect, IonSelectOption } from '@ionic/vue';
import type { Threshold } from '@/api/metricas';
import PageShell from '@/components/PageShell.vue';
import RemoteState from '@/components/RemoteState.vue';
import { dayMonth, isoDay, num1 } from '@/domain/format';
import { remoteKind, useRemote, useServices } from '@/composables';

const s = useServices();
const qc = useQueryClient();
const thresholds = useRemote(['thresholds'], () => s.metricas.thresholds());
const tKind = remoteKind(thresholds);
const profile = useRemote(['profile'], () => s.metricas.profile());
const pKind = remoteKind(profile);

const LABEL: Record<Threshold['tipo'], { nome: string; unidade: string }> = {
  ftp: { nome: 'FTP (ciclismo)', unidade: 'W' },
  pace_limiar: { nome: 'Pace de limiar (corrida)', unidade: 's/km' },
  css: { nome: 'CSS (natação)', unidade: 'm/s' },
};

// Versionado no tempo: o TSS de uma atividade usa o limiar vigente na data dela (design doc, seção 6).
const history = computed(() => [...(thresholds.data.value?.itens ?? [])].sort((a, b) => b.vigente_desde.localeCompare(a.vigente_desde)));

const form = reactive<{ tipo: Threshold['tipo']; valor: string; desde: string }>({ tipo: 'ftp', valor: '', desde: isoDay(new Date()) });
const save = useMutation({
  mutationFn: () => s.metricas.putThreshold({ tipo: form.tipo, valor: Number(form.valor.replace(',', '.')), vigente_desde: form.desde }),
  onSuccess: () => {
    form.valor = '';
    return qc.invalidateQueries({ queryKey: ['thresholds'] });
  },
});
const valid = computed(() => Number(form.valor.replace(',', '.')) > 0 && /^\d{4}-\d{2}-\d{2}$/.test(form.desde));
</script>

<template>
  <PageShell title="Limiares" back="/mais">
    <h2>Vigentes e histórico</h2>
    <RemoteState :kind="tKind" what="os limiares" empty-text="Nenhum limiar registrado. Sem eles o plano não consegue traduzir % em watts, pace ou bpm.">
      <ion-list v-if="history.length">
        <ion-item v-for="t in history" :key="t.tipo + t.vigente_desde">
          <span>{{ LABEL[t.tipo].nome }}</span>
          <span slot="end" class="num">{{ num1(t.valor) }} {{ LABEL[t.tipo].unidade }} desde {{ dayMonth(t.vigente_desde) }}</span>
        </ion-item>
      </ion-list>
      <p v-else>Nenhum limiar registrado. Sem eles o plano não consegue traduzir % em watts, pace ou bpm.</p>
    </RemoteState>

    <h2>Registrar novo limiar</h2>
    <form @submit.prevent="save.mutate()">
      <ion-item>
        <ion-select v-model="form.tipo" label="Tipo" interface="popover">
          <ion-select-option v-for="(l, k) in LABEL" :key="k" :value="k">{{ l.nome }}</ion-select-option>
        </ion-select>
      </ion-item>
      <ion-item><ion-input v-model="form.valor" :label="`Valor (${LABEL[form.tipo].unidade})`" inputmode="decimal" /></ion-item>
      <ion-item><ion-input v-model="form.desde" label="Vigente desde (AAAA-MM-DD)" /></ion-item>
      <ion-button type="submit" :disabled="!valid || save.isPending.value">Salvar limiar</ion-button>
      <p v-if="save.isError.value" role="alert" class="err">Não foi possível salvar o limiar.</p>
      <p class="hint">O histórico não é recalculado: atividades antigas mantêm o limiar da época. Para recalcular, corrija a atividade.</p>
    </form>

    <h2>Perfil fisiológico</h2>
    <RemoteState :kind="pKind" what="o perfil fisiológico" empty-text="Perfil ainda não preenchido (sexo, FC de repouso e máxima).">
      <p v-if="profile.data.value" class="num">FC repouso {{ profile.data.value.fc_repouso }} bpm · FC máx {{ profile.data.value.fc_max }} bpm</p>
    </RemoteState>

    <h2>Zonas</h2>
    <RemoteState kind="unavailable" what="as zonas de FC, potência e pace" />
  </PageShell>
</template>

<style scoped>
.err { color: var(--atp-error); }
.hint { opacity: 0.8; font-size: 14px; }
</style>
