import Testing
@testable import Alpaca

// MARK: - DaemonState Tests

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
        #expect(preset.id == "p:test-model")
        #expect(preset.identifier == "p:test-model")
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

@Suite("Model Tests")
struct ModelTests {
    @Test("Model initialization")
    func modelInit() {
        let model = Model(repo: "unsloth/gemma3", quant: "Q4_K_M", size: 1024)
        #expect(model.repo == "unsloth/gemma3")
        #expect(model.quant == "Q4_K_M")
        #expect(model.id == "h:unsloth/gemma3:Q4_K_M")
        #expect(model.identifier == "h:unsloth/gemma3:Q4_K_M")
    }

    @Test("Model display name formatting")
    func modelDisplayName() {
        let model = Model(repo: "TheBloke/CodeLlama-7B-GGUF", quant: "Q4_K_M", size: 1024)
        #expect(model.displayName == "CodeLlama-7B (Q4_K_M)")
    }

    @Test("Model size string formatting")
    func modelSizeString() {
        let model = Model(repo: "test/repo", quant: "Q4_K_M", size: 1073741824) // 1 GB
        #expect(model.sizeString.contains("GB"))
    }
}

// MARK: - AppViewModel Tests

@Suite("AppViewModel Tests")
struct AppViewModelTests {
    @Test("Initialize loads status and presets")
    @MainActor
    func initializeLoadsData() async {
        let client = TestDaemonClient()
        await client.configure(
            status: .running(preset: "test-model", endpoint: "localhost:8080"),
            presets: [Preset(name: "preset-1"), Preset(name: "preset-2")]
        )
        let viewModel = AppViewModel(client: client)

        await viewModel.initialize()

        #expect(viewModel.state == .running(preset: "test-model", endpoint: "localhost:8080"))
        #expect(viewModel.presets.count == 2)
        #expect(viewModel.errorMessage == nil)
    }

    @Test("RefreshStatus updates state from daemon")
    @MainActor
    func refreshStatusUpdatesState() async {
        let client = TestDaemonClient()
        await client.configure(status: .idle)
        let viewModel = AppViewModel(client: client)

        await viewModel.refreshStatus()

        #expect(viewModel.state == .idle)
        #expect(viewModel.errorMessage == nil)
    }

    @Test("RefreshStatus handles notRunning without error message")
    @MainActor
    func refreshStatusHandlesNotRunning() async {
        let client = TestDaemonClient()
        await client.configure(error: DaemonError.notRunning)
        let viewModel = AppViewModel(client: client)

        await viewModel.refreshStatus()

        #expect(viewModel.state == .notRunning)
        #expect(viewModel.errorMessage == nil)
    }

    @Test("RefreshStatus sets error message on failure")
    @MainActor
    func refreshStatusSetsErrorOnFailure() async {
        let client = TestDaemonClient()
        await client.configure(error: DaemonError.protocolError("Test error"))
        let viewModel = AppViewModel(client: client)

        await viewModel.refreshStatus()

        #expect(viewModel.errorMessage != nil)
    }

    @Test("LoadPresets populates presets array")
    @MainActor
    func loadPresetsPopulatesArray() async {
        let client = TestDaemonClient()
        let testPresets = [Preset(name: "model-a"), Preset(name: "model-b")]
        await client.configure(presets: testPresets)
        let viewModel = AppViewModel(client: client)

        await viewModel.loadPresets()

        #expect(viewModel.presets == testPresets)
    }

    @Test("LoadModel calls client and refreshes status")
    @MainActor
    func loadModelCallsClientAndRefreshes() async {
        let client = TestDaemonClient()
        await client.configure(status: .running(preset: "new-model", endpoint: "localhost:8080"))
        let viewModel = AppViewModel(client: client)

        await viewModel.loadModel(identifier: "new-model")

        let loadModelCalled = await client.loadModelCalled
        let lastIdentifier = await client.lastLoadedIdentifier
        #expect(loadModelCalled)
        #expect(lastIdentifier == "new-model")
        #expect(viewModel.state == .running(preset: "new-model", endpoint: "localhost:8080"))
    }

