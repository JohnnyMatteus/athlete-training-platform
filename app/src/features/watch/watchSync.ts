import { Capacitor } from '@capacitor/core';
import { Filesystem, Directory } from '@capacitor/filesystem';
import { Share } from '@capacitor/share';
import { WorkoutKit, type PlannedWorkoutDTO, type SyncWindowResult, type WorkoutKitPlugin } from '@atp/capacitor-workoutkit';
import type { ScheduledItem } from '@/api/plannedWorkout';

export type WatchSyncOutcome =
  | { status: 'indisponivel_plataforma' }
  | { status: 'autorizacao_negada' }
  | { status: 'ok'; result: SyncWindowResult };

export interface FitSharer {
  share(filename: string, base64: string): Promise<void>;
}

export class NativeFitSharer implements FitSharer {
  async share(filename: string, base64: string): Promise<void> {
    const { uri } = await Filesystem.writeFile({ path: filename, data: base64, directory: Directory.Cache });
    await Share.share({ title: filename, url: uri });
  }
}

const todayLocal = (d: Date) =>
  `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`;

export class WatchSyncService {
  constructor(
    private readonly plugin: WorkoutKitPlugin = WorkoutKit,
    private readonly isIos: () => boolean = () => Capacitor.getPlatform() === 'ios',
    private readonly sharer: FitSharer = new NativeFitSharer(),
    private readonly now: () => Date = () => new Date(),
  ) {}

  supported(): boolean {
    return this.isIos();
  }

  async sync(items: ScheduledItem[]): Promise<WatchSyncOutcome> {
    if (!this.supported()) return { status: 'indisponivel_plataforma' };
    const auth = await this.plugin.requestAuthorization();
    if (auth.status !== 'authorized') return { status: 'autorizacao_negada' };
    const result = await this.plugin.syncWindow({
      items: items.map((i) => ({ plan: i.plano, date: i.data })),
      today: todayLocal(this.now()),
    });
    return { status: 'ok', result };
  }

  // Fallback do design doc (8.2): abre no app Workout; se o sistema recusar, exporta o .fit.
  async openOrExport(plan: PlannedWorkoutDTO): Promise<'aberto' | 'fit_exportado'> {
    try {
      await this.plugin.openInWorkoutApp({ plan });
      return 'aberto';
    } catch {
      const { filename, base64 } = await this.plugin.exportFit({ plan });
      await this.sharer.share(filename, base64);
      return 'fit_exportado';
    }
  }
}
