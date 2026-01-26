import Foundation
import SwiftUI

/// Main view model managing application state and daemon communication.
@MainActor
@Observable
final class AppViewModel {
    private(set) var state: DaemonState = .idle
    private(set) var presets: [Preset] = []
    private(set) var errorMessage: String?

    private let client: DaemonClientProtocol

    init(client: DaemonClientProtocol = MockDaemonClient()) {
        self.client = client
    }

    /// Initialize and load initial data.
    func initialize() async {
        await refreshStatus()
        await loadPresets()
    }

    /// Refresh the current daemon status.
    func refreshStatus() async {
        do {
            state = try await client.getStatus()
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

    /// Load a model with the specified preset.
    func loadModel(preset: String) async {
        state = .loading(preset: preset)
        do {
            try await client.loadModel(preset: preset)
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

    /// Copy the daemon start command to clipboard.
    func copyStartCommand() {
        let pasteboard = NSPasteboard.general
        pasteboard.clearContents()
        pasteboard.setString("alpaca start", forType: .string)
    }
}
