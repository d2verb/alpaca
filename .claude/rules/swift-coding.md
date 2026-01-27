---
paths: "gui/**/*"
---

# Swift Coding Rules

## Concurrency (Swift 6)

Use Actor for shared state, MainActor for UI:
```swift
// Shared state
actor DaemonClient {
    private var connection: Connection?
    func connect() async throws { ... }
}

// UI state
@MainActor
final class ViewModel: ObservableObject {
    @Published private(set) var status: Status = .idle
}
```

## Error Handling

Define descriptive error types:
```swift
enum DaemonError: Error, LocalizedError {
    case notRunning
    case connectionFailed(underlying: Error)
    case invalidResponse(String)

    var errorDescription: String? {
        switch self {
        case .notRunning: "Daemon is not running"
        case .connectionFailed(let err): "Connection failed: \(err.localizedDescription)"
        case .invalidResponse(let msg): "Invalid response: \(msg)"
        }
    }
}

func connect() async throws {
    guard isDaemonRunning else { throw DaemonError.notRunning }
    // ...
}
```

## Optionals

Prefer guard-let over force unwrap:
```swift
// WRONG
let name = preset.name!

// CORRECT
guard let name = preset.name else { throw PresetError.missingName }
let name = preset.name ?? "Untitled"
```

## Naming
```swift
// Methods as grammatical phrases
func makeConnection(to socket: URL) -> Connection
func remove(_ entry: Entry)

// Booleans as assertions
var isEmpty: Bool
var canExecute: Bool
```

## SwiftUI State
```swift
@StateObject private var vm = ViewModel()  // View owns it
@ObservedObject var vm: ViewModel          // Passed from parent
@State private var isEnabled = false       // Simple value types
```

## SwiftUI Views

Keep body under 50 lines. Extract subviews:
```swift
struct RaceRow: View {
    let race: Race
    var body: some View {
        HStack {
            Text(race.name)
            Spacer()
            StatusBadge(status: race.status)
        }
    }
}
```