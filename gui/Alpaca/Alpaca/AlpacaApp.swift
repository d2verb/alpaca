import SwiftUI

@main
struct AlpacaApp: App {
    @NSApplicationDelegateAdaptor(AppDelegate.self) var appDelegate

    var body: some Scene {
        // No visible windows - menu bar only
        Settings {
            EmptyView()
        }
    }
}
