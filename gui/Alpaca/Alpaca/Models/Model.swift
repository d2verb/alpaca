import Foundation

/// Represents a downloaded model from HuggingFace.
struct Model: Identifiable, Equatable {
    let id: String
    let repo: String
    let quant: String
    let size: Int64

    /// Returns the identifier in HuggingFace format (repo:quant).
    var identifier: String {
        "\(repo):\(quant)"
    }

    /// Returns a formatted display name.
    var displayName: String {
        // Extract the model name from repo (e.g., "TheBloke/CodeLlama-7B-GGUF" -> "CodeLlama-7B")
        let repoName = repo.split(separator: "/").last.map(String.init) ?? repo
        let baseName = repoName.replacingOccurrences(of: "-GGUF", with: "")
        return "\(baseName) (\(quant))"
    }

    /// Returns human-readable size string.
    var sizeString: String {
        let formatter = ByteCountFormatter()
        formatter.allowedUnits = [.useGB, .useMB]
        formatter.countStyle = .file
        return formatter.string(fromByteCount: size)
    }

    init(repo: String, quant: String, size: Int64) {
        self.id = "\(repo):\(quant)"
        self.repo = repo
        self.quant = quant
        self.size = size
    }
}
