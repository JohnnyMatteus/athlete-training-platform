import Capacitor
import Foundation
import WorkoutKitCore
#if canImport(WorkoutKit)
import WorkoutKit
#endif

/// Ponte Capacitor: decodifica o DTO, traduz com o núcleo e delega ao WorkoutKit.
/// Não compilado no ambiente de desenvolvimento (sem Xcode): ver README, "O que não foi verificado".
@objc(WorkoutKitPlugin)
public class WorkoutKitPlugin: CAPPlugin, CAPBridgedPlugin {
    public let identifier = "WorkoutKitPlugin"
    public let jsName = "WorkoutKit"
    public let pluginMethods: [CAPPluginMethod] = [
        CAPPluginMethod(name: "requestAuthorization", returnType: CAPPluginReturnPromise),
        CAPPluginMethod(name: "schedule", returnType: CAPPluginReturnPromise),
        CAPPluginMethod(name: "listScheduled", returnType: CAPPluginReturnPromise),
        CAPPluginMethod(name: "remove", returnType: CAPPluginReturnPromise),
        CAPPluginMethod(name: "removeAll", returnType: CAPPluginReturnPromise),
        CAPPluginMethod(name: "openInWorkoutApp", returnType: CAPPluginReturnPromise),
        CAPPluginMethod(name: "syncWindow", returnType: CAPPluginReturnPromise),
        CAPPluginMethod(name: "exportFit", returnType: CAPPluginReturnPromise),
    ]

    // MARK: DTO

    private func blueprint(_ object: JSObject?) throws -> WorkoutBlueprint {
        guard let object else { throw TranslationError.emptyWorkout }
        let data = try JSONSerialization.data(withJSONObject: object)
        return try WorkoutTranslator.translate(PlannedWorkout.decode(from: data))
    }

    @objc func exportFit(_ call: CAPPluginCall) {
        do {
            let bp = try blueprint(call.getObject("plan"))
            // FIT conta segundos desde 1989-12-31T00:00:00Z.
            let created = UInt32(max(0, Date().timeIntervalSince1970 - 631_065_600))
            let bytes = FitWorkoutEncoder.encode(bp, createdAt: created)
            call.resolve(["filename": "\(bp.id).fit", "base64": Data(bytes).base64EncodedString()])
        } catch {
            call.reject("Treino inválido: \(error)")
        }
    }

    /// `WorkoutPlan.openInWorkoutApp()` existe só no watchOS (no iOS o compilador o marca como indisponível).
    /// No iOS 17 o caminho da Apple é a prévia do WorkoutKit (modificador SwiftUI `workoutPreview`), que não foi
    /// implementada: sem Xcode nem aparelho não há como confirmar a API. Rejeitar explicitamente, nunca resolver sem
    /// ter feito nada: o app captura a rejeição e cai no `exportFit` (watchSync.openOrExport).
    @objc func openInWorkoutApp(_ call: CAPPluginCall) {
        call.reject("openInWorkoutApp não está disponível no iOS; use exportFit.", "UNIMPLEMENTED")
    }

    #if canImport(WorkoutKit)
    private func components(_ local: String) -> DateComponents? {
        // YYYY-MM-DDTHH:mm
        let p = local.split(whereSeparator: { "-T:".contains($0) }).compactMap { Int($0) }
        guard p.count >= 5 else { return nil }
        return DateComponents(year: p[0], month: p[1], day: p[2], hour: p[3], minute: p[4])
    }

    private func local(_ c: DateComponents) -> String {
        String(format: "%04d-%02d-%02dT%02d:%02d", c.year ?? 0, c.month ?? 0, c.day ?? 0, c.hour ?? 0, c.minute ?? 0)
    }

    @objc func requestAuthorization(_ call: CAPPluginCall) {
        guard #available(iOS 17.0, *) else { return call.unavailable("Requer iOS 17.") }
        Task {
            let state = await WorkoutScheduler.shared.requestAuthorization()
            switch state {
            case .authorized: call.resolve(["status": "authorized"])
            case .notDetermined: call.resolve(["status": "notDetermined"])
            default: call.resolve(["status": "denied"])
            }
        }
    }

    @objc func schedule(_ call: CAPPluginCall) {
        guard #available(iOS 17.0, *) else { return call.unavailable("Requer iOS 17.") }
        guard let date = call.getString("date").flatMap(components) else { return call.reject("date inválida") }
        Task {
            do {
                let plan = try WorkoutKitMapper.plan(from: try blueprint(call.getObject("plan")))
                try await WorkoutScheduler.shared.schedule(plan, at: date)
                call.resolve(["id": plan.id.uuidString.lowercased()])
            } catch { call.reject("Falha ao agendar: \(error)") }
        }
    }

