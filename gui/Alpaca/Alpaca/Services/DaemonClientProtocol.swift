import Foundation

/// Protocol for communicating with the Alpaca daemon.
protocol DaemonClientProtocol: Sendable {
    func getStatus() async throws -> DaemonState
    func loadModel(identifier: String) async throws
    func stopModel() async throws
    func listPresets() async throws -> [Preset]
    func listModels() async throws -> [Model]
}
