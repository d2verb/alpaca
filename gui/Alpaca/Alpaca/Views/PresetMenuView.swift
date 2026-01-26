import SwiftUI

/// Submenu showing available presets for selection.
struct PresetMenuView: View {
    @Bindable var viewModel: AppViewModel

    var body: some View {
        ForEach(viewModel.presets) { preset in
            Button {
                Task {
                    await viewModel.loadModel(preset: preset.name)
                }
            } label: {
                HStack {
                    if viewModel.state.currentPreset == preset.name {
                        Image(systemName: "checkmark")
                    }
                    Text(preset.name)
                }
            }
        }

        if viewModel.presets.isEmpty {
            Text("No presets available")
                .foregroundStyle(.secondary)
        }
    }
}

#Preview {
    let viewModel = AppViewModel(client: MockDaemonClient())
    return Menu("Presets") {
        PresetMenuView(viewModel: viewModel)
    }
}
