# xcbolt (Go)

A modern Xcode CLI + TUI built with:

- **Cobra** for commands
- **Bubble Tea** for the interactive TUI
- **Bubbles** (viewport, spinner, help)
- **Lip Gloss** for styling
- **Huh** for the init wizard
- **Harmonica** for toast animation
- `howett.net/plist` for parsing `Info.plist`

This project is intentionally **self-contained** and uses Apple tooling via `xcrun`:

- `xcodebuild` (build/test/list schemes)
- `simctl` (simulators)
- `devicectl` (physical devices)
- `xcresulttool` (test summaries)

> It’s designed to be a solid starting point you can extend.

---

## Installation (Homebrew)

```bash
brew tap vburojevic/tap
brew install xcbolt
```

Or in one line:

```bash
brew install vburojevic/tap/xcbolt
```

---

## Quick start

```bash
cd xcbolt-go
go mod tidy
go build ./cmd/xcbolt
./xcbolt --help
```

### Initialize a project

From your Xcode project root:

```bash
./xcbolt init
```

This writes `.xcbolt/config.json`.

### Use the TUI

```bash
./xcbolt
# or
./xcbolt tui
```

Keybinds (shown in-app with `?`):

- `b` build
- `r` run
- `t` test
- `l` logs tab
- `i` init wizard
- `c` refresh context
- `esc` cancel running op
- `tab` / `shift+tab` switch tabs
- `q` quit

---

## CLI commands

```bash
./xcbolt context
./xcbolt doctor
./xcbolt build
./xcbolt test --list
./xcbolt run --console
./xcbolt logs --predicate 'process == "MyApp"'
./xcbolt simulator list
./xcbolt device list
./xcbolt apps
./xcbolt stop <bundle-id>
```

### JSON output (NDJSON)

All commands support:

```bash
./xcbolt --json build
```

This outputs newline-delimited JSON events (easy to pipe into other tools).

---

## Project layout

- `cmd/xcbolt` — entrypoint
- `internal/cli` — Cobra commands
- `internal/tui` — Bubble Tea UI
- `internal/core` — Xcode/simctl/devicectl wrappers, config, operations
- `internal/util` — small utilities

---

## Notes / next ideas

- Add smarter test selection UI (targets → test classes → test methods)
- Add result bundle browsing (xcresult object graphs)
- Add device log streaming via `devicectl` where supported
- Add `--destination` parsing like `xcodebuild -destination`
- Add concurrency limits + log filtering in TUI
