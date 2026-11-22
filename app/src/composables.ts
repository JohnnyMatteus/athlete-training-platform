import { useQuery } from '@tanstack/vue-query';
import { computed, inject, type Ref } from 'vue';
import { ApiError, ContractError, EndpointUnavailableError, NetworkError } from '@/api/http';
import { SERVICES_KEY, type Services } from '@/services';

export function useServices(): Services {
  const s = inject<Services>(SERVICES_KEY);
  if (!s) throw new Error('Services não providos');
  return s;
}

// Sem retry para o que repetir não conserta: endpoint ausente, contrato quebrado, erro 4xx.
const shouldRetry = (count: number, e: unknown) =>
  !(e instanceof EndpointUnavailableError || e instanceof NetworkError || e instanceof ContractError || (e instanceof ApiError && e.status < 500)) &&
  count < 2;

export function useRemote<T>(key: unknown[], fn: () => Promise<T>, enabled: Ref<boolean> | boolean = true) {
  return useQuery({ queryKey: key, queryFn: fn, retry: shouldRetry, staleTime: 60_000, enabled });
}

export type RemoteKind = 'loading' | 'unavailable' | 'empty' | 'contract' | 'error' | 'ready';

export function remoteKind(q: { isPending: Ref<boolean>; error: Ref<unknown> }) {
  return computed<RemoteKind>(() => {
    const e = q.error.value;
    // Serviço fora do ar ou endpoint ainda não publicado: a tela diz isso, sem inventar dado.
    if (e instanceof EndpointUnavailableError || e instanceof NetworkError) return 'unavailable';
    if (e instanceof ApiError && e.status === 404) return 'empty';
    if (e instanceof ContractError) return 'contract';
    if (e) return 'error';
    return q.isPending.value ? 'loading' : 'ready';
  });
}
