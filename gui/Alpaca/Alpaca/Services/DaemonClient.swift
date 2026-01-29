import Foundation

// MARK: - Request Type

private struct Request: Encodable, Sendable {
    let command: String
    let args: [String: String]?

    init(command: String, args: [String: String]? = nil) {
        self.command = command
        self.args = args
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
        let response: StatusResponse
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

        switch data.state {
        case "idle":
            return .idle
        case "loading":
            let preset = data.preset ?? "unknown"
            return .loading(preset: preset)
        case "running":
            let preset = data.preset ?? "unknown"
            let endpoint = data.endpoint ?? "unknown"
            return .running(preset: preset, endpoint: endpoint)
        default:
            throw DaemonError.invalidResponse("Unknown state: \(data.state)")
        }
    }

    func loadModel(identifier: String) async throws {
        let response: GenericResponse = try await sendRequest(Request(command: "load", args: ["identifier": identifier]))

        if response.status != "ok" {
            throw DaemonError.fromCode(response.errorCode, message: response.error ?? "Failed to load model")
        }
    }

    func stopModel() async throws {
        let response: GenericResponse = try await sendRequest(Request(command: "unload"))

        if response.status != "ok" {
            throw DaemonError.protocolError(response.error ?? "Failed to stop model")
        }
    }

    func listPresets() async throws -> [Preset] {
        let response: PresetsResponse
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

        return data.presets.map { Preset(name: $0) }
    }

    func listModels() async throws -> [Model] {
        let response: ModelsResponse
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

        return data.models.map { modelData in
            Model(repo: modelData.repo, quant: modelData.quant, size: modelData.size)
        }
    }

    // MARK: - Private

    private func sendRequest<T: Decodable & Sendable>(_ request: Request) async throws -> T {
        let socketPath = self.socketPath
        return try await withCheckedThrowingContinuation { continuation in
            DispatchQueue.global().async {
                do {
                    let response: T = try Self.sendRequestSync(to: socketPath, request: request)
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

        // Set receive timeout to prevent infinite blocking
        var timeout = timeval(tv_sec: 10, tv_usec: 0)
        let timeoutResult = setsockopt(fd, SOL_SOCKET, SO_RCVTIMEO, &timeout, socklen_t(MemoryLayout<timeval>.size))
        guard timeoutResult == 0 else {
            close(fd)
            throw DaemonError.connectionFailed(underlying: NSError(domain: "DaemonClient", code: -1, userInfo: [NSLocalizedDescriptionKey: "Failed to set socket timeout"]))
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

    private static func sendRequestSync<T: Decodable & Sendable>(to socketPath: String, request: Request) throws -> T {
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
            if bytesRead < 0 {
                // Check if timeout occurred
                let error = errno
                if error == EAGAIN || error == EWOULDBLOCK {
                    throw DaemonError.connectionFailed(underlying: NSError(domain: "DaemonClient", code: Int(error), userInfo: [NSLocalizedDescriptionKey: "Socket read timeout"]))
                }
                throw DaemonError.connectionFailed(underlying: NSError(domain: "DaemonClient", code: Int(error), userInfo: [NSLocalizedDescriptionKey: "Socket read error"]))
            }
            if bytesRead == 0 {
                // Connection closed by daemon
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
            return try decoder.decode(T.self, from: responseData)
        } catch {
            throw DaemonError.invalidResponse("Failed to decode response: \(error.localizedDescription)")
        }
    }
}
