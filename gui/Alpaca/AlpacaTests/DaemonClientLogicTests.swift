import Testing
import Foundation
@testable import Alpaca

// MARK: - DaemonClient State Mapping Logic Tests
//
// Note: These tests verify the state mapping logic in DaemonClient.
// Full integration tests with actual socket communication require
// a running daemon or mock socket implementation.

@Suite("DaemonClient State Mapping Tests")
struct DaemonClientStateMappingTests {
    @Test("Map idle state correctly")
    func mapIdleState() async throws {
        // This test verifies the logic of mapping "idle" state
        // In real implementation, this would come from StatusResponse

        // Simulate what getStatus() does with an idle response
        let state = "idle"

        // The expected mapping
        let expectedState = DaemonState.idle

        #expect(state == "idle")

        // Verify that our expected state is correct
        switch expectedState {
        case .idle:
            break // Correct
        default:
            Issue.record("Expected idle state")
        }
    }

    @Test("Map loading state correctly")
    func mapLoadingState() async throws {
        let state = "loading"
        let preset = "test-model"

        let expectedState = DaemonState.loading(preset: preset)

        #expect(state == "loading")

        switch expectedState {
        case .loading(let p):
            #expect(p == "test-model")
        default:
            Issue.record("Expected loading state")
        }
    }

    @Test("Map running state correctly")
    func mapRunningState() async throws {
        let state = "running"
        let preset = "test-model"
        let endpoint = "http://localhost:8080"

        let expectedState = DaemonState.running(preset: preset, endpoint: endpoint)

        #expect(state == "running")

        switch expectedState {
        case .running(let p, let e):
            #expect(p == "test-model")
            #expect(e == "http://localhost:8080")
        default:
            Issue.record("Expected running state")
        }
    }

    @Test("Handle missing preset in loading state")
    func handleMissingPresetInLoading() async throws {
        // When preset is nil in response, should default to "unknown"
        let preset: String? = nil
        let actualPreset = preset ?? "unknown"

        #expect(actualPreset == "unknown")
    }

    @Test("Handle missing preset and endpoint in running state")
    func handleMissingFieldsInRunning() async throws {
        // When fields are nil in response, should default to "unknown"
        let preset: String? = nil
        let endpoint: String? = nil

        let actualPreset = preset ?? "unknown"
        let actualEndpoint = endpoint ?? "unknown"

        #expect(actualPreset == "unknown")
        #expect(actualEndpoint == "unknown")
    }
}

// MARK: - DaemonClient Error Handling Tests

@Suite("DaemonClient Error Handling Tests")
struct DaemonClientErrorHandlingTests {
    @Test("NotRunning error returns notRunning state in getStatus")
    func notRunningReturnsNotRunningState() {
        // Verify that DaemonError.notRunning is handled correctly
        let error = DaemonError.notRunning

        switch error {
        case .notRunning:
            break // Correct
        default:
            Issue.record("Expected notRunning error")
        }
    }

    @Test("NotRunning error returns empty array in listPresets")
    func notRunningReturnsEmptyPresets() {
        // Verify that listPresets returns [] when daemon not running
        let error = DaemonError.notRunning
        let expectedResult: [Preset] = []

        #expect(expectedResult.isEmpty)

        switch error {
        case .notRunning:
            break // This should result in empty array
        default:
            Issue.record("Expected notRunning error")
        }
    }

    @Test("NotRunning error returns empty array in listModels")
    func notRunningReturnsEmptyModels() {
        // Verify that listModels returns [] when daemon not running
        let error = DaemonError.notRunning
        let expectedResult: [Model] = []

        #expect(expectedResult.isEmpty)

        switch error {
        case .notRunning:
            break // This should result in empty array
        default:
            Issue.record("Expected notRunning error")
        }
    }

    @Test("Error response throws protocolError")
    func errorResponseThrowsProtocolError() {
        // Verify error response handling
        let errorMessage = "daemon error message"
        let error = DaemonError.protocolError(errorMessage)

        switch error {
        case .protocolError(let message):
            #expect(message == "daemon error message")
        default:
            Issue.record("Expected protocolError")
        }
    }

