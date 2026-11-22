import { defineStore } from 'pinia';
import type { TokenStore } from '@/platform/tokenStore';

const JWT_SHAPE = /^[\w-]+\.[\w-]+\.[\w-]+$/;

export const useSessionStore = defineStore('session', {
  state: () => ({ authenticated: false, ready: false }),
  actions: {
    async restore(store: TokenStore) {
      this.authenticated = (await store.get()) !== null;
      this.ready = true;
    },
    // A API valida a assinatura (ADR 0011); aqui só se recusa lixo óbvio antes de guardar.
    async signIn(store: TokenStore, token: string) {
      const t = token.trim();
      if (!JWT_SHAPE.test(t)) throw new Error('Isso não parece um JWT (três partes separadas por ponto).');
      await store.set(t);
      this.authenticated = true;
    },
    async signOut(store: TokenStore) {
      await store.clear();
      this.authenticated = false;
    },
  },
});
