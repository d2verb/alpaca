import Foundation

/// Represents a model preset configuration.
struct Preset: Identifiable, Equatable {
    let id: String
    let name: String

    init(name: String) {
        self.id = name
        self.name = name
    }
}
