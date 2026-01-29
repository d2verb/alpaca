import Testing
import AppKit
@testable import Alpaca

// MARK: - StatusFormatter Tests

@Suite("StatusFormatter format Tests")
@MainActor
struct StatusFormatterFormatTests {
    @Test("Format notRunning state")
    func formatNotRunningState() {
        let result = StatusFormatter.format(state: .notRunning)
        let string = result.string

        // Should contain indicator and status text
        #expect(string.contains("○"))
        #expect(string.contains("Daemon not running"))

        // Should not have additional info
        let lines = string.components(separatedBy: "\n").filter { !$0.trimmingCharacters(in: .whitespaces).isEmpty }
        #expect(lines.count == 1)
    }

    @Test("Format idle state")
    func formatIdleState() {
        let result = StatusFormatter.format(state: .idle)
        let string = result.string

        // Should contain indicator and status text
        #expect(string.contains("●"))
        #expect(string.contains("Idle"))

        // Should have subtitle "No model loaded"
        #expect(string.contains("No model loaded"))

        // Verify attributed string has correct structure
        #expect(result.length > 0)
    }

    @Test("Format loading state")
    func formatLoadingState() {
        let result = StatusFormatter.format(state: .loading(preset: "test-model"))
        let string = result.string

        // Should contain indicator and status text
        #expect(string.contains("◐"))
        #expect(string.contains("Loading test-model..."))

        // Should have preset name as subtitle
        #expect(string.contains("test-model"))
    }

    @Test("Format running state")
    func formatRunningState() {
        let result = StatusFormatter.format(state: .running(preset: "my-preset", endpoint: "http://localhost:8080"))
        let string = result.string

        // Should contain indicator and status text
        #expect(string.contains("●"))
        #expect(string.contains("Running"))

        // Should have preset name
        #expect(string.contains("my-preset"))

        // Should have endpoint
        #expect(string.contains("http://localhost:8080"))
    }

    @Test("NotRunning state has gray indicator")
    func notRunningHasGrayIndicator() {
        let result = StatusFormatter.format(state: .notRunning)

        // Check that first character has gray color
        var effectiveRange = NSRange(location: 0, length: 0)
        if let color = result.attribute(.foregroundColor, at: 0, effectiveRange: &effectiveRange) as? NSColor {
            // systemGray comparison
            #expect(color == .systemGray)
        }
    }

    @Test("Idle state has yellow indicator")
    func idleHasYellowIndicator() {
        let result = StatusFormatter.format(state: .idle)

        // Check that first character has yellow color
        var effectiveRange = NSRange(location: 0, length: 0)
        if let color = result.attribute(.foregroundColor, at: 0, effectiveRange: &effectiveRange) as? NSColor {
            #expect(color == .systemYellow)
        }
    }

    @Test("Loading state has blue indicator")
    func loadingHasBlueIndicator() {
        let result = StatusFormatter.format(state: .loading(preset: "test"))

        var effectiveRange = NSRange(location: 0, length: 0)
        if let color = result.attribute(.foregroundColor, at: 0, effectiveRange: &effectiveRange) as? NSColor {
            #expect(color == .systemBlue)
        }
    }

    @Test("Running state has green indicator")
    func runningHasGreenIndicator() {
        let result = StatusFormatter.format(state: .running(preset: "test", endpoint: "localhost:8080"))

        var effectiveRange = NSRange(location: 0, length: 0)
        if let color = result.attribute(.foregroundColor, at: 0, effectiveRange: &effectiveRange) as? NSColor {
            #expect(color == .systemGreen)
        }
    }

    @Test("Status text has bold font")
    func statusTextHasBoldFont() {
        let result = StatusFormatter.format(state: .idle)
        let string = result.string

        // Find range of "Idle" text (after indicator)
        if let range = string.range(of: "Idle") {
            let nsRange = NSRange(range, in: string)
            var effectiveRange = NSRange(location: 0, length: 0)
            if let font = result.attribute(.font, at: nsRange.location, effectiveRange: &effectiveRange) as? NSFont {
                // Check if font is bold (weight check)
                let traits = font.fontDescriptor.symbolicTraits
                let isBold = traits.contains(.bold)
                #expect(isBold)
            }
        }
    }

    @Test("Running state endpoint has monospaced font")
    func runningStateEndpointHasMonospacedFont() {
        let result = StatusFormatter.format(state: .running(preset: "test", endpoint: "http://localhost:8080"))
        let string = result.string

        // Find endpoint range
        if let range = string.range(of: "http://localhost:8080") {
            let nsRange = NSRange(range, in: string)
            var effectiveRange = NSRange(location: 0, length: 0)
            if let font = result.attribute(.font, at: nsRange.location, effectiveRange: &effectiveRange) as? NSFont {
                // Check if font family contains "Menlo" or "Monaco" (monospaced)
                let familyName = font.familyName ?? ""
                let isMonospaced = familyName.contains("Menlo") || familyName.contains("Monaco") || familyName.contains("SF Mono")
                #expect(isMonospaced || font.fontDescriptor.symbolicTraits.contains(.monoSpace))
            }
        }
    }

