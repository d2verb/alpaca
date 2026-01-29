import Foundation

/// Represents a model preset configuration.
struct Preset: Identifiable, Equatable {
    let id: String
    let name: String

    /// Returns the identifier in preset format (p:name).
    var identifier: String {
        "p:\(name)"
    }

    init(name: String) {
        self.id = "p:\(name)"
        self.name = name
    }
}
