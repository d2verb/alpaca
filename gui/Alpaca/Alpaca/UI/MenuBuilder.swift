import AppKit
import Foundation

/// Builds and updates menu items for the Alpaca status bar.
@MainActor
final class MenuBuilder {
    /// Creates a submenu for loading/switching models and presets.
    func createPresetsSubmenu(
        models: [Model],
        presets: [Preset],
        currentPreset: String?,
        target: AnyObject
    ) -> NSMenu {
        let submenu = NSMenu()

        // Downloaded Models section
        if !models.isEmpty {
            let modelsHeader = NSMenuItem(title: "Downloaded Models", action: nil, keyEquivalent: "")
            modelsHeader.isEnabled = false
            submenu.addItem(modelsHeader)

            for model in models {
                let item = NSMenuItem(title: model.displayName, action: #selector(AppDelegate.selectModel(_:)), keyEquivalent: "")
                item.target = target
                item.representedObject = model.identifier
                item.toolTip = "\(model.identifier) (\(model.sizeString))"
                if currentPreset == model.identifier {
                    item.state = .on
                }
                submenu.addItem(item)
            }
        }

        // Separator between models and presets
        if !models.isEmpty && !presets.isEmpty {
            submenu.addItem(NSMenuItem.separator())
        }

        // Presets section
        if !presets.isEmpty {
            let presetsHeader = NSMenuItem(title: "Presets", action: nil, keyEquivalent: "")
            presetsHeader.isEnabled = false
            submenu.addItem(presetsHeader)

            for preset in presets {
                let item = NSMenuItem(title: preset.name, action: #selector(AppDelegate.selectModel(_:)), keyEquivalent: "")
                item.target = target
                item.representedObject = preset.identifier
                if currentPreset == preset.identifier {
                    item.state = .on
                }
                submenu.addItem(item)
            }
        }

        // Show message if both are empty
        if models.isEmpty && presets.isEmpty {
            let emptyItem = NSMenuItem(title: "No models or presets available", action: nil, keyEquivalent: "")
            emptyItem.isEnabled = false
            submenu.addItem(emptyItem)
        }

        return submenu
    }

    /// Updates menu item visibility based on daemon state.
    func updateMenuVisibility(
        state: DaemonState,
        browserSeparator: NSMenuItem,
        openInBrowserItem: NSMenuItem,
        loadModelItem: NSMenuItem,
        stopItem: NSMenuItem,
        cancelItem: NSMenuItem,
        actionSeparator: NSMenuItem
    ) {
        switch state {
        case .notRunning:
            browserSeparator.isHidden = true
            openInBrowserItem.isHidden = true
            loadModelItem.isHidden = true
            stopItem.isHidden = true
            cancelItem.isHidden = true
            actionSeparator.isHidden = true

        case .idle:
            browserSeparator.isHidden = true
            openInBrowserItem.isHidden = true
            loadModelItem.isHidden = false
            loadModelItem.title = "Load Model..."
            loadModelItem.image = NSImage(systemSymbolName: "play.fill", accessibilityDescription: nil)
            stopItem.isHidden = true
            cancelItem.isHidden = true
            actionSeparator.isHidden = false

        case .loading:
            browserSeparator.isHidden = true
            openInBrowserItem.isHidden = true
            loadModelItem.isHidden = true
            stopItem.isHidden = true
            cancelItem.isHidden = false
            actionSeparator.isHidden = false

        case .running:
            browserSeparator.isHidden = false
            openInBrowserItem.isHidden = false
            loadModelItem.isHidden = false
            loadModelItem.title = "Switch Model..."
            loadModelItem.image = NSImage(systemSymbolName: "play.fill", accessibilityDescription: nil)
            stopItem.isHidden = false
            cancelItem.isHidden = true
            actionSeparator.isHidden = false
        }
    }
}
