<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { useQuery } from '@tanstack/vue-query';
import { IonButton, IonCheckbox, IonItem, IonList, IonProgressBar } from '@ionic/vue';
import { ApiError } from '@/api/http';
import { TERMINAL_STATUSES, type ConsentCategory } from '@/api/schemas';
import PageShell from '@/components/PageShell.vue';
import { bytes, num0 } from '@/domain/format';
import { useServices } from '@/composables';
import { CATEGORY_LABELS, CURRENT_TERM_VERSION } from '@/features/session/consent';
import type { ImportProgress } from '@/features/imports/importService';

const s = useServices();
const categories = ref<ConsentCategory[]>(['treinos']);
const hasConsent = ref(false);
const file = ref<File | null>(null);
const progress = ref<ImportProgress | null>(null);
const activeId = ref<string | null>(null);
const error = ref<string | null>(null);
const history = ref<string[]>([]);
let abort: (() => Promise<void>) | null = null;

onMounted(async () => {
  hasConsent.value = (await s.consent.current()) !== null;
  history.value = await s.importHistory.ids();
});

async function accept() {
  await s.consent.accept(categories.value);
  hasConsent.value = true;
}

function pick(e: Event) {
  file.value = (e.target as HTMLInputElement).files?.[0] ?? null;
  error.value = null;
}

async function send() {
  if (!file.value) return;
  error.value = null;
  try {
    const { importId, handle } = await s.imports.start(file.value, (p) => (progress.value = p));
    activeId.value = importId;
    history.value = await s.importHistory.ids();
    abort = handle.abort;
    handle.done.catch((e: Error) => (error.value = `Envio interrompido: ${e.message}. Selecione o mesmo arquivo para retomar de onde parou.`));
  } catch (e) {
    error.value = e instanceof ApiError ? (e.problem?.errors?.map((x) => x.mensagem).join(' ') ?? e.message) : (e as Error).message;
  }
}

const batch = useQuery({
  queryKey: ['import', activeId],
  queryFn: () => s.imports.get(activeId.value as string),
  enabled: computed(() => Boolean(activeId.value) && progress.value?.phase === 'enviado'),
  refetchInterval: (q) => (q.state.data && TERMINAL_STATUSES.includes(q.state.data.status) ? false : 3000),
});

const PHASE = { hash: 'Calculando a impressão digital do arquivo', criando: 'Registrando o lote', enviando: 'Enviando', enviado: 'Enviado' } as const;
const STATUS: Record<string, string> = {
  aguardando_upload: 'aguardando o envio', recebido: 'recebido', processando: 'processando', concluido: 'concluído', falhou: 'falhou',
};
</script>

<template>
  <PageShell title="Importar" back="/mais">
    <section v-if="!hasConsent">
      <h2>1. Consentimento</h2>
      <p>Seus dados de treino são dados de saúde. Escolha o que a plataforma pode processar (termo {{ CURRENT_TERM_VERSION }}); você pode revogar depois em Privacidade.</p>
      <ion-list>
        <ion-item v-for="(l, k) in CATEGORY_LABELS" :key="k">
          <ion-checkbox :checked="categories.includes(k)" label-placement="end" justify="start"
            @ion-change="categories = $event.detail.checked ? [...categories, k] : categories.filter((c) => c !== k)">
            <strong>{{ l.titulo }}</strong><br /><small>{{ l.detalhe }}</small>
          </ion-checkbox>
        </ion-item>
      </ion-list>
      <ion-button :disabled="categories.length === 0" @click="accept">Aceitar e continuar</ion-button>
    </section>

    <section v-else>
      <h2>2. Escolher o arquivo</h2>
      <p>Export do app Saúde (.zip), ou arquivos .fit, .tcx e .gpx. Se a conexão cair, escolha o mesmo arquivo de novo: o envio retoma.</p>
      <input type="file" accept=".zip,.fit,.tcx,.gpx" aria-label="Arquivo de treino" @change="pick" />
      <p v-if="file" class="num">{{ file.name }} · {{ bytes(file.size) }}</p>
      <ion-button :disabled="!file || (progress !== null && progress.phase !== 'enviado')" @click="send">Enviar</ion-button>
      <ion-button v-if="abort && progress?.phase === 'enviando'" fill="outline" @click="abort?.()">Pausar</ion-button>

      <div v-if="progress" class="progress" role="status">
        <p>{{ PHASE[progress.phase] }} <span class="num">{{ num0(progress.fraction * 100) }}%</span></p>
        <ion-progress-bar :value="progress.fraction" />
      </div>
      <p v-if="error" role="alert" class="err">{{ error }}</p>

      <div v-if="batch.data.value">
        <h2>3. Resultado</h2>
        <p>Lote {{ STATUS[batch.data.value.status] }}.</p>
        <p class="num">
          importadas {{ batch.data.value.importadas }} · duplicadas {{ batch.data.value.duplicadas }} · erros {{ batch.data.value.erros }}
        </p>
        <p v-if="batch.data.value.motivo_falha" class="err">{{ batch.data.value.motivo_falha }}</p>
      </div>
      <p v-if="history.length" class="hint">{{ history.length }} envio(s) anteriores neste aparelho.</p>
    </section>
  </PageShell>
</template>

<style scoped>
.err { color: var(--atp-error); }
.hint { opacity: 0.8; }
.progress { margin: 16px 0; }
</style>
