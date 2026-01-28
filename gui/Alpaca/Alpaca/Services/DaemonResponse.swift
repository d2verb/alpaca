import Foundation

/// Response structure for status command.
struct StatusResponse: Decodable, Sendable {
    let status: String
    let data: StatusData?
    let error: String?

    struct StatusData: Decodable, Sendable {
        let state: String
        let preset: String?
        let endpoint: String?
    }
}

/// Response structure for list_presets command.
struct PresetsResponse: Decodable, Sendable {
    let status: String
    let data: PresetsData?
    let error: String?

    struct PresetsData: Decodable, Sendable {
        let presets: [String]
    }
}

/// Response structure for list_models command.
struct ModelsResponse: Decodable, Sendable {
    let status: String
    let data: ModelsData?
    let error: String?

    struct ModelsData: Decodable, Sendable {
        let models: [ModelData]

        struct ModelData: Decodable, Sendable {
            let repo: String
            let quant: String
            let size: Int64
        }
    }
}

/// Generic response structure for commands without specific data.
struct GenericResponse: Decodable, Sendable {
    let status: String
    let error: String?
}
