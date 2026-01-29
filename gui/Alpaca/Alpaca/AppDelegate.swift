import AppKit
import SwiftUI

@MainActor
final class AppDelegate: NSObject, NSApplicationDelegate, NSMenuDelegate {
    private var statusItem: NSStatusItem!
    private var menu: NSMenu!
    private let viewModel = AppViewModel(client: DaemonClient())
    private var menuBuilder: MenuBuilder!

    // Menu items that need dynamic updates
    private var statusMenuItem: NSMenuItem!
    private var errorMenuItem: NSMenuItem!
    private var browserSeparator: NSMenuItem!
    private var openInBrowserItem: NSMenuItem!
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
            button.image = NSImage(systemSymbolName: "brain", accessibilityDescription: "Alpaca")
        }
    }

    private func setupMenu() {
        menu = NSMenu()
        menuBuilder = MenuBuilder()

        // Status section - needs action and target to avoid grayed-out styling
        statusMenuItem = NSMenuItem()
        statusMenuItem.action = #selector(statusItemClicked)
        statusMenuItem.target = self
        menu.addItem(statusMenuItem)

        // Error message (shown when errorMessage is set)
        errorMenuItem = NSMenuItem()
        errorMenuItem.isHidden = true
        menu.addItem(errorMenuItem)

        // Separator before browser action
        browserSeparator = NSMenuItem.separator()
        menu.addItem(browserSeparator)

        // Open in Browser
        openInBrowserItem = NSMenuItem(title: "Open in Browser", action: #selector(openInBrowser), keyEquivalent: "")
        openInBrowserItem.target = self
        openInBrowserItem.image = NSImage(systemSymbolName: "safari.fill", accessibilityDescription: nil)
        menu.addItem(openInBrowserItem)

        // Separator before actions
        actionSeparator = NSMenuItem.separator()
        menu.addItem(actionSeparator)

        // Load/Switch Model submenu
        loadModelItem = NSMenuItem(title: "Load Model...", action: nil, keyEquivalent: "")
        loadModelItem.submenu = menuBuilder.createPresetsSubmenu(
            models: viewModel.models,
            presets: viewModel.presets,
            currentPreset: viewModel.state.currentPreset,
            target: self
        )
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

    private func updateMenu() {
        // Update status display
        statusMenuItem.attributedTitle = StatusFormatter.format(
            state: viewModel.state,
            errorMessage: viewModel.errorMessage
        )

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
        loadModelItem.submenu = menuBuilder.createPresetsSubmenu(
            models: viewModel.models,
            presets: viewModel.presets,
            currentPreset: viewModel.state.currentPreset,
            target: self
        )

        // Update menu visibility based on state
        menuBuilder.updateMenuVisibility(
            state: viewModel.state,
            browserSeparator: browserSeparator,
            openInBrowserItem: openInBrowserItem,
            loadModelItem: loadModelItem,
            stopItem: stopItem,
            cancelItem: cancelItem,
            actionSeparator: actionSeparator
        )
    }

    @objc func selectModel(_ sender: NSMenuItem) {
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

    @objc private func openInBrowser() {
        guard case .running(_, let endpoint) = viewModel.state else { return }
        guard let url = URL(string: endpoint) else { return }
        NSWorkspace.shared.open(url)
    }

    @objc private func statusItemClicked() {
        // No-op: status item is display-only but needs action to avoid gray styling
    }
}
