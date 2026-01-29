import Testing
import AppKit
@testable import Alpaca

// MARK: - MenuBuilder Tests

@Suite("MenuBuilder createPresetsSubmenu Tests")
@MainActor
struct MenuBuilderPresetsSubmenuTests {
    // Mock target for menu actions
    private final class MockTarget: NSObject {
        @objc func selectModel(_ sender: NSMenuItem) {
            // Mock action
        }
    }

    @Test("Create submenu with models only")
    func createSubmenuWithModelsOnly() {
        let builder = MenuBuilder()
        let models = [
            Model(repo: "TheBloke/CodeLlama-7B-GGUF", quant: "Q4_K_M", size: 4368438272),
            Model(repo: "TheBloke/Mistral-7B-GGUF", quant: "Q5_K_M", size: 5152665600)
        ]
        let presets: [Preset] = []
        let target = MockTarget()

        let submenu = builder.createPresetsSubmenu(
            models: models,
            presets: presets,
            currentPreset: nil,
            target: target
        )

        // Should have: header + 2 models = 3 items
        #expect(submenu.items.count == 3)

        // First item should be disabled header
        #expect(submenu.items[0].title == "Downloaded Models")
        #expect(submenu.items[0].isEnabled == false)

        // Second item should be first model
        #expect(submenu.items[1].title == "TheBloke/CodeLlama-7B-GGUF:Q4_K_M")
        #expect(submenu.items[1].representedObject as? String == "h:TheBloke/CodeLlama-7B-GGUF:Q4_K_M")
        #expect(submenu.items[1].state == .off)

        // Third item should be second model
        #expect(submenu.items[2].title == "TheBloke/Mistral-7B-GGUF:Q5_K_M")
        #expect(submenu.items[2].representedObject as? String == "h:TheBloke/Mistral-7B-GGUF:Q5_K_M")
    }

    @Test("Create submenu with presets only")
    func createSubmenuWithPresetsOnly() {
        let builder = MenuBuilder()
        let models: [Model] = []
        let presets = [
            Preset(name: "preset-1"),
            Preset(name: "preset-2"),
            Preset(name: "preset-3")
        ]
        let target = MockTarget()

        let submenu = builder.createPresetsSubmenu(
            models: models,
            presets: presets,
            currentPreset: nil,
            target: target
        )

        // Should have: header + 3 presets = 4 items
        #expect(submenu.items.count == 4)

        // First item should be disabled header
        #expect(submenu.items[0].title == "Presets")
        #expect(submenu.items[0].isEnabled == false)

        // Verify preset items
        #expect(submenu.items[1].title == "preset-1")
        #expect(submenu.items[1].representedObject as? String == "p:preset-1")
        #expect(submenu.items[2].title == "preset-2")
        #expect(submenu.items[3].title == "preset-3")
    }

    @Test("Create submenu with both models and presets")
    func createSubmenuWithBoth() {
        let builder = MenuBuilder()
        let models = [Model(repo: "TheBloke/CodeLlama-7B-GGUF", quant: "Q4_K_M", size: 4368438272)]
        let presets = [Preset(name: "my-preset")]
        let target = MockTarget()

        let submenu = builder.createPresetsSubmenu(
            models: models,
            presets: presets,
            currentPreset: nil,
            target: target
        )

        // Should have: models header + 1 model + separator + presets header + 1 preset = 5 items
        #expect(submenu.items.count == 5)

        // Verify structure
        #expect(submenu.items[0].title == "Downloaded Models")
        #expect(submenu.items[1].representedObject as? String == "h:TheBloke/CodeLlama-7B-GGUF:Q4_K_M")
        #expect(submenu.items[2].isSeparatorItem)
        #expect(submenu.items[3].title == "Presets")
        #expect(submenu.items[4].title == "my-preset")
    }

