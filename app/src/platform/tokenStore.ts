import { Capacitor } from '@capacitor/core';
import { SecureStorage } from '@aparajita/capacitor-secure-storage';

const KEY = 'access_token';

export interface TokenStore {
  get(): Promise<string | null>;
  set(token: string): Promise<void>;
  clear(): Promise<void>;
}

// Keychain no iOS, Keystore no Android: o JWT nunca toca localStorage nem Preferences.
export class SecureTokenStore implements TokenStore {
  private ready: Promise<void> | null = null;

  private init(): Promise<void> {
    this.ready ??= SecureStorage.setKeyPrefix('atp.');
    return this.ready;
  }
  async get(): Promise<string | null> {
    await this.init();
    return (await SecureStorage.getItem(KEY)) ?? null;
  }
  async set(token: string): Promise<void> {
    await this.init();
    await SecureStorage.setItem(KEY, token);
  }
  async clear(): Promise<void> {
    await this.init();
    await SecureStorage.removeItem(KEY);
  }
}

// Na web o token vive só na memória da aba: some ao recarregar, de propósito.
export class MemoryTokenStore implements TokenStore {
  private token: string | null = null;
  async get(): Promise<string | null> {
    return this.token;
  }
  async set(token: string): Promise<void> {
    this.token = token;
  }
  async clear(): Promise<void> {
    this.token = null;
  }
}

export function createTokenStore(): TokenStore {
  return Capacitor.isNativePlatform() ? new SecureTokenStore() : new MemoryTokenStore();
}
