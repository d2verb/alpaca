import Foundation
import SwiftUI

/// Main view model managing application state and daemon communication.
@MainActor
@Observable
final class AppViewModel {
    private(set) var state: DaemonState = .idle
    private(set) var presets: [Preset] = []
    private(set) var models: [Model] = []
    private(set) var errorMessage: String?

    private let client: DaemonClientProtocol

    init(client: DaemonClientProtocol) {
        self.client = client
    }

    /// Initialize and load initial data.
    func initialize() async {
        await refreshStatus()
        await loadPresets()
        await loadModels()
    }

    /// Refresh the current daemon status.
    func refreshStatus() async {
        do {
            state = try await client.getStatus()
            errorMessage = nil
        } catch DaemonError.notRunning {
            state = .notRunning
            errorMessage = nil
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    /// Load the list of available presets.
    func loadPresets() async {
        do {
            presets = try await client.listPresets()
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    /// Load the list of downloaded models.
    func loadModels() async {
        do {
            models = try await client.listModels()
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    /// Load a model with the specified identifier (preset name or repo:quant).
    func loadModel(identifier: String) async {
        state = .loading(preset: identifier)
        do {
            try await client.loadModel(identifier: identifier)
            await refreshStatus()
        } catch {
            errorMessage = error.localizedDescription
            await refreshStatus()
        }
    }

    /// Stop the currently running model.
    func stopModel() async {
        do {
            try await client.stopModel()
            await refreshStatus()
        } catch {
            errorMessage = error.localizedDescription
        }
    }
}
