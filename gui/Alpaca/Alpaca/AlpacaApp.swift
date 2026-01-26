import SwiftUI

@main
struct AlpacaApp: App {
    @State private var viewModel = AppViewModel(client: MockDaemonClient())

    var body: some Scene {
        MenuBarExtra {
            StatusMenuView(viewModel: viewModel)
                .task {
                    await viewModel.initialize()
                }
        } label: {
            Image(systemName: "hare.fill")
                .symbolRenderingMode(.hierarchical)
        }
        .menuBarExtraStyle(.window)
    }
}
