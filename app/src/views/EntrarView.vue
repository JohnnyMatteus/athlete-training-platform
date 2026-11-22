<script setup lang="ts">
import { ref } from 'vue';
import { useRouter } from 'vue-router';
import { IonButton, IonTextarea } from '@ionic/vue';
import PageShell from '@/components/PageShell.vue';
import { useServices } from '@/composables';
import { useSessionStore } from '@/features/session/sessionStore';

const s = useServices();
const session = useSessionStore();
const router = useRouter();
const token = ref('');
const error = ref<string | null>(null);
// Sem provedor de identidade escolhido (ADR 0011), só o build de desenvolvimento aceita um token colado.
const dev = import.meta.env.DEV;

async function enter() {
  try {
    await session.signIn(s.tokenStore, token.value);
    token.value = '';
    await router.replace('/hoje');
  } catch (e) {
    error.value = (e as Error).message;
  }
}
</script>

<template>
  <PageShell title="Entrar">
    <template v-if="dev">
      <p>Build de desenvolvimento: cole um JWT gerado pelo script local de chaves de teste.</p>
      <ion-textarea v-model="token" label="JWT" label-placement="stacked" :rows="4" autocomplete="off" />
      <ion-button :disabled="!token.trim()" @click="enter">Entrar</ion-button>
      <p v-if="error" role="alert">{{ error }}</p>
    </template>
    <p v-else data-testid="sem-provedor">
      O login ainda não está disponível: o provedor de identidade não foi escolhido (ADR 0011). Quando for, esta tela abre o fluxo dele.
    </p>
  </PageShell>
</template>