    @objc func listScheduled(_ call: CAPPluginCall) {
        guard #available(iOS 17.0, *) else { return call.unavailable("Requer iOS 17.") }
        Task {
            let items = await WorkoutScheduler.shared.scheduledWorkouts.map {
                ["id": $0.plan.id.uuidString.lowercased(), "date": local($0.date), "name": "\($0.plan.workout)"]
            }
            call.resolve(["items": items])
        }
    }

    @objc func remove(_ call: CAPPluginCall) {
        guard #available(iOS 17.0, *) else { return call.unavailable("Requer iOS 17.") }
        guard let id = call.getString("id")?.lowercased() else { return call.reject("id ausente") }
        Task {
            for s in await WorkoutScheduler.shared.scheduledWorkouts where s.plan.id.uuidString.lowercased() == id {
                await WorkoutScheduler.shared.remove(s.plan, at: s.date)
            }
            call.resolve()
        }
    }

    @objc func removeAll(_ call: CAPPluginCall) {
        guard #available(iOS 17.0, *) else { return call.unavailable("Requer iOS 17.") }
        Task {
            await WorkoutScheduler.shared.removeAllWorkouts()
            call.resolve()
        }
    }

    /// Reconcilia o agendado com a fila do backend (janela deslizante, ADR 0034).
    @objc func syncWindow(_ call: CAPPluginCall) {
        guard #available(iOS 17.0, *) else { return call.unavailable("Requer iOS 17.") }
        guard let today = call.getString("today").flatMap({ LocalDate($0) }),
              let raw = call.getArray("items", JSObject.self) else { return call.reject("today ou items inválidos") }
        let windowDays = call.getInt("windowDays") ?? SlidingWindowPlanner.defaultWindowDays

        Task {
            var plans: [String: (JSObject, String)] = [:]
            for item in raw {
                guard let plan = item["plan"] as? JSObject, let date = item["date"] as? String,
                      let id = plan["id"] as? String else { return call.reject("item inválido") }
                plans[id.lowercased()] = (plan, date)
            }
            let scheduled = await WorkoutScheduler.shared.scheduledWorkouts
            let max = await WorkoutScheduler.maxAllowedScheduledWorkoutCount
            let decision = SlidingWindowPlanner.plan(
                desired: plans.map { WindowEntry(id: $0.key, date: $0.value.1) },
                scheduled: scheduled.map { WindowEntry(id: $0.plan.id.uuidString.lowercased(), date: local($0.date)) },
                today: today, windowDays: windowDays,
                maxCount: max > 0 ? max : SlidingWindowPlanner.defaultMaxCount
            )

            for s in scheduled where decision.toRemove.contains(s.plan.id.uuidString.lowercased()) {
                await WorkoutScheduler.shared.remove(s.plan, at: s.date)
            }
            var added: [String] = [], failed: [[String: String]] = []
            for id in decision.toAdd {
                guard let (plan, date) = plans[id], let dc = components(date) else { continue }
                do {
                    let wk = try WorkoutKitMapper.plan(from: try blueprint(plan))
                    try await WorkoutScheduler.shared.schedule(wk, at: dc)
                    added.append(id)
                } catch { failed.append(["id": id, "reason": "\(error)"]) }
            }
            call.resolve(["added": added, "removed": decision.toRemove, "deferred": decision.deferred, "failed": failed])
        }
    }
    #else
    @objc func requestAuthorization(_ call: CAPPluginCall) { call.unavailable("WorkoutKit indisponível nesta plataforma.") }
    @objc func schedule(_ call: CAPPluginCall) { call.unavailable("WorkoutKit indisponível nesta plataforma.") }
    @objc func listScheduled(_ call: CAPPluginCall) { call.unavailable("WorkoutKit indisponível nesta plataforma.") }
    @objc func remove(_ call: CAPPluginCall) { call.unavailable("WorkoutKit indisponível nesta plataforma.") }
    @objc func removeAll(_ call: CAPPluginCall) { call.unavailable("WorkoutKit indisponível nesta plataforma.") }
    @objc func syncWindow(_ call: CAPPluginCall) { call.unavailable("WorkoutKit indisponível nesta plataforma.") }
    #endif
}
