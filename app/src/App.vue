<script setup lang="ts">
import { IonApp, IonIcon, IonLabel, IonRouterOutlet, IonTabBar, IonTabButton, IonTabs } from '@ionic/vue';
import { barChartOutline, calendarOutline, ellipsisHorizontalOutline, pulseOutline, todayOutline } from 'ionicons/icons';
import { useSessionStore } from '@/features/session/sessionStore';

const session = useSessionStore();
</script>

<template>
  <ion-app>
    <ion-tabs v-if="session.authenticated">
      <ion-router-outlet />
      <ion-tab-bar slot="bottom">
        <ion-tab-button tab="hoje" href="/hoje"><ion-icon :icon="todayOutline" /><ion-label>Hoje</ion-label></ion-tab-button>
        <ion-tab-button tab="carga" href="/carga"><ion-icon :icon="pulseOutline" /><ion-label>Carga</ion-label></ion-tab-button>
        <ion-tab-button tab="atividades" href="/atividades"><ion-icon :icon="barChartOutline" /><ion-label>Atividades</ion-label></ion-tab-button>
        <ion-tab-button tab="plano" href="/plano"><ion-icon :icon="calendarOutline" /><ion-label>Plano</ion-label></ion-tab-button>
        <ion-tab-button tab="mais" href="/mais"><ion-icon :icon="ellipsisHorizontalOutline" /><ion-label>Mais</ion-label></ion-tab-button>
      </ion-tab-bar>
    </ion-tabs>
    <ion-router-outlet v-else />
  </ion-app>
</template>
