package core

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"
)

type EnumeratedTests struct {
	Raw any `json:"raw"`
}

func XcodebuildEnumerateTests(ctx context.Context, projectRoot string, cfg Config) (EnumeratedTests, error) {
	args := []string{"xcodebuild", "-enumerate-tests", "-json"}
	if cfg.Workspace != "" {
		args = append(args, "-workspace", filepath.Join(projectRoot, cfg.Workspace))
	} else if cfg.Project != "" {
		args = append(args, "-project", filepath.Join(projectRoot, cfg.Project))
	}
	if cfg.Scheme != "" {
		args = append(args, "-scheme", cfg.Scheme)
	}
	if cfg.Configuration != "" {
		args = append(args, "-configuration", cfg.Configuration)
	}
	if dest := BuildDestinationString(cfg); dest != "" {
		args = append(args, "-destination", dest)
	}

	var out strings.Builder
	_, err := RunStreaming(ctx, CmdSpec{
		Path: "xcrun",
		Args: args,
		Dir:  projectRoot,
		StdoutLine: func(s string) {
			out.WriteString(s)
			out.WriteString("\n")
		},
	})
	if err != nil {
		return EnumeratedTests{}, err
	}
	trim := extractJSONObject(out.String())
	if trim == "" {
		trim = out.String()
	}
	var anyJSON any
	if err := json.Unmarshal([]byte(trim), &anyJSON); err != nil {
		// Return raw string if JSON parsing fails.
		return EnumeratedTests{Raw: map[string]any{"text": out.String()}}, nil
	}
	return EnumeratedTests{Raw: anyJSON}, nil
}

func PrettyEnumeratedTests(et EnumeratedTests) string {
	if et.Raw == nil {
		return "(no tests)"
	}
	b, err := json.MarshalIndent(et.Raw, "", "  ")
	if err != nil {
		return fmt.Sprintf("%v", et.Raw)
	}
	return string(b)
}