    @Test("Create submenu with empty lists")
    func createSubmenuWithEmptyLists() {
        let builder = MenuBuilder()
        let models: [Model] = []
        let presets: [Preset] = []
        let target = MockTarget()

        let submenu = builder.createPresetsSubmenu(
            models: models,
            presets: presets,
            currentPreset: nil,
            target: target
        )

        // Should have: 1 empty message item
        #expect(submenu.items.count == 1)
        #expect(submenu.items[0].title == "No models or presets available")
        #expect(submenu.items[0].isEnabled == false)
    }

    @Test("Mark current model with checkmark")
    func markCurrentModelWithCheckmark() {
        let builder = MenuBuilder()
        let models = [
            Model(repo: "TheBloke/CodeLlama-7B-GGUF", quant: "Q4_K_M", size: 4368438272),
            Model(repo: "TheBloke/Mistral-7B-GGUF", quant: "Q5_K_M", size: 5152665600)
        ]
        let target = MockTarget()
        let currentPreset = "h:TheBloke/Mistral-7B-GGUF:Q5_K_M"

        let submenu = builder.createPresetsSubmenu(
            models: models,
            presets: [],
            currentPreset: currentPreset,
            target: target
        )

        // First model should not have checkmark
        #expect(submenu.items[1].state == .off)

        // Second model should have checkmark
        #expect(submenu.items[2].state == .on)
    }

    @Test("Mark current preset with checkmark")
    func markCurrentPresetWithCheckmark() {
        let builder = MenuBuilder()
        let presets = [
            Preset(name: "preset-1"),
            Preset(name: "preset-2")
        ]
        let target = MockTarget()
        let currentPreset = "p:preset-1"

        let submenu = builder.createPresetsSubmenu(
            models: [],
            presets: presets,
            currentPreset: currentPreset,
            target: target
        )

        // First preset should have checkmark
        #expect(submenu.items[1].state == .on)

        // Second preset should not have checkmark
        #expect(submenu.items[2].state == .off)
    }

    @Test("Model items have tooltips with size info")
    func modelItemsHaveTooltips() {
        let builder = MenuBuilder()
        let models = [Model(repo: "TheBloke/CodeLlama-7B-GGUF", quant: "Q4_K_M", size: 4368438272)]
        let target = MockTarget()

        let submenu = builder.createPresetsSubmenu(
            models: models,
            presets: [],
            currentPreset: nil,
            target: target
        )

        let modelItem = submenu.items[1]
        #expect(modelItem.toolTip != nil)
        #expect(modelItem.toolTip?.contains("h:TheBloke/CodeLlama-7B-GGUF:Q4_K_M") == true)
        #expect(modelItem.toolTip?.contains("GB") == true)
    }
}

@Suite("MenuBuilder updateMenuVisibility Tests")
@MainActor
struct MenuBuilderVisibilityTests {
    private func createMenuItems() -> (
        browserSeparator: NSMenuItem,
        openInBrowserItem: NSMenuItem,
        loadModelItem: NSMenuItem,
        stopItem: NSMenuItem,
        cancelItem: NSMenuItem,
        actionSeparator: NSMenuItem
    ) {
        return (
            browserSeparator: NSMenuItem.separator(),
            openInBrowserItem: NSMenuItem(title: "Open in Browser", action: nil, keyEquivalent: ""),
            loadModelItem: NSMenuItem(title: "Load Model...", action: nil, keyEquivalent: ""),
            stopItem: NSMenuItem(title: "Stop", action: nil, keyEquivalent: ""),
            cancelItem: NSMenuItem(title: "Cancel", action: nil, keyEquivalent: ""),
            actionSeparator: NSMenuItem.separator()
        )
    }

    @Test("NotRunning state hides all action items")
    func notRunningStateHidesAll() {
        let builder = MenuBuilder()
        let items = createMenuItems()

        builder.updateMenuVisibility(
            state: .notRunning,
            browserSeparator: items.browserSeparator,
            openInBrowserItem: items.openInBrowserItem,
            loadModelItem: items.loadModelItem,
            stopItem: items.stopItem,
            cancelItem: items.cancelItem,
            actionSeparator: items.actionSeparator
        )

        #expect(items.browserSeparator.isHidden == true)
        #expect(items.openInBrowserItem.isHidden == true)
        #expect(items.loadModelItem.isHidden == true)
        #expect(items.stopItem.isHidden == true)
        #expect(items.cancelItem.isHidden == true)
        #expect(items.actionSeparator.isHidden == true)
    }

