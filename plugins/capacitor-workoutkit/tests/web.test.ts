import { describe, expect, it } from 'vitest';
import { WorkoutKitWeb } from '../src/web';

describe('plugin na web', () => {
  it('recusa em vez de fingir sucesso', () => {
    const p = new WorkoutKitWeb();
    expect(() => p.schedule()).toThrow(/iOS/);
    expect(() => p.syncWindow()).toThrow(/iOS/);
  });
});
