import Foundation
@testable import Alpaca

/// Configurable mock client for testing AppViewModel.
actor TestDaemonClient: DaemonClientProtocol {
    var statusToReturn: DaemonState = .idle
    var presetsToReturn: [Preset] = []
    var modelsToReturn: [Model] = []
    var errorToThrow: Error?
    var loadModelCalled = false
    var stopModelCalled = false
    var lastLoadedIdentifier: String?

    func configure(status: DaemonState = .idle, presets: [Preset] = [], models: [Model] = [], error: Error? = nil) {
        statusToReturn = status
        presetsToReturn = presets
        modelsToReturn = models
        errorToThrow = error
    }

    func getStatus() async throws -> DaemonState {
        if let error = errorToThrow {
            throw error
        }
        return statusToReturn
    }

    func loadModel(identifier: String) async throws {
        loadModelCalled = true
        lastLoadedIdentifier = identifier
        if let error = errorToThrow {
            throw error
        }
    }

    func stopModel() async throws {
        stopModelCalled = true
        if let error = errorToThrow {
            throw error
        }
    }

    func listPresets() async throws -> [Preset] {
        if let error = errorToThrow {
            throw error
        }
        return presetsToReturn
    }

    func listModels() async throws -> [Model] {
        if let error = errorToThrow {
            throw error
        }
        return modelsToReturn
    }
}
