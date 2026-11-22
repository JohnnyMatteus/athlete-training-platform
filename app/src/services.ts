import { createAtividadesApi } from '@/api/atividades';
import { createHttp } from '@/api/http';
import { createImportsApi } from '@/api/imports';
import { createMetricasApi } from '@/api/metricas';
import { createPlanejamentoApi } from '@/api/planejamento';
import { createCoachApi } from '@/api/coach';
import { createHealthIngestApi } from '@/api/proposed';
import { env } from '@/config/env';
import { CapgoHealthSource } from '@/features/health/capgoHealthSource';
import { AnchorStore, HealthIngestSink, HealthSyncService } from '@/features/health/syncService';
import { ImportHistory, ImportService, PendingUploads } from '@/features/imports/importService';
import { TusUploader } from '@/features/imports/tusUploader';
import { ConsentRepository } from '@/features/session/consent';
import { WatchSyncService } from '@/features/watch/watchSync';
import { PreferencesStore } from '@/platform/kv';
import { createTokenStore } from '@/platform/tokenStore';

// Composição única: troque adaptadores aqui (ou nos testes) sem tocar nos casos de uso.
export function createServices() {
  const kv = new PreferencesStore();
  const tokenStore = createTokenStore();
  let onUnauthorized: () => void = () => undefined;
  const http = (baseUrl: string) =>
    createHttp({ baseUrl, token: () => tokenStore.get(), onUnauthorized: () => onUnauthorized() });

  const importsHttp = http(env.VITE_API_URL);
  const consent = new ConsentRepository(kv);
  const importHistory = new ImportHistory(kv);

  return {
    kv,
    tokenStore,
    consent,
    importHistory,
    imports: new ImportService(
      createImportsApi(importsHttp),
      new TusUploader([env.VITE_TUS_URL]),
      consent,
      importHistory,
      new PendingUploads(kv),
      () => tokenStore.get(),
      [env.VITE_TUS_URL],
    ),
    metricas: createMetricasApi(http(env.VITE_METRICAS_URL)),
    planejamento: createPlanejamentoApi(http(env.VITE_PLANEJAMENTO_URL)),
    atividades: createAtividadesApi(http(env.VITE_ATIVIDADES_URL)),
    coach: createCoachApi(http(env.VITE_COACH_URL)),
    healthSync: new HealthSyncService(
      new CapgoHealthSource(),
      new AnchorStore(kv),
      new HealthIngestSink(createHealthIngestApi(importsHttp)),
      consent,
    ),
    watch: new WatchSyncService(),
    setUnauthorizedHandler: (fn: () => void) => {
      onUnauthorized = fn;
    },
  };
}

export type Services = ReturnType<typeof createServices>;
export const SERVICES_KEY = Symbol('services');