    @Test("Idle state shows Load Model only")
    func idleStateShowsLoadModel() {
        let builder = MenuBuilder()
        let items = createMenuItems()

        builder.updateMenuVisibility(
            state: .idle,
            browserSeparator: items.browserSeparator,
            openInBrowserItem: items.openInBrowserItem,
            loadModelItem: items.loadModelItem,
            stopItem: items.stopItem,
            cancelItem: items.cancelItem,
            actionSeparator: items.actionSeparator
        )

        #expect(items.browserSeparator.isHidden == true)
        #expect(items.openInBrowserItem.isHidden == true)
        #expect(items.loadModelItem.isHidden == false)
        #expect(items.loadModelItem.title == "Load Model...")
        #expect(items.stopItem.isHidden == true)
        #expect(items.cancelItem.isHidden == true)
        #expect(items.actionSeparator.isHidden == false)
    }

    @Test("Loading state shows Cancel only")
    func loadingStateShowsCancel() {
        let builder = MenuBuilder()
        let items = createMenuItems()

        builder.updateMenuVisibility(
            state: .loading(preset: "test-model"),
            browserSeparator: items.browserSeparator,
            openInBrowserItem: items.openInBrowserItem,
            loadModelItem: items.loadModelItem,
            stopItem: items.stopItem,
            cancelItem: items.cancelItem,
            actionSeparator: items.actionSeparator
        )

        #expect(items.browserSeparator.isHidden == true)
        #expect(items.openInBrowserItem.isHidden == true)
        #expect(items.loadModelItem.isHidden == true)
        #expect(items.stopItem.isHidden == true)
        #expect(items.cancelItem.isHidden == false)
        #expect(items.actionSeparator.isHidden == false)
    }

    @Test("Running state shows all action items")
    func runningStateShowsAll() {
        let builder = MenuBuilder()
        let items = createMenuItems()

        builder.updateMenuVisibility(
            state: .running(preset: "test-model", endpoint: "localhost:8080"),
            browserSeparator: items.browserSeparator,
            openInBrowserItem: items.openInBrowserItem,
            loadModelItem: items.loadModelItem,
            stopItem: items.stopItem,
            cancelItem: items.cancelItem,
            actionSeparator: items.actionSeparator
        )

        #expect(items.browserSeparator.isHidden == false)
        #expect(items.openInBrowserItem.isHidden == false)
        #expect(items.loadModelItem.isHidden == false)
        #expect(items.loadModelItem.title == "Switch Model...")
        #expect(items.stopItem.isHidden == false)
        #expect(items.cancelItem.isHidden == true)
        #expect(items.actionSeparator.isHidden == false)
    }

    @Test("Load Model item changes title based on state")
    func loadModelItemChangesTitle() {
        let builder = MenuBuilder()
        let loadModelItem = NSMenuItem(title: "", action: nil, keyEquivalent: "")

        // In idle state
        builder.updateMenuVisibility(
            state: .idle,
            browserSeparator: NSMenuItem.separator(),
            openInBrowserItem: NSMenuItem(),
            loadModelItem: loadModelItem,
            stopItem: NSMenuItem(),
            cancelItem: NSMenuItem(),
            actionSeparator: NSMenuItem.separator()
        )
        #expect(loadModelItem.title == "Load Model...")

        // In running state
        builder.updateMenuVisibility(
            state: .running(preset: "test", endpoint: "localhost:8080"),
            browserSeparator: NSMenuItem.separator(),
            openInBrowserItem: NSMenuItem(),
            loadModelItem: loadModelItem,
            stopItem: NSMenuItem(),
            cancelItem: NSMenuItem(),
            actionSeparator: NSMenuItem.separator()
        )
        #expect(loadModelItem.title == "Switch Model...")
    }
}
