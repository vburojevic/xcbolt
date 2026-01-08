package tui

import "github.com/charmbracelet/lipgloss"

// =============================================================================
// Pastel Color Palettes - Dark and Light modes
// =============================================================================

// PastelColorsDark returns the dark mode pastel palette (Tokyo Night inspired)
func PastelColorsDark() Colors {
	return Colors{
		// Accent - soft pastel blue
		Accent:      lipgloss.AdaptiveColor{Light: "#5A7BC0", Dark: "#7AA2F7"},
		AccentMuted: lipgloss.AdaptiveColor{Light: "#7B96D3", Dark: "#3D59A1"},

		// Semantic colors - soft pastels
		Success: lipgloss.AdaptiveColor{Light: "#5B8A3A", Dark: "#9ECE6A"}, // Soft green
		Warning: lipgloss.AdaptiveColor{Light: "#C48F2C", Dark: "#E0AF68"}, // Warm amber
		Error:   lipgloss.AdaptiveColor{Light: "#C74B5C", Dark: "#F7768E"}, // Soft coral
		Running: lipgloss.AdaptiveColor{Light: "#8B6AB0", Dark: "#BB9AF7"}, // Soft purple

		// Backgrounds - deep navy
		Background: lipgloss.AdaptiveColor{Light: "#FFFBF5", Dark: "#1A1B26"},
		Surface:    lipgloss.AdaptiveColor{Light: "#F5F0E8", Dark: "#24283B"},
		Overlay:    lipgloss.AdaptiveColor{Light: "#EDE8E0", Dark: "#414868"},

		// Text - clear hierarchy
		Text:       lipgloss.AdaptiveColor{Light: "#383A42", Dark: "#C0CAF5"},
		TextMuted:  lipgloss.AdaptiveColor{Light: "#6C6E7A", Dark: "#9AA5CE"},
		TextSubtle: lipgloss.AdaptiveColor{Light: "#9DA0AB", Dark: "#565F89"},

		// Borders
		Border:      lipgloss.AdaptiveColor{Light: "#D5D1C9", Dark: "#3B4261"},
		BorderMuted: lipgloss.AdaptiveColor{Light: "#E8E4DC", Dark: "#292E42"},
	}
}

// PastelColorsLight returns the light mode pastel palette (Rosé Pine Dawn inspired)
func PastelColorsLight() Colors {
	return Colors{
		// Accent - muted cobalt
		Accent:      lipgloss.AdaptiveColor{Light: "#5A7BC0", Dark: "#7AA2F7"},
		AccentMuted: lipgloss.AdaptiveColor{Light: "#7B96D3", Dark: "#3D59A1"},

		// Semantic colors - muted pastels
		Success: lipgloss.AdaptiveColor{Light: "#5B8A3A", Dark: "#9ECE6A"},
		Warning: lipgloss.AdaptiveColor{Light: "#C48F2C", Dark: "#E0AF68"},
		Error:   lipgloss.AdaptiveColor{Light: "#C74B5C", Dark: "#F7768E"},
		Running: lipgloss.AdaptiveColor{Light: "#8B6AB0", Dark: "#BB9AF7"},

		// Backgrounds - warm cream
		Background: lipgloss.AdaptiveColor{Light: "#FFFBF5", Dark: "#1A1B26"},
		Surface:    lipgloss.AdaptiveColor{Light: "#F5F0E8", Dark: "#24283B"},
		Overlay:    lipgloss.AdaptiveColor{Light: "#EDE8E0", Dark: "#414868"},

		// Text
		Text:       lipgloss.AdaptiveColor{Light: "#383A42", Dark: "#C0CAF5"},
		TextMuted:  lipgloss.AdaptiveColor{Light: "#6C6E7A", Dark: "#9AA5CE"},
		TextSubtle: lipgloss.AdaptiveColor{Light: "#9DA0AB", Dark: "#565F89"},

		// Borders
		Border:      lipgloss.AdaptiveColor{Light: "#D5D1C9", Dark: "#3B4261"},
		BorderMuted: lipgloss.AdaptiveColor{Light: "#E8E4DC", Dark: "#292E42"},
	}
}

// =============================================================================
// Syntax Highlighting Colors
// =============================================================================

