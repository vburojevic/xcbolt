# ⚡ xcbolt

**A modern Xcode CLI and interactive TUI that humans and AI agents both love.**

![License](https://img.shields.io/badge/license-MIT-blue.svg)
![Go](https://img.shields.io/badge/Go-1.25+-00ADD8?logo=go&logoColor=white)
![Platform](https://img.shields.io/badge/Platform-macOS-lightgrey?logo=apple)
![Homebrew](https://img.shields.io/badge/Homebrew-vburojevic%2Ftap-FBB040?logo=homebrew)

---

## Highlights

- **Interactive TUI** — Card-based dashboard with live build progress, errors, and quick actions
- **CLI Power** — Full `xcodebuild` control with NDJSON output for CI/CD pipelines
- **AI-Friendly** — Structured JSON events perfect for agent orchestration and automation
- **Zero Config** — Auto-detects workspaces, schemes, simulators, and devices
- **Unified Tooling** — Build, test, run, and manage simulators/devices in one tool

---

## Installation

**Homebrew (recommended):**
```bash
brew install vburojevic/tap/xcbolt
```

**From source:**
```bash
git clone https://github.com/xcbolt/xcbolt.git
cd xcbolt
go build -o xcbolt ./cmd/xcbolt
```

---

## Quick Start

```bash
# 1. Navigate to your Xcode project
cd ~/Projects/MyApp

# 2. Initialize config (optional — auto-config works for simple projects)
xcbolt init

# 3. Launch the TUI
xcbolt
```

---

## Interactive TUI

Launch with `xcbolt` or `xcbolt tui`. The interface has three tabs:

| Tab | Description |
|-----|-------------|
| **Stream** | Real-time build output with search and filtering |
| **Issues** | Errors and warnings extracted for quick navigation |
| **Summary** | Card-based dashboard with project info and build status |

### Keybindings

**Actions:**
| Key | Action | Key | Action |
|-----|--------|-----|--------|
| `b` | Build | `c` | Clean |
| `r` | Run | `x` | Stop app |
| `t` | Test | `esc` | Cancel |

**Navigation:**
| Key | Action | Key | Action |
|-----|--------|-----|--------|
| `1` | Stream tab | `s` | Select scheme |
| `2` | Issues tab | `d` | Select destination |
| `3` | Summary tab | `Ctrl+K` | Command palette |
| `tab` | Next tab | `i` | Init wizard |

**Search & View:**
| Key | Action | Key | Action |
|-----|--------|-----|--------|
| `/` | Search logs | `v` | Toggle view mode |
| `n` / `N` | Next/prev error | `e` / `E` | Expand/collapse all |
| `o` | Open in Xcode | `y` | Copy line |
| `?` | Help | `q` | Quit |

**Scrolling:** `j`/`k`, arrows, `PgUp`/`PgDn`, `Ctrl+U`/`Ctrl+D`, `g`/`G` (top/bottom)

---

## CLI Commands

### Build Commands

| Command | Description |
|---------|-------------|
| `xcbolt build` | Build the configured scheme |
| `xcbolt test` | Run tests |
| `xcbolt run` | Build, install, and launch on simulator/device |
| `xcbolt clean` | Clean derived data |

### Info & Setup

| Command | Description |
|---------|-------------|
| `xcbolt init` | Interactive setup wizard |
| `xcbolt context` | Show project context (schemes, destinations) |
| `xcbolt doctor` | Validate Xcode environment |

### Simulator Management

| Command | Description |
|---------|-------------|
| `xcbolt simulator list` | List available simulators |
| `xcbolt simulator boot <udid>` | Boot a simulator |
| `xcbolt simulator shutdown <udid>` | Shutdown a simulator |

### Device Management

| Command | Description |
|---------|-------------|
| `xcbolt device list` | List connected physical devices |

### Other

| Command | Description |
|---------|-------------|
| `xcbolt logs` | Stream simulator/device logs |
| `xcbolt apps` | List installed apps |
| `xcbolt stop <bundle-id>` | Stop a running app |

### Examples

```bash
# Build with scheme override
xcbolt build --scheme MyScheme --configuration Release

# Run with console output
xcbolt run --console

# Stream logs with predicate filter
xcbolt logs --predicate 'process == "MyApp"'

# List tests without running
xcbolt test --list
```

---

## CI/CD Integration

All commands support NDJSON event streaming with the `--json` flag:

```bash
xcbolt --json build
xcbolt --json test | jq '.type'
```

**Event types emitted:**

| Type | Description |
|------|-------------|
| `log` | Build output line |
| `error` | Error message |
| `warning` | Warning message |
| `result` | Operation result with data |
| `status` | Status update |

---

## Configuration

Running `xcbolt init` creates `.xcbolt/config.json`:

```json
{
  "version": 1,
  "workspace": "MyApp.xcworkspace",
  "scheme": "MyApp",
  "configuration": "Debug",
  "destination": {
    "kind": "simulator",
    "udid": "...",
    "name": "iPhone 16 Pro"
  },
  "xcodebuild": {
    "logFormat": "auto",
    "logFormatArgs": []
  }
}
```

| Field | Description |
|-------|-------------|
| `workspace` / `project` | Path to `.xcworkspace` or `.xcodeproj` |
| `scheme` | Build scheme name |
| `configuration` | Build configuration (`Debug` / `Release`) |
| `destination` | Target simulator or device |
| `xcodebuild.logFormat` | Log formatter: `auto`, `xcpretty`, `xcbeautify`, `raw` |

---

## AI Agent Context

For AI agents, coding assistants, and automation tools — structured context for working with xcbolt.

### Project Metadata

```yaml
name: xcbolt
type: CLI + TUI
language: Go 1.25+
frameworks:
  - Cobra (CLI)
  - Bubble Tea (TUI)
  - Lip Gloss (styling)
config_file: .xcbolt/config.json
output_format: NDJSON (--json flag)
```

### File Structure

```
xcbolt/
├── cmd/xcbolt/main.go      # Entrypoint
├── internal/
│   ├── cli/                # Cobra command definitions
│   ├── tui/                # Bubble Tea TUI components
│   ├── core/               # Xcode tooling wrappers
│   └── util/               # Shared utilities
└── .xcbolt/
    ├── config.json         # Project configuration
    ├── DerivedData/        # Build artifacts
    └── Results/            # Test result bundles
```

### Key Patterns for Agents

**Building:**
```bash
xcbolt --json build          # Structured output
# Parse NDJSON: {"type": "log"|"error"|"result", "msg": "..."}
# Exit code: 0 = success, non-zero = failure
```

**Testing:**
```bash
xcbolt test --list           # Enumerate available tests
xcbolt --json test           # Run with structured output
xcbolt test --only "Tests/MyTest/testMethod"  # Filter tests
```

**Environment validation:**
```bash
xcbolt doctor                # Check Xcode toolchain
xcbolt context               # Current project state
```

**Simulator control:**
```bash
xcbolt simulator list        # JSON array of simulators
xcbolt simulator boot <udid>
xcbolt simulator shutdown <udid>
```

### Event Schema

```typescript
interface Event {
  type: "log" | "error" | "warning" | "result" | "status";
  msg: string;
  err?: string;    // Present on error events
  data?: object;   // Present on result events
}
```

---

## Development

```bash
make build   # Build binary
make test    # Run tests
make run     # Build and launch
```

See [AGENTS.md](./AGENTS.md) for contribution guidelines, coding standards, and architecture documentation.

---

## Requirements

- **macOS** with Xcode Command Line Tools
- **Go 1.25+** (building from source only)
- **Xcode 15+** recommended for full feature support

---

## License

MIT — see [LICENSE](./LICENSE) for details.
