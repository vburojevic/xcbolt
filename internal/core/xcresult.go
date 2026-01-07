package core

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

type TestSummary struct {
	Raw any `json:"raw"`
}

// XcresultTestSummary attempts to extract a structured test summary from an .xcresult bundle.
func XcresultTestSummary(ctx context.Context, resultBundlePath string) (TestSummary, error) {
	if resultBundlePath == "" {
		return TestSummary{}, errors.New("missing result bundle path")
	}

	candidates := [][]string{
		// Modern
		{"xcresulttool", "get", "test-results", "summary", "--path", resultBundlePath, "--format", "json"},
		// Some versions accept no --format
		{"xcresulttool", "get", "test-results", "summary", "--path", resultBundlePath},
		// Fallback: dump top-level
		{"xcresulttool", "get", "--path", resultBundlePath, "--format", "json"},
		{"xcresulttool", "get", "--path", resultBundlePath},
	}

	var lastErr error
	for _, args := range candidates {
		var out strings.Builder
		_, err := RunStreaming(ctx, CmdSpec{
			Path: "xcrun",
			Args: args,
			StdoutLine: func(s string) {
				out.WriteString(s)
				out.WriteString("\n")
			},
		})
		if err != nil {
			lastErr = err
			continue
		}
		trim := extractJSONObject(out.String())
		if trim == "" {
			trim = out.String()
		}
		var anyJSON any
		if err := json.Unmarshal([]byte(trim), &anyJSON); err != nil {
			lastErr = fmt.Errorf("xcresult json parse: %w", err)
			continue
		}
		return TestSummary{Raw: anyJSON}, nil
	}
	if lastErr == nil {
		lastErr = errors.New("failed to read xcresult")
	}
	return TestSummary{}, lastErr
}
