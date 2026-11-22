import { registerPlugin } from '@capacitor/core';
import type { WorkoutKitPlugin } from './definitions';

export * from './definitions';

export const WorkoutKit = registerPlugin<WorkoutKitPlugin>('WorkoutKit', {
  web: () => import('./web').then((m) => new m.WorkoutKitWeb()),
});
