import Foundation

/// Errors that can occur during daemon communication.
enum DaemonError: Error, LocalizedError {
    case notRunning
    case connectionFailed(underlying: Error)
    case invalidResponse(String)
    case protocolError(String)
    case presetNotFound(String)
    case modelNotFound(String)
    case serverFailed(String)

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
        case .presetNotFound(let msg):
            return "Preset not found: \(msg)"
        case .modelNotFound(let msg):
            return "Model not found: \(msg)"
        case .serverFailed(let msg):
            return "Server failed: \(msg)"
        }
    }

    /// Creates a DaemonError from an error code and message.
    static func fromCode(_ code: String?, message: String) -> DaemonError {
        guard let code = code, let errorCode = DaemonErrorCode(rawValue: code) else {
            return .protocolError(message)
        }
        switch errorCode {
        case .presetNotFound:
            return .presetNotFound(message)
        case .modelNotFound:
            return .modelNotFound(message)
        case .serverFailed:
            return .serverFailed(message)
        }
    }
}