    @Test("StopModel calls client and refreshes status")
    @MainActor
    func stopModelCallsClientAndRefreshes() async {
        let client = TestDaemonClient()
        await client.configure(status: .idle)
        let viewModel = AppViewModel(client: client)

        await viewModel.stopModel()

        let stopModelCalled = await client.stopModelCalled
        #expect(stopModelCalled)
        #expect(viewModel.state == .idle)
    }

    @Test("LoadModel keeps error message after subsequent refresh")
    @MainActor
    func loadModelKeepsErrorAfterRefresh() async {
        let client = TestDaemonClient()
        await client.configure(status: .idle, loadError: DaemonError.presetNotFound("test error"))
        let viewModel = AppViewModel(client: client)

        // Load model fails and sets error
        await viewModel.loadModel(identifier: "p:nonexistent")
        #expect(viewModel.errorMessage != nil)
        let errorBeforeRefresh = viewModel.errorMessage

        // Clear load error for next refresh
        await client.configure(status: .idle)

        // Refresh status should NOT clear the error
        await viewModel.refreshStatus()

        #expect(viewModel.errorMessage == errorBeforeRefresh)
        #expect(viewModel.errorMessage != nil)
    }

    @Test("LoadModel clears previous error before starting")
    @MainActor
    func loadModelClearsPreviousError() async {
        let client = TestDaemonClient()
        await client.configure(
            status: .idle,
            loadError: DaemonError.presetNotFound("first error")
        )
        let viewModel = AppViewModel(client: client)

        // First load fails and sets error
        await viewModel.loadModel(identifier: "p:bad")
        #expect(viewModel.errorMessage != nil)

        // Configure for successful second load
        await client.configure(status: .running(preset: "test", endpoint: "localhost:8080"))

        // Second load succeeds - previous error should be cleared
        await viewModel.loadModel(identifier: "p:test")
        #expect(viewModel.errorMessage == nil)
    }
}

// MARK: - DaemonError Tests

@Suite("DaemonError Tests")
struct DaemonErrorTests {
    @Test("NotRunning error description")
    func notRunningDescription() {
        let error = DaemonError.notRunning
        #expect(error.errorDescription == "Daemon is not running")
    }

    @Test("ProtocolError error description")
    func protocolErrorDescription() {
        let error = DaemonError.protocolError("Something went wrong")
        #expect(error.errorDescription == "Something went wrong")
    }

    @Test("InvalidResponse error description")
    func invalidResponseDescription() {
        let error = DaemonError.invalidResponse("Bad data")
        #expect(error.errorDescription == "Invalid response: Bad data")
    }

    @Test("PresetNotFound error description")
    func presetNotFoundDescription() {
        let error = DaemonError.presetNotFound("my-preset")
        #expect(error.errorDescription == "Preset not found: my-preset")
    }

    @Test("ModelNotFound error description")
    func modelNotFoundDescription() {
        let error = DaemonError.modelNotFound("h:org/repo:Q4")
        #expect(error.errorDescription == "Model not found: h:org/repo:Q4")
    }

    @Test("fromCode creates correct error types")
    func fromCodeCreatesCorrectTypes() {
        let presetError = DaemonError.fromCode("preset_not_found", message: "test")
        let modelError = DaemonError.fromCode("model_not_found", message: "test")
        let serverError = DaemonError.fromCode("server_failed", message: "test")
        let unknownError = DaemonError.fromCode("unknown_code", message: "test")
        let nilCodeError = DaemonError.fromCode(nil, message: "test")

        if case .presetNotFound = presetError {} else { Issue.record("Expected presetNotFound") }
        if case .modelNotFound = modelError {} else { Issue.record("Expected modelNotFound") }
        if case .serverFailed = serverError {} else { Issue.record("Expected serverFailed") }
        if case .protocolError = unknownError {} else { Issue.record("Expected protocolError for unknown code") }
        if case .protocolError = nilCodeError {} else { Issue.record("Expected protocolError for nil code") }
    }
}
