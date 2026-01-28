import AppKit
import SwiftUI

@MainActor
final class AppDelegate: NSObject, NSApplicationDelegate, NSMenuDelegate {
    private var statusItem: NSStatusItem!
    private var menu: NSMenu!
    private let viewModel = AppViewModel(client: DaemonClient())

    // Menu items that need dynamic updates
    private var statusMenuItem: NSMenuItem!
    private var errorMenuItem: NSMenuItem!
    private var actionSeparator: NSMenuItem!
    private var loadModelItem: NSMenuItem!
    private var stopItem: NSMenuItem!
    private var cancelItem: NSMenuItem!

    func applicationDidFinishLaunching(_ notification: Notification) {
        setupStatusItem()
        setupMenu()
        Task {
            defer { updateMenu() }
            await viewModel.initialize()
        }
    }

    private func setupStatusItem() {
        statusItem = NSStatusBar.system.statusItem(withLength: NSStatusItem.variableLength)
        if let button = statusItem.button {
            button.image = NSImage(systemSymbolName: "hare.fill", accessibilityDescription: "Alpaca")
        }
    }

    private func setupMenu() {
        menu = NSMenu()

        // Status section - needs action and target to avoid grayed-out styling
        statusMenuItem = NSMenuItem()
        statusMenuItem.action = #selector(statusItemClicked)
        statusMenuItem.target = self
        menu.addItem(statusMenuItem)

        // Error message (shown when errorMessage is set)
        errorMenuItem = NSMenuItem()
        errorMenuItem.isHidden = true
        menu.addItem(errorMenuItem)

        // Separator before actions
        actionSeparator = NSMenuItem.separator()
        menu.addItem(actionSeparator)

        // Load/Switch Model submenu
        loadModelItem = NSMenuItem(title: "Load Model...", action: nil, keyEquivalent: "")
        loadModelItem.submenu = createPresetsSubmenu()
        menu.addItem(loadModelItem)

        // Stop item
        stopItem = NSMenuItem(title: "Stop", action: #selector(stopModel), keyEquivalent: "")
        stopItem.target = self
        stopItem.image = NSImage(systemSymbolName: "stop.fill", accessibilityDescription: nil)
        menu.addItem(stopItem)

        // Cancel item (for loading state)
        cancelItem = NSMenuItem(title: "Cancel", action: #selector(stopModel), keyEquivalent: "")
        cancelItem.target = self
        cancelItem.image = NSImage(systemSymbolName: "xmark", accessibilityDescription: nil)
        menu.addItem(cancelItem)

        // Separator before footer
        menu.addItem(NSMenuItem.separator())

        // Preferences (disabled for now)
        let prefsItem = NSMenuItem(title: "Preferences...", action: nil, keyEquivalent: ",")
        prefsItem.image = NSImage(systemSymbolName: "gearshape", accessibilityDescription: nil)
        prefsItem.isEnabled = false
        menu.addItem(prefsItem)

        // Quit (no icon per macOS HIG)
        let quitItem = NSMenuItem(title: "Quit Alpaca", action: #selector(NSApplication.terminate(_:)), keyEquivalent: "q")
        menu.addItem(quitItem)

        menu.delegate = self
        statusItem.menu = menu
    }

    // MARK: - NSMenuDelegate

    func menuWillOpen(_ menu: NSMenu) {
        updateMenu()
        Task {
            await viewModel.refreshStatus()
            await viewModel.loadPresets()
            await viewModel.loadModels()
            updateMenu()
        }
    }

    private func createPresetsSubmenu() -> NSMenu {
        let submenu = NSMenu()

        // Downloaded Models section
        if !viewModel.models.isEmpty {
            let modelsHeader = NSMenuItem(title: "Downloaded Models", action: nil, keyEquivalent: "")
            modelsHeader.isEnabled = false
            submenu.addItem(modelsHeader)

            for model in viewModel.models {
                let item = NSMenuItem(title: model.displayName, action: #selector(selectModel(_:)), keyEquivalent: "")
                item.target = self
                item.representedObject = model.identifier
                item.toolTip = "\(model.identifier) (\(model.sizeString))"
                if viewModel.state.currentPreset == model.identifier {
                    item.state = .on
                }
                submenu.addItem(item)
            }
        }

        // Separator between models and presets
        if !viewModel.models.isEmpty && !viewModel.presets.isEmpty {
            submenu.addItem(NSMenuItem.separator())
        }

        // Presets section
        if !viewModel.presets.isEmpty {
            let presetsHeader = NSMenuItem(title: "Presets", action: nil, keyEquivalent: "")
            presetsHeader.isEnabled = false
            submenu.addItem(presetsHeader)

            for preset in viewModel.presets {
                let item = NSMenuItem(title: preset.name, action: #selector(selectModel(_:)), keyEquivalent: "")
                item.target = self
                item.representedObject = preset.name
                if viewModel.state.currentPreset == preset.name {
                    item.state = .on
                }
                submenu.addItem(item)
            }
        }

        // Show message if both are empty
        if viewModel.models.isEmpty && viewModel.presets.isEmpty {
            let emptyItem = NSMenuItem(title: "No models or presets available", action: nil, keyEquivalent: "")
            emptyItem.isEnabled = false
            submenu.addItem(emptyItem)
        }

        return submenu
    }

    private func updateMenu() {
        // Update status display
        statusMenuItem.attributedTitle = createStatusAttributedString()

        // Update error display
        if let errorMessage = viewModel.errorMessage {
            errorMenuItem.attributedTitle = NSAttributedString(
                string: "⚠ \(errorMessage)",
                attributes: [
                    .font: NSFont.systemFont(ofSize: 11),
                    .foregroundColor: NSColor.systemRed
                ]
            )
            errorMenuItem.isHidden = false
        } else {
            errorMenuItem.isHidden = true
        }

        // Update presets submenu
        loadModelItem.submenu = createPresetsSubmenu()

        // Show/hide items based on state
        switch viewModel.state {
        case .notRunning:
            loadModelItem.isHidden = true
            stopItem.isHidden = true
            cancelItem.isHidden = true
            actionSeparator.isHidden = true

        case .idle:
            loadModelItem.isHidden = false
            loadModelItem.title = "Load Model..."
            loadModelItem.image = NSImage(systemSymbolName: "play.fill", accessibilityDescription: nil)
            stopItem.isHidden = true
            cancelItem.isHidden = true
            actionSeparator.isHidden = false

        case .loading:
            loadModelItem.isHidden = true
            stopItem.isHidden = true
            cancelItem.isHidden = false
            actionSeparator.isHidden = false

        case .running:
            loadModelItem.isHidden = false
            loadModelItem.title = "Switch Model..."
            loadModelItem.image = NSImage(systemSymbolName: "play.fill", accessibilityDescription: nil)
            stopItem.isHidden = false
            cancelItem.isHidden = true
            actionSeparator.isHidden = false
        }
    }

    private func createStatusAttributedString() -> NSAttributedString {
        let result = NSMutableAttributedString()

        // Paragraph style to prevent automatic indentation on subsequent lines
        let paragraphStyle = NSMutableParagraphStyle()
        paragraphStyle.headIndent = 0
        paragraphStyle.firstLineHeadIndent = 0

        // Status indicator and text
        let (indicator, color) = statusIndicator()
        let indicatorAttr = NSAttributedString(
            string: "\(indicator) ",
            attributes: [
                .font: NSFont.systemFont(ofSize: 13),
                .foregroundColor: color,
                .paragraphStyle: paragraphStyle
            ]
        )
        result.append(indicatorAttr)

        let statusText = NSAttributedString(
            string: viewModel.state.statusText,
            attributes: [
                .font: NSFont.boldSystemFont(ofSize: 13),
                .foregroundColor: NSColor.labelColor,
                .paragraphStyle: paragraphStyle
            ]
        )
        result.append(statusText)

        // Additional info based on state
        switch viewModel.state {
        case .notRunning:
            break

        case .idle:
            // Add spacing using a small font line
            result.append(NSAttributedString(
                string: "\n \n",
                attributes: [.font: NSFont.systemFont(ofSize: 4), .paragraphStyle: paragraphStyle]
            ))
            let subtitleAttr = NSAttributedString(
                string: "No model loaded",
                attributes: [
                    .font: NSFont.systemFont(ofSize: 11),
                    .foregroundColor: NSColor.secondaryLabelColor,
                    .paragraphStyle: paragraphStyle
                ]
            )
            result.append(subtitleAttr)

        case .loading(let preset):
            result.append(NSAttributedString(string: "\n", attributes: [.paragraphStyle: paragraphStyle]))
            let subtitleAttr = NSAttributedString(
                string: preset,
                attributes: [
                    .font: NSFont.systemFont(ofSize: 11),
                    .foregroundColor: NSColor.secondaryLabelColor,
                    .paragraphStyle: paragraphStyle
                ]
            )
            result.append(subtitleAttr)

        case .running(let preset, let endpoint):
            // Add spacing using a small font line
            result.append(NSAttributedString(
                string: "\n \n",
                attributes: [.font: NSFont.systemFont(ofSize: 4), .paragraphStyle: paragraphStyle]
            ))
            let presetAttr = NSAttributedString(
                string: preset,
                attributes: [
                    .font: NSFont.systemFont(ofSize: 11),
                    .foregroundColor: NSColor.labelColor,
                    .paragraphStyle: paragraphStyle
                ]
            )
            result.append(presetAttr)
            result.append(NSAttributedString(string: "\n", attributes: [.paragraphStyle: paragraphStyle]))
            let endpointAttr = NSAttributedString(
                string: endpoint,
                attributes: [
                    .font: NSFont.monospacedSystemFont(ofSize: 11, weight: .regular),
                    .foregroundColor: NSColor.labelColor,
                    .paragraphStyle: paragraphStyle
                ]
            )
            result.append(endpointAttr)
        }

        return result
    }

    private func statusIndicator() -> (String, NSColor) {
        switch viewModel.state {
        case .notRunning:
            return ("○", .systemGray)
        case .idle:
            return ("●", .systemYellow)
        case .loading:
            return ("◐", .systemBlue)
        case .running:
            return ("●", .systemGreen)
        }
    }

    @objc private func selectModel(_ sender: NSMenuItem) {
        guard let identifier = sender.representedObject as? String else { return }
        Task {
            await viewModel.loadModel(identifier: identifier)
            updateMenu()
        }
    }

    @objc private func stopModel() {
        Task {
            await viewModel.stopModel()
            updateMenu()
        }
    }

    @objc private func statusItemClicked() {
        // No-op: status item is display-only but needs action to avoid gray styling
    }
}