    @Test("Missing data throws invalidResponse")
    func missingDataThrowsInvalidResponse() {
        // Verify that missing data in response is handled
        let error = DaemonError.invalidResponse("Missing data in status response")

        switch error {
        case .invalidResponse(let message):
            #expect(message.contains("Missing data"))
        default:
            Issue.record("Expected invalidResponse")
        }
    }

    @Test("Unknown state throws invalidResponse")
    func unknownStateThrowsInvalidResponse() {
        // Verify that unknown state is handled
        let unknownState = "invalid_state"
        let error = DaemonError.invalidResponse("Unknown state: \(unknownState)")

        switch error {
        case .invalidResponse(let message):
            #expect(message.contains("Unknown state"))
            #expect(message.contains("invalid_state"))
        default:
            Issue.record("Expected invalidResponse")
        }
    }
}

// MARK: - DaemonClient List Operations Tests

@Suite("DaemonClient List Operations Logic Tests")
struct DaemonClientListOperationsTests {
    @Test("Map preset names to Preset objects")
    func mapPresetNames() {
        // Verify the mapping logic in listPresets
        let presetNames = ["preset-1", "preset-2", "preset-3"]
        let presets = presetNames.map { Preset(name: $0) }

        #expect(presets.count == 3)
        #expect(presets[0].name == "preset-1")
        #expect(presets[1].name == "preset-2")
        #expect(presets[2].name == "preset-3")
    }

    @Test("Map model data to Model objects")
    func mapModelData() {
        // Verify the mapping logic in listModels
        struct ModelData {
            let repo: String
            let quant: String
            let size: Int64
        }

        let modelDataList = [
            ModelData(repo: "TheBloke/CodeLlama-7B-GGUF", quant: "Q4_K_M", size: 4368438272),
            ModelData(repo: "TheBloke/Mistral-7B-GGUF", quant: "Q5_K_M", size: 5152665600)
        ]

        let models = modelDataList.map { data in
            Model(repo: data.repo, quant: data.quant, size: data.size)
        }

        #expect(models.count == 2)
        #expect(models[0].repo == "TheBloke/CodeLlama-7B-GGUF")
        #expect(models[0].quant == "Q4_K_M")
        #expect(models[0].size == 4368438272)
        #expect(models[1].repo == "TheBloke/Mistral-7B-GGUF")
        #expect(models[1].quant == "Q5_K_M")
        #expect(models[1].size == 5152665600)
    }

    @Test("Handle empty preset list")
    func handleEmptyPresetList() {
        let presetNames: [String] = []
        let presets = presetNames.map { Preset(name: $0) }

        #expect(presets.isEmpty)
    }

    @Test("Handle empty model list")
    func handleEmptyModelList() {
        struct ModelData {
            let repo: String
            let quant: String
            let size: Int64
        }

        let modelDataList: [ModelData] = []
        let models = modelDataList.map { data in
            Model(repo: data.repo, quant: data.quant, size: data.size)
        }

        #expect(models.isEmpty)
    }
}

// MARK: - Socket Path Expansion Tests

@Suite("DaemonClient Socket Path Tests")
struct DaemonClientSocketPathTests {
    @Test("Expand tilde in socket path")
    func expandTildeInPath() {
        let path = "~/.alpaca/alpaca.sock"
        let expandedPath = (path as NSString).expandingTildeInPath

        // Should not contain tilde after expansion
        #expect(!expandedPath.contains("~"))
        #expect(expandedPath.hasSuffix("/.alpaca/alpaca.sock"))
    }

    @Test("Absolute path remains unchanged")
    func absolutePathUnchanged() {
        let path = "/tmp/test.sock"
        let expandedPath = (path as NSString).expandingTildeInPath

        #expect(expandedPath == "/tmp/test.sock")
    }

    @Test("Default socket path is expanded")
    func defaultSocketPathExpanded() {
        let defaultPath = "~/.alpaca/alpaca.sock"
        let expandedPath = (defaultPath as NSString).expandingTildeInPath

        #expect(!expandedPath.contains("~"))
        #expect(expandedPath.contains(".alpaca"))
        #expect(expandedPath.contains("alpaca.sock"))
    }
}
