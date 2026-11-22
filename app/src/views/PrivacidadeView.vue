<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { IonButton } from '@ionic/vue';
import PageShell from '@/components/PageShell.vue';
import { CATEGORY_LABELS } from '@/features/session/consent';
import type { SyncOutcome } from '@/features/health/syncService';
import { useServices } from '@/composables';
import { useSessionStore } from '@/features/session/sessionStore';
import { useRouter } from 'vue-router';

const s = useServices();
const session = useSessionStore();
const router = useRouter();
const current = ref<Awaited<ReturnType<typeof s.consent.current>>>(null);
const outcome = ref<SyncOutcome | null>(null);
const busy = ref(false);

onMounted(async () => (current.value = await s.consent.current()));

async function revoke() {
  await s.consent.revoke();
  current.value = null;
}
async function sync() {
  busy.value = true;
  try {
    outcome.value = await s.healthSync.run();
  } finally {
    busy.value = false;
  }
}
async function signOut() {
  await session.signOut(s.tokenStore);
  await router.replace('/entrar');
}

const MSG: Record<SyncOutcome['status'], string> = {
  ok: 'Sincronizado.',
  sem_consentimento: 'Aceite o consentimento de treinos (em Importar) antes de ler a saúde do aparelho.',
  indisponivel_plataforma: 'HealthKit e Health Connect só existem no app instalado (iOS ou Android).',
  permissao_negada: 'Permissão de leitura negada nos Ajustes do sistema.',
  servidor_indisponivel: 'A API ainda não publica o endpoint que recebe treinos do HealthKit / Health Connect. Nada foi marcado como enviado; a âncora não avançou.',
};
</script>

<template>
  <PageShell title="Privacidade" back="/mais">
    <h2>Consentimento</h2>
    <template v-if="current">
      <p>Termo {{ current.versao_termo }}, aceito em <span class="num">{{ current.aceito_em.slice(0, 10) }}</span>.</p>
      <ul><li v-for="c in current.categorias" :key="c">{{ CATEGORY_LABELS[c].titulo }}</li></ul>
      <ion-button fill="outline" @click="revoke">Revogar consentimento</ion-button>
    </template>
    <p v-else>Sem consentimento ativo. Nenhum dado é enviado. Você aceita ao importar.</p>

    <h2>Leitura do aparelho</h2>
    <p>Lê só o que mudou desde a última sincronização, usando uma âncora guardada no aparelho.</p>
    <ion-button :disabled="busy" @click="sync">Sincronizar agora</ion-button>
    <p v-if="outcome" role="status">{{ MSG[outcome.status] }}<span v-if="'sent' in outcome" class="num"> ({{ outcome.sent }} treinos)</span></p>

    <h2>Conta</h2>
    <ion-button fill="outline" color="danger" @click="signOut">Sair</ion-button>
    <p class="hint">Apagar a conta e exportar os dados dependem de endpoints que o contrato ainda não define.</p>
  </PageShell>
</template>

<style scoped>.hint { opacity: 0.8; font-size: 14px; }</style>
