package tui

import (
	"regexp"
	"strings"
)

var (
	fileErrorRE = regexp.MustCompile(`(?i)^[^:\s].*:\d+(?::\d+)?:\s*(fatal error|error):`)
)

func issueSeverity(line string) TabLineType {
	lower := strings.ToLower(line)

	if strings.Contains(lower, "warning:") || strings.Contains(line, "⚠") {
		return TabLineTypeWarning
	}
	if strings.Contains(lower, "note:") || strings.Contains(lower, "remark:") {
		return TabLineTypeNote
	}

	if isFatalBuildErrorLine(lower, line) {
		return TabLineTypeError
	}

	if !isErrorishLine(lower, line) {
		return TabLineTypeNormal
	}

	if fileErrorRE.MatchString(line) {
		return TabLineTypeError
	}
	return TabLineTypeWarning
}

func isErrorishLine(lower, line string) bool {
	return strings.Contains(lower, "error:") ||
		strings.Contains(lower, "fatal error") ||
		strings.Contains(lower, "❌") ||
		strings.Contains(line, "✗")
}

func isFatalBuildErrorLine(lower, line string) bool {
	switch {
	case strings.Contains(lower, "fatal error"):
		return true
	case strings.Contains(lower, "clang: error"):
		return true
	case strings.Contains(lower, "swiftc: error"):
		return true
	case strings.Contains(lower, "swift-frontend: error"):
		return true
	case strings.Contains(lower, "ld: error"):
		return true
	case strings.Contains(lower, "linker command failed"):
		return true
	case strings.Contains(lower, "command swiftcompile failed"):
		return true
	case strings.Contains(lower, "command compilec failed"):
		return true
	case strings.Contains(lower, "command link failed"):
		return true
	case strings.Contains(lower, "codesign error"):
		return true
	case strings.Contains(lower, "code signing") && strings.Contains(lower, "error"):
		return true
	case strings.Contains(lower, "provisioning profile") && strings.Contains(lower, "error"):
		return true
	case strings.Contains(lower, "no such module"):
		return true
	case strings.Contains(lower, "undefined symbols"):
		return true
	case strings.Contains(lower, "symbol(s) not found"):
		return true
	case strings.Contains(lower, "framework not found"):
		return true
	case strings.Contains(lower, "library not found"):
		return true
	case strings.Contains(lower, "failed with exit code"):
		return true
	case strings.Contains(lower, "build failed"):
		return true
	case strings.Contains(lower, "xcodebuild: error"):
		return true
	case strings.Contains(lower, "error: unable to find a destination"):
		return true
	case strings.Contains(lower, "error: no profiles"):
		return true
	case strings.Contains(lower, "error: exit status"):
		return true
	}

	return false
}
