import Foundation

/// Protocol for communicating with the Alpaca daemon.
protocol DaemonClientProtocol: Sendable {
    func getStatus() async throws -> DaemonState
    func loadModel(identifier: String) async throws
    func stopModel() async throws
    func listPresets() async throws -> [Preset]
    func listModels() async throws -> [Model]
}

/// Errors that can occur during daemon communication.
enum DaemonError: Error, LocalizedError {
    case notRunning
    case connectionFailed(underlying: Error)
    case invalidResponse(String)
    case protocolError(String)

    var errorDescription: String? {
        switch self {
        case .notRunning:
            return "Daemon is not running"
        case .connectionFailed(let err):
            return "Connection failed: \(err.localizedDescription)"
        case .invalidResponse(let msg):
            return "Invalid response: \(msg)"
        case .protocolError(let msg):
            return msg
        }
    }
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
    private let mockModels: [Model] = [
        Model(repo: "TheBloke/CodeLlama-7B-GGUF", quant: "Q4_K_M", size: 4_370_000_000),
        Model(repo: "TheBloke/Mistral-7B-GGUF", quant: "Q5_K_M", size: 5_130_000_000),
    ]

    func getStatus() async throws -> DaemonState {
        return mockState
    }

    func loadModel(identifier: String) async throws {
        mockState = .loading(preset: identifier)
        try await Task.sleep(for: .seconds(2))
        mockState = .running(preset: identifier, endpoint: "localhost:8080")
    }

    func stopModel() async throws {
        mockState = .idle
    }

    func listPresets() async throws -> [Preset] {
        return mockPresets
    }

    func listModels() async throws -> [Model] {
        return mockModels
    }
}

// MARK: - Protocol Types

private struct Request: Encodable {
    let command: String
    let args: [String: String]?

    init(command: String, args: [String: String]? = nil) {
        self.command = command
        self.args = args
    }
}

private struct Response: Decodable {
    let status: String
    let data: [String: AnyCodable]?
    let error: String?
}

/// Type-erased Codable wrapper for heterogeneous JSON values.
private struct AnyCodable: Decodable {
    let value: Any

    init(from decoder: Decoder) throws {
        let container = try decoder.singleValueContainer()

        if container.decodeNil() {
            value = NSNull()
        } else if let bool = try? container.decode(Bool.self) {
            value = bool
        } else if let int = try? container.decode(Int.self) {
            value = int
        } else if let double = try? container.decode(Double.self) {
            value = double
        } else if let string = try? container.decode(String.self) {
            value = string
        } else if let array = try? container.decode([AnyCodable].self) {
            value = array.map { $0.value }
        } else if let dict = try? container.decode([String: AnyCodable].self) {
            value = dict.mapValues { $0.value }
        } else {
            throw DecodingError.dataCorruptedError(in: container, debugDescription: "Unsupported type")
        }
    }
}

// MARK: - DaemonClient

/// Real implementation of DaemonClient using Unix socket.
final class DaemonClient: DaemonClientProtocol, Sendable {
    private let socketPath: String

    init(socketPath: String = "~/.alpaca/alpaca.sock") {
        self.socketPath = (socketPath as NSString).expandingTildeInPath
    }

    func getStatus() async throws -> DaemonState {
        let response: Response
        do {
            response = try await sendRequest(Request(command: "status"))
        } catch DaemonError.notRunning {
            return .notRunning
        }

        guard response.status == "ok", let data = response.data else {
            if let error = response.error {
                throw DaemonError.protocolError(error)
            }
            throw DaemonError.invalidResponse("Missing data in status response")
        }

        guard let stateString = data["state"]?.value as? String else {
            throw DaemonError.invalidResponse("Missing state in status response")
        }

        switch stateString {
        case "idle":
            return .idle
        case "loading":
            let preset = data["preset"]?.value as? String ?? "unknown"
            return .loading(preset: preset)
        case "running":
            let preset = data["preset"]?.value as? String ?? "unknown"
            let endpoint = data["endpoint"]?.value as? String ?? "unknown"
            return .running(preset: preset, endpoint: endpoint)
        default:
            throw DaemonError.invalidResponse("Unknown state: \(stateString)")
        }
    }

    func loadModel(identifier: String) async throws {
        let response = try await sendRequest(Request(command: "load", args: ["identifier": identifier]))

        if response.status != "ok" {
            throw DaemonError.protocolError(response.error ?? "Failed to load model")
        }
    }

    func stopModel() async throws {
        let response = try await sendRequest(Request(command: "kill"))

        if response.status != "ok" {
            throw DaemonError.protocolError(response.error ?? "Failed to stop model")
        }
    }

