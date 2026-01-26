import Testing
@testable import Alpaca

@Suite("DaemonState Tests")
struct DaemonStateTests {
    @Test("Status text for notRunning state")
    func statusTextNotRunning() {
        let state = DaemonState.notRunning
        #expect(state.statusText == "Daemon not running")
    }

    @Test("Status text for idle state")
    func statusTextIdle() {
        let state = DaemonState.idle
        #expect(state.statusText == "Idle")
    }

    @Test("Status text for loading state")
    func statusTextLoading() {
        let state = DaemonState.loading(preset: "test-model")
        #expect(state.statusText == "Loading test-model...")
    }

    @Test("Status text for running state")
    func statusTextRunning() {
        let state = DaemonState.running(preset: "test-model", endpoint: "localhost:8080")
        #expect(state.statusText == "Running")
    }

    @Test("isRunning returns true only for running state")
    func isRunningProperty() {
        #expect(!DaemonState.notRunning.isRunning)
        #expect(!DaemonState.idle.isRunning)
        #expect(!DaemonState.loading(preset: "test").isRunning)
        #expect(DaemonState.running(preset: "test", endpoint: "localhost:8080").isRunning)
    }

    @Test("isLoading returns true only for loading state")
    func isLoadingProperty() {
        #expect(!DaemonState.notRunning.isLoading)
        #expect(!DaemonState.idle.isLoading)
        #expect(DaemonState.loading(preset: "test").isLoading)
        #expect(!DaemonState.running(preset: "test", endpoint: "localhost:8080").isLoading)
    }

    @Test("currentPreset returns preset name when applicable")
    func currentPresetProperty() {
        #expect(DaemonState.notRunning.currentPreset == nil)
        #expect(DaemonState.idle.currentPreset == nil)
        #expect(DaemonState.loading(preset: "loading-preset").currentPreset == "loading-preset")
        #expect(DaemonState.running(preset: "running-preset", endpoint: "localhost:8080").currentPreset == "running-preset")
    }
}

@Suite("Preset Tests")
struct PresetTests {
    @Test("Preset initialization")
    func presetInit() {
        let preset = Preset(name: "test-model")
        #expect(preset.name == "test-model")
        #expect(preset.id == "test-model")
    }

    @Test("Preset equality")
    func presetEquality() {
        let preset1 = Preset(name: "model-a")
        let preset2 = Preset(name: "model-a")
        let preset3 = Preset(name: "model-b")

        #expect(preset1 == preset2)
        #expect(preset1 != preset3)
    }
}
