import Foundation

/// Protocol for communicating with the Alpaca daemon.
protocol DaemonClientProtocol: Sendable {
    func getStatus() async throws -> DaemonState
    func loadModel(preset: String) async throws
    func stopModel() async throws
    func listPresets() async throws -> [Preset]
}

/// Mock implementation of DaemonClient for UI development.
actor MockDaemonClient: DaemonClientProtocol {
    private var mockState: DaemonState = .idle
    private let mockPresets: [Preset] = [
        Preset(name: "codellama-7b-q4"),
        Preset(name: "mistral-7b-q4"),
        Preset(name: "deepseek-coder-6.7b"),
        Preset(name: "llama3-8b-q4"),
    ]

    func getStatus() async throws -> DaemonState {
        return mockState
    }

    func loadModel(preset: String) async throws {
        mockState = .loading(preset: preset)
        try await Task.sleep(for: .seconds(2))
        mockState = .running(preset: preset, endpoint: "localhost:8080")
    }

    func stopModel() async throws {
        mockState = .idle
    }

    func listPresets() async throws -> [Preset] {
        return mockPresets
    }
}

/// Real implementation of DaemonClient using Unix socket.
/// This is a stub that will be implemented when daemon communication is added.
final class DaemonClient: DaemonClientProtocol, Sendable {
    private let socketPath: String

    init(socketPath: String = "~/.alpaca/alpaca.sock") {
        self.socketPath = (socketPath as NSString).expandingTildeInPath
    }

    func getStatus() async throws -> DaemonState {
        // TODO: Implement Unix socket communication
        return .notRunning
    }

    func loadModel(preset: String) async throws {
        // TODO: Implement Unix socket communication
    }

    func stopModel() async throws {
        // TODO: Implement Unix socket communication
    }

    func listPresets() async throws -> [Preset] {
        // TODO: Implement Unix socket communication
        return []
    }
}
