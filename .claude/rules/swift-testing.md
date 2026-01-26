---
paths: "gui/**/*"
---

# Swift Testing Rules

## Test ViewModels, Not Views

```swift
@MainActor
func testStatusUpdatesOnConnect() async {
    let viewModel = MenuBarViewModel(client: mockClient)

    await viewModel.connect()

    XCTAssertEqual(viewModel.status, .connected)
}
```

## Mock Dependencies

Use protocols for testable dependencies:

```swift
protocol DaemonClientProtocol {
    func connect() async throws
    func send(_ command: Command) async throws -> Response
}

// In tests
class MockDaemonClient: DaemonClientProtocol {
    var connectCalled = false

    func connect() async throws {
        connectCalled = true
    }
    // ...
}
```

## What to Test

- ViewModel state transitions
- Business logic in services
- JSON encoding/decoding

## What Not to Test

- SwiftUI view layouts
- System framework behavior
