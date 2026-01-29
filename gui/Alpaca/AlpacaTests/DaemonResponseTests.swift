import Testing
import Foundation
@testable import Alpaca

// MARK: - StatusResponse Tests

@Suite("StatusResponse Decoding Tests")
struct StatusResponseTests {
    @Test("Decode status response with running state")
    func decodeRunningState() throws {
        let json = """
        {
            "status": "ok",
            "data": {
                "state": "running",
                "preset": "test-model",
                "endpoint": "http://localhost:8080"
            }
        }
        """.data(using: .utf8)!

        let decoder = JSONDecoder()
        let response = try decoder.decode(StatusResponse.self, from: json)

        #expect(response.status == "ok")
        #expect(response.data?.state == "running")
        #expect(response.data?.preset == "test-model")
        #expect(response.data?.endpoint == "http://localhost:8080")
        #expect(response.error == nil)
    }

    @Test("Decode status response with idle state")
    func decodeIdleState() throws {
        let json = """
        {
            "status": "ok",
            "data": {
                "state": "idle"
            }
        }
        """.data(using: .utf8)!

        let decoder = JSONDecoder()
        let response = try decoder.decode(StatusResponse.self, from: json)

        #expect(response.status == "ok")
        #expect(response.data?.state == "idle")
        #expect(response.data?.preset == nil)
        #expect(response.data?.endpoint == nil)
    }

    @Test("Decode status response with loading state")
    func decodeLoadingState() throws {
        let json = """
        {
            "status": "ok",
            "data": {
                "state": "loading",
                "preset": "test-model"
            }
        }
        """.data(using: .utf8)!

        let decoder = JSONDecoder()
        let response = try decoder.decode(StatusResponse.self, from: json)

        #expect(response.status == "ok")
        #expect(response.data?.state == "loading")
        #expect(response.data?.preset == "test-model")
        #expect(response.data?.endpoint == nil)
    }

    @Test("Decode error response")
    func decodeErrorResponse() throws {
        let json = """
        {
            "status": "error",
            "error": "daemon error message"
        }
        """.data(using: .utf8)!

        let decoder = JSONDecoder()
        let response = try decoder.decode(StatusResponse.self, from: json)

        #expect(response.status == "error")
        #expect(response.error == "daemon error message")
        #expect(response.data == nil)
    }
}

// MARK: - PresetsResponse Tests

@Suite("PresetsResponse Decoding Tests")
struct PresetsResponseTests {
    @Test("Decode presets response with multiple presets")
    func decodeMultiplePresets() throws {
        let json = """
        {
            "status": "ok",
            "data": {
                "presets": ["preset-1", "preset-2", "preset-3"]
            }
        }
        """.data(using: .utf8)!

        let decoder = JSONDecoder()
        let response = try decoder.decode(PresetsResponse.self, from: json)

        #expect(response.status == "ok")
        #expect(response.data?.presets.count == 3)
        #expect(response.data?.presets == ["preset-1", "preset-2", "preset-3"])
        #expect(response.error == nil)
    }

    @Test("Decode presets response with empty list")
    func decodeEmptyPresets() throws {
        let json = """
        {
            "status": "ok",
            "data": {
                "presets": []
            }
        }
        """.data(using: .utf8)!

        let decoder = JSONDecoder()
        let response = try decoder.decode(PresetsResponse.self, from: json)

        #expect(response.status == "ok")
        #expect(response.data?.presets.isEmpty == true)
    }

    @Test("Decode presets error response")
    func decodePresetsError() throws {
        let json = """
        {
            "status": "error",
            "error": "failed to list presets"
        }
        """.data(using: .utf8)!

        let decoder = JSONDecoder()
        let response = try decoder.decode(PresetsResponse.self, from: json)

        #expect(response.status == "error")
        #expect(response.error == "failed to list presets")
        #expect(response.data == nil)
    }
}

// MARK: - ModelsResponse Tests

@Suite("ModelsResponse Decoding Tests")
struct ModelsResponseTests {
    @Test("Decode models response with multiple models")
    func decodeMultipleModels() throws {
        let json = """
        {
            "status": "ok",
            "data": {
                "models": [
                    {
                        "repo": "TheBloke/CodeLlama-7B-GGUF",
                        "quant": "Q4_K_M",
                        "size": 4368438272
                    },
                    {
                        "repo": "TheBloke/Mistral-7B-GGUF",
                        "quant": "Q5_K_M",
                        "size": 5152665600
                    }
                ]
            }
        }
        """.data(using: .utf8)!

        let decoder = JSONDecoder()
        let response = try decoder.decode(ModelsResponse.self, from: json)

        #expect(response.status == "ok")
        #expect(response.data?.models.count == 2)

        let firstModel = response.data?.models[0]
        #expect(firstModel?.repo == "TheBloke/CodeLlama-7B-GGUF")
        #expect(firstModel?.quant == "Q4_K_M")
        #expect(firstModel?.size == 4368438272)

        let secondModel = response.data?.models[1]
        #expect(secondModel?.repo == "TheBloke/Mistral-7B-GGUF")
        #expect(secondModel?.quant == "Q5_K_M")
        #expect(secondModel?.size == 5152665600)
    }

