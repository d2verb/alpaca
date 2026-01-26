import Foundation

/// Represents the current state of the Alpaca daemon.
enum DaemonState: Equatable {
    /// Daemon is not running
    case notRunning
    /// Daemon is running but no model is loaded
    case idle
    /// A model is being loaded
    case loading(preset: String)
    /// A model is running and ready
    case running(preset: String, endpoint: String)

    var statusText: String {
        switch self {
        case .notRunning:
            return "Daemon not running"
        case .idle:
            return "Idle"
        case .loading(let preset):
            return "Loading \(preset)..."
        case .running:
            return "Running"
        }
    }

    var isRunning: Bool {
        if case .running = self {
            return true
        }
        return false
    }

    var isLoading: Bool {
        if case .loading = self {
            return true
        }
        return false
    }

    var currentPreset: String? {
        switch self {
        case .running(let preset, _):
            return preset
        case .loading(let preset):
            return preset
        default:
            return nil
        }
    }
}
