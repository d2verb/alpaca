import AppKit
import Foundation

/// Formats daemon state as styled attributed strings for menu display.
@MainActor
struct StatusFormatter {
    /// Creates a formatted attributed string displaying the current daemon state.
    static func format(state: DaemonState, errorMessage: String? = nil) -> NSAttributedString {
        let result = NSMutableAttributedString()
        let paragraphStyle = Self.paragraphStyle()

        // Status indicator and text
        let (indicator, color) = Self.indicator(for: state)
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
            string: state.statusText,
            attributes: [
                .font: NSFont.boldSystemFont(ofSize: 13),
                .foregroundColor: NSColor.labelColor,
                .paragraphStyle: paragraphStyle
            ]
        )
        result.append(statusText)

        // Additional info based on state
        switch state {
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
            // Add spacing using a small font line
            result.append(NSAttributedString(
                string: "\n \n",
                attributes: [.font: NSFont.systemFont(ofSize: 4), .paragraphStyle: paragraphStyle]
            ))
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

    /// Returns the status indicator symbol and color for a given state.
    private static func indicator(for state: DaemonState) -> (symbol: String, color: NSColor) {
        switch state {
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

    /// Returns a paragraph style with no indentation.
    private static func paragraphStyle() -> NSParagraphStyle {
        let style = NSMutableParagraphStyle()
        style.headIndent = 0
        style.firstLineHeadIndent = 0
        return style
    }
}
