export class UntrustedUploadOriginError extends Error {
  constructor(url: string) {
    super(`Servidor de upload não confiável: ${safeOrigin(url)}. O token não foi enviado.`);
  }
}

const LOCAL = new Set(['localhost', '127.0.0.1', '[::1]']);

function safeOrigin(url: string): string {
  try {
    return new URL(url).origin;
  } catch {
    return 'URL inválida';
  }
}

/*
 * O JWT só vai para origens configuradas (VITE_TUS_URL). A upload_url vem do corpo da resposta e a URL
 * retomada vem do armazenamento do tus: nenhuma das duas é confiável por si só. Fora de localhost exige https.
 */
export function isTrustedUploadUrl(url: string, allowedOrigins: readonly string[]): boolean {
  let u: URL;
  try {
    u = new URL(url);
  } catch {
    return false;
  }
  if (u.protocol !== 'https:' && !(u.protocol === 'http:' && LOCAL.has(u.hostname))) return false;
  return allowedOrigins.some((o) => new URL(o).origin === u.origin);
}

export function assertTrustedUploadUrl(url: string, allowedOrigins: readonly string[]): void {
  if (!isTrustedUploadUrl(url, allowedOrigins)) throw new UntrustedUploadOriginError(url);
}
