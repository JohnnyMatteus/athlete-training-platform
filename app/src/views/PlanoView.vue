<script setup lang="ts">
import { computed, ref } from 'vue';
import { IonButton, IonItem, IonList } from '@ionic/vue';
import PageShell from '@/components/PageShell.vue';
import RemoteState from '@/components/RemoteState.vue';
import { dayMonth, duration, MODALIDADE_LABEL, num0 } from '@/domain/format';
import { useServices } from '@/composables';
import { scheduleFromPlan } from '@/features/watch/toPlannedWorkout';
import type { WatchSyncOutcome } from '@/features/watch/watchSync';
import { useCurrentPlan, useLimits } from '@/queries';

const s = useServices();
const cur = useCurrentPlan();
const { limits } = useLimits();
const outcome = ref<WatchSyncOutcome | null>(null);
const failure = ref<string | null>(null);
const busy = ref(false);

const weeks = computed(() => cur.plan.value?.mesociclos.flatMap((m) => m.microciclos.map((c) => ({ ...c, fase: m.fase }))) ?? []);
const canSend = computed(() => cur.plan.value?.status === 'ativo' && s.watch.supported());
const exportable = computed(
  () => new Map(cur.plan.value && canSend.value ? scheduleFromPlan(cur.plan.value, limits.value).map((i) => [i.id, i.plano]) : []),
);

async function exportWorkout(id: string) {
  const plan = exportable.value.get(id);
  if (!plan) return;
  busy.value = true;
  failure.value = null;
  try {
    await s.watch.openOrExport(plan);
  } catch (e) {
    failure.value = (e as Error).message;
  } finally {
    busy.value = false;
  }
}

async function sendToWatch() {
  if (!cur.plan.value) return;
  busy.value = true;
  failure.value = null;
  try {
    outcome.value = await s.watch.sync(scheduleFromPlan(cur.plan.value, limits.value));
  } catch (e) {
    failure.value = (e as Error).message;
  } finally {
    busy.value = false;
  }
}
</script>

<template>
  <PageShell title="Plano">
    <RemoteState :kind="cur.kind.value" what="o plano de treino" empty-text="Você ainda não tem plano.">
      <template v-if="cur.plan.value">
        <p>
          Meta: {{ cur.plan.value.prova.replace(/_/g, ' ') }} em <span class="num">{{ dayMonth(cur.plan.value.data_prova) }}</span>
          · versão {{ cur.plan.value.versao_plano }}
        </p>
        <p v-if="cur.plan.value.status !== 'ativo'" class="warn">
          {{ cur.plan.value.status === 'aguardando_aprovacao' ? 'Aguardando aprovação do treinador: nada é enviado ao relógio antes disso.' : 'Plano rejeitado pelo treinador.' }}
        </p>
        <ul v-if="cur.plan.value.avisos.length" class="avisos">
          <li v-for="(a, i) in cur.plan.value.avisos" :key="i">{{ a.mensagem }}</li>
        </ul>

        <div class="sync">
          <ion-button :disabled="!canSend || busy" @click="sendToWatch">Enviar ao relógio</ion-button>
          <p class="hint">
            {{ s.watch.supported()
              ? 'Só os próximos dias cabem no relógio (janela deslizante, limite de planos do WorkoutKit); o resto entra a cada abertura do app.'
              : 'O envio ao Apple Watch só existe no app para iOS 17+.' }}
          </p>
          <p v-if="outcome?.status === 'autorizacao_negada'" class="warn">Autorização do WorkoutKit negada nos Ajustes do iPhone.</p>
          <p v-if="outcome?.status === 'ok'" class="num" role="status">
            agendados {{ outcome.result.added.length }} · removidos {{ outcome.result.removed.length }} · aguardando vaga {{ outcome.result.deferred.length }} · falhas {{ outcome.result.failed.length }}
          </p>
          <p v-if="failure" role="alert" class="warn">{{ failure }}</p>
        </div>

        <section v-for="w in weeks" :key="w.semana">
          <h3>Semana {{ w.semana }} · {{ w.fase }}{{ w.descarga ? ' · descarga' : '' }}
            <span class="num">{{ num0(w.tss_planejado) }} TSS, CTL {{ num0(w.ctl_projetado) }}</span></h3>
          <ion-list>
            <ion-item v-for="t in w.treinos.filter((x) => x.intensidade !== 'descanso')" :key="t.id">
              <span><span class="num">{{ dayMonth(t.data) }}</span> {{ MODALIDADE_LABEL[t.modalidade] }}: {{ t.nome }}</span>
              <span slot="end" class="num">{{ duration(t.duracao_s) }} · {{ num0(t.tss_planejado) }}</span>
              <ion-button
                v-if="exportable.has(t.id)"
                slot="end"
                fill="clear"
                :disabled="busy"
                :aria-label="`Exportar .fit de ${t.nome}, ${dayMonth(t.data)}`"
                @click="exportWorkout(t.id)"
              >
                .fit
              </ion-button>
            </ion-item>
          </ion-list>
        </section>
      </template>
      <p v-else>Você ainda não tem plano.</p>
    </RemoteState>
  </PageShell>
</template>

<style scoped>
.warn { color: var(--atp-warning); }
.hint { opacity: 0.8; font-size: 14px; }
.sync { margin: 12px 0 20px; }
section h3 { margin-top: 20px; font-size: 17px; }
.avisos { color: var(--atp-warning); }
</style>
