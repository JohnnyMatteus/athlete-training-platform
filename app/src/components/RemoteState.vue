<script setup lang="ts">
import { IonSpinner } from '@ionic/vue';
import type { RemoteKind } from '@/composables';

defineProps<{ kind: RemoteKind; what: string; emptyText?: string }>();
</script>

<template>
  <div v-if="kind === 'loading'" class="state" role="status">
    <ion-spinner name="dots" /> <span>Carregando {{ what }}…</span>
  </div>
  <div v-else-if="kind === 'unavailable'" class="state unavailable" role="status" data-testid="indisponivel">
    <h3>Indisponível por enquanto</h3>
    <p>
      O serviço que entrega {{ what }} não respondeu ou ainda não publicou este endpoint. Nada é estimado no lugar:
      quando ele estiver no ar, esta tela se preenche sozinha.
    </p>
  </div>
  <div v-else-if="kind === 'empty'" class="state" role="status">
    <h3>Nada por aqui ainda</h3>
    <p>{{ emptyText ?? `Ainda não há ${what}.` }}</p>
  </div>
  <div v-else-if="kind === 'contract'" class="state error" role="alert">
    <h3>Resposta fora do contrato</h3>
    <p>O servidor devolveu {{ what }} num formato que o app não reconhece. Atualize o app ou avise quem mantém a API.</p>
  </div>
  <div v-else-if="kind === 'error'" class="state error" role="alert">
    <h3>Não foi possível carregar</h3>
    <p>Tente de novo em instantes.</p>
  </div>
  <slot v-else />
</template>

<style scoped>
.state { padding: 16px; border: 1px dashed var(--atp-gap-edge); border-radius: 4px; background: var(--atp-surface); }
.state h3 { margin: 0 0 4px; font-size: 18px; }
.state p { margin: 0; opacity: 0.85; }
.error h3 { color: var(--atp-error); }
</style>