    func listPresets() async throws -> [Preset] {
        let response: Response
        do {
            response = try await sendRequest(Request(command: "list_presets"))
        } catch DaemonError.notRunning {
            return []
        }

        guard response.status == "ok", let data = response.data else {
            if let error = response.error {
                throw DaemonError.protocolError(error)
            }
            throw DaemonError.invalidResponse("Missing data in list_presets response")
        }

        guard let presetNames = data["presets"]?.value as? [Any] else {
            throw DaemonError.invalidResponse("Missing presets in response")
        }

        return presetNames.compactMap { name in
            guard let nameString = name as? String else { return nil }
            return Preset(name: nameString)
        }
    }

    func listModels() async throws -> [Model] {
        let response: Response
        do {
            response = try await sendRequest(Request(command: "list_models"))
        } catch DaemonError.notRunning {
            return []
        }

        guard response.status == "ok", let data = response.data else {
            if let error = response.error {
                throw DaemonError.protocolError(error)
            }
            throw DaemonError.invalidResponse("Missing data in list_models response")
        }

        guard let modelList = data["models"]?.value as? [Any] else {
            // Return empty array if no models key (might be null)
            return []
        }

        return modelList.compactMap { item in
            guard let dict = item as? [String: Any],
                  let repo = dict["repo"] as? String,
                  let quant = dict["quant"] as? String else {
                return nil
            }
            let size = (dict["size"] as? Int64) ?? (dict["size"] as? Int).map(Int64.init) ?? 0
            return Model(repo: repo, quant: quant, size: size)
        }
    }

    // MARK: - Private

    private func sendRequest(_ request: Request) async throws -> Response {
        try await withCheckedThrowingContinuation { continuation in
            DispatchQueue.global().async { [socketPath] in
                do {
                    let response = try Self.sendRequestSync(to: socketPath, request: request)
                    continuation.resume(returning: response)
                } catch {
                    continuation.resume(throwing: error)
                }
            }
        }
    }

    private static func connectToSocket(at path: String) throws -> Int32 {
        let fd = socket(AF_UNIX, SOCK_STREAM, 0)
        guard fd >= 0 else {
            throw DaemonError.notRunning
        }

        var addr = sockaddr_un()
        addr.sun_family = sa_family_t(AF_UNIX)

        let pathBytes = path.utf8CString
        guard pathBytes.count <= MemoryLayout.size(ofValue: addr.sun_path) else {
            close(fd)
            throw DaemonError.invalidResponse("Socket path too long")
        }

        withUnsafeMutablePointer(to: &addr.sun_path) { ptr in
            ptr.withMemoryRebound(to: CChar.self, capacity: pathBytes.count) { dest in
                for (i, byte) in pathBytes.enumerated() {
                    dest[i] = byte
                }
            }
        }

        let result = withUnsafePointer(to: &addr) { ptr in
            ptr.withMemoryRebound(to: sockaddr.self, capacity: 1) { sockaddrPtr in
                Darwin.connect(fd, sockaddrPtr, socklen_t(MemoryLayout<sockaddr_un>.size))
            }
        }

        guard result == 0 else {
            close(fd)
            throw DaemonError.notRunning
        }

        return fd
    }

    private static func sendRequestSync(to socketPath: String, request: Request) throws -> Response {
        let fd = try connectToSocket(at: socketPath)
        defer { close(fd) }

        // Send request (JSON + newline)
        let encoder = JSONEncoder()
        var requestData = try encoder.encode(request)
        requestData.append(0x0A) // newline

        let writeResult = requestData.withUnsafeBytes { ptr in
            Darwin.write(fd, ptr.baseAddress, ptr.count)
        }

        guard writeResult == requestData.count else {
            throw DaemonError.connectionFailed(underlying: NSError(domain: "DaemonClient", code: -1, userInfo: [NSLocalizedDescriptionKey: "Failed to write request"]))
        }

        // Read response
        var responseData = Data()
        var buffer = [UInt8](repeating: 0, count: 4096)

        while true {
            let bytesRead = Darwin.read(fd, &buffer, buffer.count)
            if bytesRead <= 0 {
                break
            }
            responseData.append(contentsOf: buffer.prefix(bytesRead))

            // Check for newline (end of response)
            if buffer.prefix(bytesRead).contains(0x0A) {
                break
            }
        }

        guard !responseData.isEmpty else {
            throw DaemonError.invalidResponse("Empty response from daemon")
        }

        // Decode response
        let decoder = JSONDecoder()
        do {
            return try decoder.decode(Response.self, from: responseData)
        } catch {
            throw DaemonError.invalidResponse("Failed to decode response: \(error.localizedDescription)")
        }
    }
}
