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
    private var copyCommandItem: NSMenuItem!
    private var actionSeparator: NSMenuItem!
    private var loadModelItem: NSMenuItem!
    private var stopItem: NSMenuItem!
    private var cancelItem: NSMenuItem!

    // For cancelling pending copy feedback reset
    private var copyCommandResetWorkItem: DispatchWorkItem?

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

        // Copy command item (shown in notRunning state)
        copyCommandItem = NSMenuItem(title: "Copy \"alpaca start\"", action: #selector(copyStartCommand), keyEquivalent: "")
        copyCommandItem.target = self
        copyCommandItem.image = NSImage(systemSymbolName: "doc.on.doc", accessibilityDescription: nil)
        copyCommandItem.isHidden = true
        menu.addItem(copyCommandItem)

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
            updateMenu()
        }
    }

    private func createPresetsSubmenu() -> NSMenu {
        let submenu = NSMenu()
        for preset in viewModel.presets {
            let item = NSMenuItem(title: preset.name, action: #selector(selectPreset(_:)), keyEquivalent: "")
            item.target = self
            item.representedObject = preset.name
            if viewModel.state.currentPreset == preset.name {
                item.state = .on
            }
            submenu.addItem(item)
        }
        if viewModel.presets.isEmpty {
            let emptyItem = NSMenuItem(title: "No presets available", action: nil, keyEquivalent: "")
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
            copyCommandItem.isHidden = false
            loadModelItem.isHidden = true
            stopItem.isHidden = true
            cancelItem.isHidden = true
            actionSeparator.isHidden = false

        case .idle:
            copyCommandItem.isHidden = true
            loadModelItem.isHidden = false
            loadModelItem.title = "Load Model..."
            loadModelItem.image = NSImage(systemSymbolName: "play.fill", accessibilityDescription: nil)
            stopItem.isHidden = true
            cancelItem.isHidden = true
            actionSeparator.isHidden = false

        case .loading:
            copyCommandItem.isHidden = true
            loadModelItem.isHidden = true
            stopItem.isHidden = true
            cancelItem.isHidden = false
            actionSeparator.isHidden = false

        case .running:
            copyCommandItem.isHidden = true
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

        // Status indicator and text
        let (indicator, color) = statusIndicator()
        let indicatorAttr = NSAttributedString(
            string: "\(indicator) ",
            attributes: [.foregroundColor: color]
        )
        result.append(indicatorAttr)

        let statusText = NSAttributedString(
            string: viewModel.state.statusText,
            attributes: [
                .font: NSFont.boldSystemFont(ofSize: 13),
                .foregroundColor: NSColor.labelColor
            ]
        )
        result.append(statusText)

        // Additional info based on state
        switch viewModel.state {
        case .notRunning:
            result.append(NSAttributedString(string: "\n"))
            let commandAttr = NSAttributedString(
                string: "$ alpaca start",
                attributes: [
                    .font: NSFont.monospacedSystemFont(ofSize: 11, weight: .regular),
                    .foregroundColor: NSColor.secondaryLabelColor
                ]
            )
            result.append(commandAttr)

        case .idle:
            // Add spacing using a small font line
            result.append(NSAttributedString(
                string: "\n \n",
                attributes: [.font: NSFont.systemFont(ofSize: 4)]
            ))
            let subtitleAttr = NSAttributedString(
                string: "No model loaded",
                attributes: [
                    .font: NSFont.systemFont(ofSize: 11),
                    .foregroundColor: NSColor.secondaryLabelColor
                ]
            )
            result.append(subtitleAttr)

        case .loading(let preset):
            result.append(NSAttributedString(string: "\n"))
            let subtitleAttr = NSAttributedString(
                string: preset,
                attributes: [
                    .font: NSFont.systemFont(ofSize: 11),
                    .foregroundColor: NSColor.secondaryLabelColor
                ]
            )
            result.append(subtitleAttr)

        case .running(let preset, let endpoint):
            // Add spacing using a small font line
            result.append(NSAttributedString(
                string: "\n \n",
                attributes: [.font: NSFont.systemFont(ofSize: 4)]
            ))
            let presetAttr = NSAttributedString(
                string: preset,
                attributes: [
                    .font: NSFont.systemFont(ofSize: 11),
                    .foregroundColor: NSColor.labelColor
                ]
            )
            result.append(presetAttr)
            result.append(NSAttributedString(string: "\n"))
            let endpointAttr = NSAttributedString(
                string: endpoint,
                attributes: [
                    .font: NSFont.monospacedSystemFont(ofSize: 11, weight: .regular),
                    .foregroundColor: NSColor.labelColor
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

    @objc private func selectPreset(_ sender: NSMenuItem) {
        guard let presetName = sender.representedObject as? String else { return }
        Task {
            await viewModel.loadModel(preset: presetName)
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

    @objc private func copyStartCommand() {
        viewModel.copyStartCommand()

        // Cancel any pending reset from previous clicks
        copyCommandResetWorkItem?.cancel()

        // Show "Copied!" feedback
        copyCommandItem.title = "Copied!"

        // Schedule reset after delay
        let workItem = DispatchWorkItem { [weak self] in
            self?.copyCommandItem.title = "Copy \"alpaca start\""
        }
        copyCommandResetWorkItem = workItem
        DispatchQueue.main.asyncAfter(deadline: .now() + 1.5, execute: workItem)
    }
}