    @Test("Decode models response with empty list")
    func decodeEmptyModels() throws {
        let json = """
        {
            "status": "ok",
            "data": {
                "models": []
            }
        }
        """.data(using: .utf8)!

        let decoder = JSONDecoder()
        let response = try decoder.decode(ModelsResponse.self, from: json)

        #expect(response.status == "ok")
        #expect(response.data?.models.isEmpty == true)
    }

    @Test("Decode models error response")
    func decodeModelsError() throws {
        let json = """
        {
            "status": "error",
            "error": "failed to list models"
        }
        """.data(using: .utf8)!

        let decoder = JSONDecoder()
        let response = try decoder.decode(ModelsResponse.self, from: json)

        #expect(response.status == "error")
        #expect(response.error == "failed to list models")
        #expect(response.data == nil)
    }
}

// MARK: - GenericResponse Tests

@Suite("GenericResponse Decoding Tests")
struct GenericResponseTests {
    @Test("Decode success response")
    func decodeSuccess() throws {
        let json = """
        {
            "status": "ok"
        }
        """.data(using: .utf8)!

        let decoder = JSONDecoder()
        let response = try decoder.decode(GenericResponse.self, from: json)

        #expect(response.status == "ok")
        #expect(response.error == nil)
        #expect(response.errorCode == nil)
    }

    @Test("Decode error response with error code")
    func decodeErrorWithCode() throws {
        let json = """
        {
            "status": "error",
            "error": "preset not found",
            "error_code": "preset_not_found"
        }
        """.data(using: .utf8)!

        let decoder = JSONDecoder()
        let response = try decoder.decode(GenericResponse.self, from: json)

        #expect(response.status == "error")
        #expect(response.error == "preset not found")
        #expect(response.errorCode == "preset_not_found")
    }

    @Test("Decode error response without error code")
    func decodeErrorWithoutCode() throws {
        let json = """
        {
            "status": "error",
            "error": "general error"
        }
        """.data(using: .utf8)!

        let decoder = JSONDecoder()
        let response = try decoder.decode(GenericResponse.self, from: json)

        #expect(response.status == "error")
        #expect(response.error == "general error")
        #expect(response.errorCode == nil)
    }

    @Test("Decode error with model_not_found code")
    func decodeModelNotFoundError() throws {
        let json = """
        {
            "status": "error",
            "error": "model file not found",
            "error_code": "model_not_found"
        }
        """.data(using: .utf8)!

        let decoder = JSONDecoder()
        let response = try decoder.decode(GenericResponse.self, from: json)

        #expect(response.status == "error")
        #expect(response.error == "model file not found")
        #expect(response.errorCode == "model_not_found")
    }

    @Test("Decode error with server_failed code")
    func decodeServerFailedError() throws {
        let json = """
        {
            "status": "error",
            "error": "llama-server failed to start",
            "error_code": "server_failed"
        }
        """.data(using: .utf8)!

        let decoder = JSONDecoder()
        let response = try decoder.decode(GenericResponse.self, from: json)

        #expect(response.status == "error")
        #expect(response.error == "llama-server failed to start")
        #expect(response.errorCode == "server_failed")
    }
}

// MARK: - DaemonErrorCode Tests

@Suite("DaemonErrorCode Tests")
struct DaemonErrorCodeTests {
    @Test("Parse preset_not_found code")
    func parsePresetNotFoundCode() {
        let code = DaemonErrorCode(rawValue: "preset_not_found")
        #expect(code == .presetNotFound)
    }

    @Test("Parse model_not_found code")
    func parseModelNotFoundCode() {
        let code = DaemonErrorCode(rawValue: "model_not_found")
        #expect(code == .modelNotFound)
    }

    @Test("Parse server_failed code")
    func parseServerFailedCode() {
        let code = DaemonErrorCode(rawValue: "server_failed")
        #expect(code == .serverFailed)
    }

    @Test("Parse invalid code returns nil")
    func parseInvalidCode() {
        let code = DaemonErrorCode(rawValue: "invalid_code")
        #expect(code == nil)
    }
}
