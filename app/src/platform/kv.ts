import { Preferences } from '@capacitor/preferences';

// Armazenamento de preferências não secretas (âncora, consentimento). Segredos usam TokenStore.
export interface KeyValueStore {
  get(key: string): Promise<string | null>;
  set(key: string, value: string): Promise<void>;
  remove(key: string): Promise<void>;
}

export class PreferencesStore implements KeyValueStore {
  async get(key: string): Promise<string | null> {
    return (await Preferences.get({ key })).value;
  }
  async set(key: string, value: string): Promise<void> {
    await Preferences.set({ key, value });
  }
  async remove(key: string): Promise<void> {
    await Preferences.remove({ key });
  }
}

export class MemoryStore implements KeyValueStore {
  private readonly data = new Map<string, string>();
  async get(key: string): Promise<string | null> {
    return this.data.get(key) ?? null;
  }
  async set(key: string, value: string): Promise<void> {
    this.data.set(key, value);
  }
  async remove(key: string): Promise<void> {
    this.data.delete(key);
  }
}
