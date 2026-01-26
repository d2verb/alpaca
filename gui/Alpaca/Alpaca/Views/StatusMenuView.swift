import SwiftUI

/// Main popover content showing daemon status and controls.
struct StatusMenuView: View {
    @Bindable var viewModel: AppViewModel
    @State private var showCopiedFeedback = false

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            statusSection
            if viewModel.errorMessage != nil {
                errorSection
            }
            Divider()
            actionSection
            Divider()
            footerSection
        }
        .frame(width: 240)
    }

    @ViewBuilder
    private var statusSection: some View {
        VStack(alignment: .leading, spacing: 4) {
            HStack(spacing: 8) {
                statusIndicator
                Text(viewModel.state.statusText)
                    .font(.headline)
            }

            switch viewModel.state {
            case .notRunning:
                HStack {
                    Text("$ alpaca start")
                        .font(.system(.caption, design: .monospaced))
                        .foregroundStyle(.secondary)
                    Spacer()
                    Button {
                        viewModel.copyStartCommand()
                        showCopiedFeedback = true
                        Task {
                            try? await Task.sleep(for: .seconds(1.5))
                            showCopiedFeedback = false
                        }
                    } label: {
                        Image(systemName: showCopiedFeedback ? "checkmark" : "doc.on.doc")
                            .foregroundStyle(showCopiedFeedback ? .green : .secondary)
                    }
                    .buttonStyle(.plain)
                    .help("Copy command")
                }
            case .idle:
                Text("No model loaded")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            case .loading(let preset):
                Text(preset)
                    .font(.caption)
                    .foregroundStyle(.secondary)
            case .running(let preset, let endpoint):
                Text(preset)
                    .font(.caption)
                Text(endpoint)
                    .font(.system(.caption, design: .monospaced))
                    .foregroundStyle(.secondary)
            }
        }
        .padding(12)
    }

    @ViewBuilder
    private var errorSection: some View {
        if let error = viewModel.errorMessage {
            HStack(spacing: 4) {
                Image(systemName: "exclamationmark.triangle.fill")
                    .foregroundStyle(.yellow)
                Text(error)
                    .font(.caption)
                    .foregroundStyle(.secondary)
                    .lineLimit(2)
            }
            .padding(.horizontal, 12)
            .padding(.bottom, 8)
        }
    }

    @ViewBuilder
    private var statusIndicator: some View {
        switch viewModel.state {
        case .notRunning:
            Circle()
                .fill(.gray)
                .frame(width: 10, height: 10)
        case .idle:
            Circle()
                .fill(.yellow)
                .frame(width: 10, height: 10)
        case .loading:
            ProgressView()
                .scaleEffect(0.5)
                .frame(width: 10, height: 10)
        case .running:
            Circle()
                .fill(.green)
                .frame(width: 10, height: 10)
        }
    }

    @ViewBuilder
    private var actionSection: some View {
        VStack(alignment: .leading, spacing: 0) {
            switch viewModel.state {
            case .notRunning:
                EmptyView()
            case .idle:
                Menu {
                    PresetMenuView(viewModel: viewModel)
                } label: {
                    HStack {
                        Image(systemName: "play.fill")
                            .frame(width: 16)
                        Text("Load Model...")
                        Spacer()
                        Image(systemName: "chevron.right")
                            .foregroundStyle(.secondary)
                    }
                }
                .menuStyle(.borderlessButton)
                .padding(.horizontal, 12)
                .padding(.vertical, 8)
            case .loading:
                Button {
                    Task {
                        await viewModel.stopModel()
                    }
                } label: {
                    HStack {
                        Image(systemName: "xmark")
                            .frame(width: 16)
                        Text("Cancel")
                        Spacer()
                    }
                }
                .buttonStyle(.plain)
                .padding(.horizontal, 12)
                .padding(.vertical, 8)
            case .running:
                Menu {
                    PresetMenuView(viewModel: viewModel)
                } label: {
                    HStack {
                        Image(systemName: "play.fill")
                            .frame(width: 16)
                        Text("Switch Model...")
                        Spacer()
                        Image(systemName: "chevron.right")
                            .foregroundStyle(.secondary)
                    }
                }
                .menuStyle(.borderlessButton)
                .padding(.horizontal, 12)
                .padding(.vertical, 8)

                Button {
                    Task {
                        await viewModel.stopModel()
                    }
                } label: {
                    HStack {
                        Image(systemName: "stop.fill")
                            .frame(width: 16)
                        Text("Stop")
                        Spacer()
                    }
                }
                .buttonStyle(.plain)
                .padding(.horizontal, 12)
                .padding(.vertical, 8)
            }
        }
    }

    @ViewBuilder
    private var footerSection: some View {
        VStack(alignment: .leading, spacing: 0) {
            Button {
                NSApplication.shared.terminate(nil)
            } label: {
                HStack {
                    Text("Quit Alpaca")
                    Spacer()
                    Text("⌘Q")
                        .foregroundStyle(.secondary)
                }
            }
            .buttonStyle(.plain)
            .padding(.horizontal, 12)
            .padding(.vertical, 8)
            .keyboardShortcut("q", modifiers: .command)
        }
    }
}

#Preview {
    let viewModel = AppViewModel(client: MockDaemonClient())
    return StatusMenuView(viewModel: viewModel)
}