    @Test("Formatted string has proper paragraph style")
    func formattedStringHasParagraphStyle() {
        let result = StatusFormatter.format(state: .idle)

        var effectiveRange = NSRange(location: 0, length: 0)
        if let paragraphStyle = result.attribute(.paragraphStyle, at: 0, effectiveRange: &effectiveRange) as? NSParagraphStyle {
            // Verify no indentation
            #expect(paragraphStyle.headIndent == 0)
            #expect(paragraphStyle.firstLineHeadIndent == 0)
        }
    }

    @Test("Format handles special characters in preset name")
    func formatHandlesSpecialCharacters() {
        let result = StatusFormatter.format(state: .loading(preset: "test-model-@#$%"))
        let string = result.string

        #expect(string.contains("test-model-@#$%"))
    }

    @Test("Format handles empty preset name")
    func formatHandlesEmptyPresetName() {
        let result = StatusFormatter.format(state: .loading(preset: ""))
        let string = result.string

        // Should still format correctly even with empty preset
        #expect(string.contains("Loading..."))
    }

    @Test("Format handles very long preset name")
    func formatHandlesLongPresetName() {
        let longName = String(repeating: "a", count: 100)
        let result = StatusFormatter.format(state: .loading(preset: longName))
        let string = result.string

        // Should contain the full name
        #expect(string.contains(longName))
    }

    @Test("Format handles URL-like endpoint")
    func formatHandlesURLEndpoint() {
        let result = StatusFormatter.format(state: .running(preset: "test", endpoint: "https://example.com:8080/path"))
        let string = result.string

        #expect(string.contains("https://example.com:8080/path"))
    }

    @Test("NotRunning state has no subtitle")
    func notRunningHasNoSubtitle() {
        let result = StatusFormatter.format(state: .notRunning)
        let string = result.string

        // Should only have one line (indicator + status)
        let trimmed = string.trimmingCharacters(in: .whitespacesAndNewlines)
        let lines = trimmed.components(separatedBy: "\n").filter { !$0.isEmpty }
        #expect(lines.count == 1)
    }

    @Test("Idle state has subtitle")
    func idleHasSubtitle() {
        let result = StatusFormatter.format(state: .idle)
        let string = result.string

        // Should have multiple lines
        let trimmed = string.trimmingCharacters(in: .whitespacesAndNewlines)
        let lines = trimmed.components(separatedBy: "\n").filter { !$0.isEmpty }
        #expect(lines.count >= 2)
    }

    @Test("Loading state shows preset name")
    func loadingShowsPresetName() {
        let presetName = "codellama-7b-q4"
        let result = StatusFormatter.format(state: .loading(preset: presetName))
        let string = result.string

        #expect(string.contains(presetName))
    }

    @Test("Running state shows both preset and endpoint")
    func runningShowsBothPresetAndEndpoint() {
        let preset = "my-preset"
        let endpoint = "http://localhost:8080"
        let result = StatusFormatter.format(state: .running(preset: preset, endpoint: endpoint))
        let string = result.string

        #expect(string.contains(preset))
        #expect(string.contains(endpoint))

        // Preset should appear before endpoint
        if let presetRange = string.range(of: preset),
           let endpointRange = string.range(of: endpoint) {
            #expect(presetRange.lowerBound < endpointRange.lowerBound)
        }
    }
}

@Suite("StatusFormatter indicator Tests")
@MainActor
struct StatusFormatterIndicatorTests {
    @Test("Indicator symbols are correct for each state")
    func indicatorSymbolsAreCorrect() {
        let notRunning = StatusFormatter.format(state: .notRunning).string
        let idle = StatusFormatter.format(state: .idle).string
        let loading = StatusFormatter.format(state: .loading(preset: "test")).string
        let running = StatusFormatter.format(state: .running(preset: "test", endpoint: "test")).string

        #expect(notRunning.starts(with: "○"))
        #expect(idle.starts(with: "●"))
        #expect(loading.starts(with: "◐"))
        #expect(running.starts(with: "●"))
    }

    @Test("Idle and running have different colors despite same symbol")
    func idleAndRunningHaveDifferentColors() {
        let idleResult = StatusFormatter.format(state: .idle)
        let runningResult = StatusFormatter.format(state: .running(preset: "test", endpoint: "test"))

        var idleRange = NSRange(location: 0, length: 0)
        let idleColor = idleResult.attribute(.foregroundColor, at: 0, effectiveRange: &idleRange) as? NSColor

        var runningRange = NSRange(location: 0, length: 0)
        let runningColor = runningResult.attribute(.foregroundColor, at: 0, effectiveRange: &runningRange) as? NSColor

        // Idle should be yellow, running should be green
        #expect(idleColor == .systemYellow)
        #expect(runningColor == .systemGreen)
        #expect(idleColor != runningColor)
    }
}
