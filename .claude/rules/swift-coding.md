---
paths: "gui/**/*"
---

# Swift Coding Rules

## Concurrency (Swift 6)

Use async/await and Actor for concurrency:

```swift
// CORRECT: Actor for shared state
actor DaemonClient {
    private var connection: Connection?

    func connect() async throws {
        // ...
    }
}

// CORRECT: MainActor for UI state
@MainActor
class MenuBarViewModel: ObservableObject {
    @Published var status: DaemonStatus = .disconnected
}
```

## Naming

Follow Swift API Design Guidelines:

- Methods read as grammatical phrases
- Factory methods begin with "make"
- Boolean properties read as assertions (`isEmpty`, `canExecute`)

```swift
// CORRECT
func makeConnection(to socket: URL) -> Connection
var isRunning: Bool
func loadPreset(named name: String) async throws

// WRONG
func connection(socket: URL) -> Connection
var running: Bool
func load(name: String) async throws
```

## Error Handling

Use typed throws (Swift 6) when possible:

```swift
enum DaemonError: Error {
    case notRunning
    case connectionFailed(underlying: Error)
    case invalidResponse
}

func connect() async throws(DaemonError) {
    // ...
}
```

## SwiftUI View Structure

Keep views small and focused:

```swift
// CORRECT: Extracted subview
struct StatusRow: View {
    let status: DaemonStatus

    var body: some View {
        HStack {
            StatusIndicator(status: status)
            Text(status.displayName)
        }
    }
}
```

## State Management

- Use `@StateObject` for owned objects
- Use `@ObservedObject` for passed objects
- Keep business logic in ViewModels, not Views
