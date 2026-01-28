import Foundation

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
