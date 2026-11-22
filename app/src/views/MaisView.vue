<script setup lang="ts">
import { IonItem, IonLabel, IonList } from '@ionic/vue';
import PageShell from '@/components/PageShell.vue';

const links = [
  { to: '/importar', t: 'Importar treinos', d: 'export do app Saúde, .fit, .tcx, .gpx' },
  { to: '/limiares', t: 'Limiares e zonas', d: 'FTP, pace de limiar, CSS, perfil' },
  { to: '/calendario', t: 'Calendário', d: 'carga prevista por dia' },
  { to: '/coach', t: 'Coach', d: 'explica e propõe ajustes' },
  { to: '/privacidade', t: 'Privacidade e saúde', d: 'consentimento, leitura do HealthKit / Health Connect' },
];
</script>

<template>
  <PageShell title="Mais">
    <ion-list>
      <ion-item v-for="l in links" :key="l.to" :router-link="l.to" detail>
        <ion-label><h3>{{ l.t }}</h3><p>{{ l.d }}</p></ion-label>
      </ion-item>
    </ion-list>
  </PageShell>
</template>
