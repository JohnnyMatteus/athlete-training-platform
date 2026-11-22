import { createRouter, createWebHistory } from '@ionic/vue-router';
import type { Pinia } from 'pinia';
import { useSessionStore } from '@/features/session/sessionStore';

export function createAppRouter(pinia: Pinia) {
  const router = createRouter({
    history: createWebHistory(),
    routes: [
      { path: '/', redirect: '/hoje' },
      { path: '/entrar', component: () => import('@/views/EntrarView.vue') },
      { path: '/hoje', component: () => import('@/views/HojeView.vue') },
      { path: '/carga', component: () => import('@/views/CargaView.vue') },
      { path: '/atividades', component: () => import('@/views/AtividadesView.vue') },
      { path: '/plano', component: () => import('@/views/PlanoView.vue') },
      { path: '/mais', component: () => import('@/views/MaisView.vue') },
      { path: '/importar', component: () => import('@/views/ImportarView.vue') },
      { path: '/limiares', component: () => import('@/views/LimiaresView.vue') },
      { path: '/calendario', component: () => import('@/views/CalendarioView.vue') },
      { path: '/coach', component: () => import('@/views/CoachView.vue') },
      { path: '/privacidade', component: () => import('@/views/PrivacidadeView.vue') },
    ],
  });
  router.beforeEach((to) => {
    const session = useSessionStore(pinia);
    if (!session.authenticated && to.path !== '/entrar') return '/entrar';
    if (session.authenticated && to.path === '/entrar') return '/hoje';
    return true;
  });
  return router;
}
