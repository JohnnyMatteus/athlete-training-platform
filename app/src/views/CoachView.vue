<script setup lang="ts">
import { computed, ref } from 'vue';
import { useMutation, useQueryClient } from '@tanstack/vue-query';
import { IonButton, IonSelect, IonSelectOption, IonTextarea } from '@ionic/vue';
import { ApiError, NetworkError } from '@/api/http';
import { intentSchema, type Intent } from '@/api/coach';
import PageShell from '@/components/PageShell.vue';
import RemoteState from '@/components/RemoteState.vue';
import { dayMonth } from '@/domain/format';
import { remoteKind, useRemote, useServices } from '@/composables';

const s = useServices();
const qc = useQueryClient();
const texto = ref('');
const intent = ref<Intent>('adjust_week');
const INTENTS: Record<Intent, string> = {
  adjust_week: 'Ajustar a semana',
  explain_session: 'Explicar uma sessão',
  weekly_summary: 'Resumo da semana',
  parse_goal: 'Interpretar uma meta',
};
const ask = useMutation({
  mutationFn: () => s.coach.ask(intentSchema.parse(intent.value), texto.value.trim()),
  onSuccess: () => qc.invalidateQueries({ queryKey: ['proposals'] }),
});
const proposals = useRemote(['proposals'], () => s.coach.proposals());
const kind = remoteKind(proposals);
const decide = useMutation({
  mutationFn: (v: { id: string; ok: boolean }) => (v.ok ? s.coach.approve(v.id) : s.coach.reject(v.id)),
  onSettled: () => qc.invalidateQueries({ queryKey: ['proposals'] }),
});

const askUnavailable = computed(() => ask.error.value instanceof NetworkError);
const STATUS: Record<string, string> = {
  pending_athlete: 'aguardando você', pending_coach: 'aguardando o treinador', applied: 'aplicada', rejected: 'recusada', expired: 'expirada',
};
const ACTION: Record<string, string> = { move: 'mover', replace_intensity: 'trocar intensidade', skip: 'pular' };

function errText(e: unknown): string {
  if (!(e instanceof ApiError)) return 'Não foi possível concluir.';
  if (e.status === 429) return 'Limite diário de uso do coach atingido.';
  if (e.status === 409) return `Conflito: ${e.detail ?? 'o plano mudou ou a proposta expirou'}.`;
  return e.detail ?? 'Não foi possível concluir.';
}
</script>

<template>
  <PageShell title="Coach" back="/mais">
    <p>O coach explica o plano e propõe ajustes; quem decide é você (ou seu treinador). Ele não orienta sobre dor ou lesão: nesses casos, procure um profissional de saúde.</p>
    <form @submit.prevent="texto.trim() && ask.mutate()">
      <ion-select v-model="intent" label="O que você quer" interface="popover">
        <ion-select-option v-for="(l, k) in INTENTS" :key="k" :value="k">{{ l }}</ion-select-option>
      </ion-select>
      <ion-textarea v-model="texto" label="Conte como está" label-placement="stacked" :rows="3" :maxlength="1000" />
      <ion-button type="submit" :disabled="!texto.trim() || ask.isPending.value">Enviar</ion-button>
    </form>
    <RemoteState v-if="askUnavailable" kind="unavailable" what="as respostas do coach" />
    <p v-else-if="ask.isError.value" role="alert" class="err">{{ errText(ask.error.value) }}</p>
    <div v-if="ask.data.value" role="status" class="reply">
      <p>{{ ask.data.value.message }}</p>
      <ul v-if="ask.data.value.kind === 'no_valid_proposal'">
        <li v-for="v in ask.data.value.violations" :key="v.rule">{{ v.message }}</li>
      </ul>
      <p v-if="ask.data.value.kind === 'goal'" class="num">
        {{ ask.data.value.constraints.event_format }} em {{ dayMonth(ask.data.value.constraints.event_date) }}, {{ ask.data.value.constraints.weekly_hours }} h por semana
      </p>
    </div>

    <h2>Propostas</h2>
    <RemoteState :kind="kind" what="as propostas do coach" empty-text="Nenhuma proposta.">
      <template v-if="proposals.data.value?.items.length">
        <article v-for="p in proposals.data.value.items" :key="p.id" class="proposal">
          <strong>{{ STATUS[p.status] }}</strong>
          <p>{{ p.rationale }}</p>
          <ul>
            <li v-for="c in p.changes" :key="c.session_id">
              {{ ACTION[c.action] }}<span v-if="c.new_date"> para <span class="num">{{ dayMonth(c.new_date) }}</span></span><span v-if="c.new_intensity"> ({{ c.new_intensity }})</span>
            </li>
          </ul>
          <p class="hint">Validada pelo servidor antes de mostrar e de novo ao aprovar. Expira em <span class="num">{{ dayMonth(p.expiresAt) }}</span>.</p>
          <template v-if="p.status === 'pending_athlete'">
            <ion-button :disabled="decide.isPending.value" @click="decide.mutate({ id: p.id, ok: true })">Aprovar</ion-button>
            <ion-button fill="outline" :disabled="decide.isPending.value" @click="decide.mutate({ id: p.id, ok: false })">Recusar</ion-button>
          </template>
        </article>
      </template>
      <p v-else>Nenhuma proposta.</p>
    </RemoteState>
    <p v-if="decide.isError.value" role="alert" class="err">{{ errText(decide.error.value) }}</p>
  </PageShell>
</template>

<style scoped>
.proposal { border-left: 3px solid var(--atp-primary); padding-left: 12px; margin-bottom: 16px; }
.err { color: var(--atp-error); }
.hint { opacity: 0.8; font-size: 14px; }
</style>