// SyntaxColors holds colors for log syntax highlighting
type SyntaxColors struct {
	Keyword    lipgloss.AdaptiveColor // Swift keywords, modifiers
	String     lipgloss.AdaptiveColor // String literals
	Number     lipgloss.AdaptiveColor // Numeric values
	Type       lipgloss.AdaptiveColor // Type names, classes
	Function   lipgloss.AdaptiveColor // Function names
	Comment    lipgloss.AdaptiveColor // Comments, muted info
	FilePath   lipgloss.AdaptiveColor // File paths in errors
	LineNumber lipgloss.AdaptiveColor // Line number gutter
	Timestamp  lipgloss.AdaptiveColor // Log timestamps
	Verbose    lipgloss.AdaptiveColor // Dimmed verbose output
}

// DefaultSyntaxColors returns syntax highlighting colors
func DefaultSyntaxColors() SyntaxColors {
	return SyntaxColors{
		Keyword:    lipgloss.AdaptiveColor{Light: "#8B6AB0", Dark: "#BB9AF7"}, // Purple
		String:     lipgloss.AdaptiveColor{Light: "#5B8A3A", Dark: "#9ECE6A"}, // Green
		Number:     lipgloss.AdaptiveColor{Light: "#D06D38", Dark: "#FF9E64"}, // Orange
		Type:       lipgloss.AdaptiveColor{Light: "#5A7BC0", Dark: "#7AA2F7"}, // Blue
		Function:   lipgloss.AdaptiveColor{Light: "#4A9AC0", Dark: "#7DCFFF"}, // Cyan
		Comment:    lipgloss.AdaptiveColor{Light: "#9DA0AB", Dark: "#565F89"}, // Gray
		FilePath:   lipgloss.AdaptiveColor{Light: "#D06D38", Dark: "#FF9E64"}, // Orange
		LineNumber: lipgloss.AdaptiveColor{Light: "#9DA0AB", Dark: "#565F89"}, // Gray
		Timestamp:  lipgloss.AdaptiveColor{Light: "#6C6E7A", Dark: "#9AA5CE"}, // Muted
		Verbose:    lipgloss.AdaptiveColor{Light: "#9DA0AB", Dark: "#565F89"}, // Dimmed
	}
}

// =============================================================================
// Pastel Colors - Single unified palette (uses AdaptiveColor for auto dark/light)
// =============================================================================

// PastelColors returns the unified pastel color palette
// Uses AdaptiveColor so it automatically adapts to terminal dark/light mode
func PastelColors() Colors {
	return Colors{
		// Accent - soft pastel blue
		Accent:      lipgloss.AdaptiveColor{Light: "#5A7BC0", Dark: "#7AA2F7"},
		AccentMuted: lipgloss.AdaptiveColor{Light: "#7B96D3", Dark: "#3D59A1"},

		// Semantic colors
		Success: lipgloss.AdaptiveColor{Light: "#5B8A3A", Dark: "#9ECE6A"},
		Warning: lipgloss.AdaptiveColor{Light: "#C48F2C", Dark: "#E0AF68"},
		Error:   lipgloss.AdaptiveColor{Light: "#C74B5C", Dark: "#F7768E"},
		Running: lipgloss.AdaptiveColor{Light: "#8B6AB0", Dark: "#BB9AF7"},

		// Backgrounds
		Background: lipgloss.AdaptiveColor{Light: "#FFFBF5", Dark: "#1A1B26"},
		Surface:    lipgloss.AdaptiveColor{Light: "#F5F0E8", Dark: "#24283B"},
		Overlay:    lipgloss.AdaptiveColor{Light: "#EDE8E0", Dark: "#414868"},

		// Text
		Text:       lipgloss.AdaptiveColor{Light: "#383A42", Dark: "#C0CAF5"},
		TextMuted:  lipgloss.AdaptiveColor{Light: "#6C6E7A", Dark: "#9AA5CE"},
		TextSubtle: lipgloss.AdaptiveColor{Light: "#9DA0AB", Dark: "#565F89"},

		// Borders
		Border:      lipgloss.AdaptiveColor{Light: "#D5D1C9", Dark: "#3B4261"},
		BorderMuted: lipgloss.AdaptiveColor{Light: "#E8E4DC", Dark: "#292E42"},
	}
}
