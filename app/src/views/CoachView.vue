<script setup lang="ts">
import { ref } from 'vue';
import { useMutation } from '@tanstack/vue-query';
import { IonButton, IonTextarea } from '@ionic/vue';
import { EndpointUnavailableError, NetworkError } from '@/api/http';
import PageShell from '@/components/PageShell.vue';
import RemoteState from '@/components/RemoteState.vue';
import { useServices } from '@/composables';

const s = useServices();
const texto = ref('');
const ask = useMutation({ mutationFn: (t: string) => s.coach.ask(t) });
const unavailable = () => ask.error.value instanceof EndpointUnavailableError || ask.error.value instanceof NetworkError;
</script>

<template>
  <PageShell title="Coach" back="/mais">
    <p>O coach explica o plano e propõe ajustes. Ele não prescreve para dor ou lesão: nesses casos, procure um profissional de saúde.</p>
    <form @submit.prevent="texto.trim() && ask.mutate(texto.trim())">
      <ion-textarea v-model="texto" label="Pergunte ou relate como está" label-placement="stacked" :rows="3" />
      <ion-button type="submit" :disabled="!texto.trim() || ask.isPending.value">Enviar</ion-button>
    </form>
    <RemoteState v-if="ask.isError.value" :kind="unavailable() ? 'unavailable' : 'error'" what="as respostas do coach" />
    <div v-if="ask.data.value" role="status">
      <p>{{ ask.data.value.resposta }}</p>
      <div v-if="ask.data.value.proposta" class="proposal">
        <strong>Proposta de ajuste</strong>
        <p>{{ ask.data.value.proposta.resumo }}</p>
        <p>Regras do validador que ela passou: {{ ask.data.value.proposta.regras_validadas.join(', ') || 'nenhuma listada' }}.</p>
        <p v-if="ask.data.value.proposta.requer_aprovacao_treinador">Depende de aprovação do seu treinador.</p>
      </div>
    </div>
  </PageShell>
</template>

<style scoped>.proposal { border-left: 3px solid var(--atp-primary); padding-left: 12px; }</style>
