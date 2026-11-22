import '@ionic/vue/css/core.css';
import '@ionic/vue/css/normalize.css';
import '@ionic/vue/css/structure.css';
import '@ionic/vue/css/typography.css';
import '@fontsource/saira-semi-condensed/600.css';
import '@fontsource/atkinson-hyperlegible/400.css';
import '@fontsource/atkinson-hyperlegible/700.css';
import '@fontsource/jetbrains-mono/500.css';
import '@/theme/tokens.css';

import { IonicVue } from '@ionic/vue';
import { VueQueryPlugin } from '@tanstack/vue-query';
import { createPinia } from 'pinia';
import { createApp } from 'vue';
import App from './App.vue';
import { createAppRouter } from './router';
import { createServices, SERVICES_KEY } from './services';
import { useSessionStore } from './features/session/sessionStore';

const pinia = createPinia();
const services = createServices();
const session = useSessionStore(pinia);
services.setUnauthorizedHandler(() => void session.signOut(services.tokenStore));

const router = createAppRouter(pinia);
createApp(App)
  .use(pinia)
  .use(VueQueryPlugin)
  .use(IonicVue)
  .use(router)
  .provide(SERVICES_KEY, services)
  .mount('#app');

session.restore(services.tokenStore).then(() => router.isReady().then(() => router.replace(router.currentRoute.value.fullPath)));
