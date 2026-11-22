import { WebPlugin } from '@capacitor/core';
import type { WorkoutKitPlugin } from './definitions';

// WorkoutKit só existe no iOS: a web recusa explicitamente em vez de fingir sucesso.
export class WorkoutKitWeb extends WebPlugin implements WorkoutKitPlugin {
  private unsupported(): never {
    throw this.unavailable('WorkoutKit só está disponível no iOS 17+ com Apple Watch.');
  }
  requestAuthorization(): never { return this.unsupported(); }
  schedule(): never { return this.unsupported(); }
  listScheduled(): never { return this.unsupported(); }
  remove(): never { return this.unsupported(); }
  removeAll(): never { return this.unsupported(); }
  openInWorkoutApp(): never { return this.unsupported(); }
  syncWindow(): never { return this.unsupported(); }
  exportFit(): never { return this.unsupported(); }
}
