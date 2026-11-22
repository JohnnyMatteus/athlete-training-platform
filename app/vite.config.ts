import { fileURLToPath, URL } from 'node:url';
import vue from '@vitejs/plugin-vue';
import { defineConfig } from 'vitest/config';

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
      // O plugin nativo é consumido pelas definições TypeScript; a instalação nativa fica por conta do Capacitor.
      '@capacitor/core': fileURLToPath(new URL('./node_modules/@capacitor/core', import.meta.url)),
      '@atp/capacitor-workoutkit': fileURLToPath(
        new URL('../plugins/capacitor-workoutkit/src/index.ts', import.meta.url),
      ),
    },
  },
  server: { fs: { allow: ['..'] } },
  test: {
    environment: 'happy-dom',
    include: ['tests/**/*.test.ts', '../plugins/capacitor-workoutkit/tests/**/*.test.ts'],
    server: { deps: { inline: [/@ionic/] } },
  },
});
