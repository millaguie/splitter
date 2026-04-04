# AGENTS.md - SPLITTER Project

## Project Overview

SPLITTER is a Go-based tool that creates and manages multiple Tor network instances
load-balanced via HAProxy, with geolocation-based anti-correlation rules for Tor
entry/exit node selection.

Two proxy modes are supported:
- **Native** (default): Uses Tor's built-in `HTTPTunnelPort` — no Privoxy dependency.
  TCP path: User -> HAProxy -> Tor (per-instance) -> Tor Network -> Destination.
- **Legacy**: Uses Privoxy as the HTTP-to-SOCKS bridge.
  TCP path: User -> HAProxy -> Privoxy (per-instance) -> Tor (per-instance) -> Tor Network -> Destination.

Modern Tor features are auto-detected and enabled when available: Conflux (multi-leg circuits),
congestion control, CGO relay cryptography, post-quantum key exchange, and bridge/pluggable
transport support (Snowflake, WebTunnel, obfs4).

## Build / Run / Test Commands

### Build

```bash
go build -o splitter .
```

### Run

```bash
# Run with defaults
./splitter run

# Run with flags
./splitter run --instances 3 --countries 8 --relay-enforce exit
# Short flags (legacy aliases): -i, -c, -re
./splitter run -i 3 -c 8 -re exit

# Run with a profile
./splitter run --profile stealth

# Other subcommands
./splitter status
./splitter test dns
./splitter version

# Relay enforce modes: entry (default, best security), exit (GeoIP bypass), speed (fastest)
# Proxy modes: native (default), legacy (Privoxy)
```

### Test

```bash
# Run all tests
go test ./...

# Run tests for a specific package
go test ./internal/config/...

# Run with verbose output
go test -v ./...

# Run integration tests (requires tor, haproxy installed)
go test -tags=integration ./...
```

### Lint

```bash
# Go vet
go vet ./...

# golangci-lint (if installed)
golangci-lint run
```

### Docker

```bash
docker build -t splitter .
docker run -d --name splitter -p 63536:63536 -p 63537:63537 splitter

# Or with docker-compose
docker compose up -d
```

## Project Structure

```
main.go                              # Entry point
go.mod / go.sum                      # Go module definition
cmd/
  root.go                            # Root Cobra command
  run.go                             # `splitter run` subcommand
  status.go                          # `splitter status` - live dashboard
  test.go                            # `splitter test dns` / `splitter test exit-reputation`
  version.go                         # `splitter version` - detected Tor features
  reload.go                          # SIGHUP config reload handler
internal/
  cli/                               # Cobra setup, flag bindings, input validation
  config/                            # Config loading: YAML + env vars (SPLITTER_*) + CLI flags
  tor/                               # Tor instance lifecycle: spawn, config gen, signal, restart
  haproxy/                           # HAProxy config generation, process management
  proxy/                             # Proxy abstraction: HTTPTunnelPort (native) or Privoxy (legacy)
  country/                           # Country selection, rotation daemon, Tor Metrics API client
  circuit/                           # Circuit renewal, NEWNYM via Tor control protocol (cookie auth)
  process/                           # Process group lifecycle: spawn, graceful shutdown, SIGTERM->SIGKILL
  metrics/                           # Prometheus metrics endpoint
  health/                            # Health checks, DNS leak tests, exit node reputation
  network/                           # Port allocation via net.Listen (no netstat)
  profile/                           # Predefined profiles: stealth, balanced, streaming, pentest
  template/                          # Go template helpers for config generation
templates/
  torrc.gotmpl                       # Tor config template
  haproxy.cfg.gotmpl                 # HAProxy config template
  privoxy.cfg.gotmpl                 # Privoxy config template (legacy mode only)
configs/
  default.yaml                       # Default configuration (replaces settings.cfg)
  bridges.yaml                       # Bridge configuration (Snowflake, obfs4, WebTunnel)
  profiles.yaml                      # Profile definitions (stealth, balanced, streaming, pentest)
  useragents.yaml                    # Bundled Tor Browser User-Agent list
legacy/                              # Original Bash version preserved for reference
  splitter.sh
  func/
  settings.cfg
Dockerfile                           # Multi-stage: golang build -> alpine runtime
docker-compose.yml                   # Service definition with healthcheck
```

## Code Style Guidelines

### Go Version and Modules

- Go 1.23+ (module: `github.com/user/splitter`).
- Use Go modules exclusively; no `GOPATH` mode.
- Dependencies: `cobra` (CLI), `gopkg.in/yaml.v3` (config). Avoid adding new dependencies without justification.

### Package Naming

- Package names: lowercase, single word, no underscores (e.g., `tor`, `haproxy`, `network`).
- Package names match their directory name.
- No `util` or `helpers` packages — put functionality in domain-specific packages.

### Error Handling

- Wrap errors with context: `fmt.Errorf("functionName: %w", err)`.
- Always check errors; never silently discard them with `_`.
- Return errors up the call stack; handle at the appropriate level.
- Use `errors.Is()` and `errors.As()` for error inspection.

### Context

- `context.Context` is the first parameter in all I/O and long-running functions.
- Use `signal.NotifyContext` for graceful shutdown.
- Goroutines must accept a context or done channel for cancellation.

### Logging

- Use `log/slog` structured logging.
- **Logging is OFF by default** (philosophy: no logs, no crime).
- Enable via `--log` flag or `SPLITTER_LOG=1` env var.
- JSON format for Docker (detected via `TERM=dumb` or `NO_COLOR`), text for terminal.
- `--log-level` controls verbosity (default INFO when logs are on).
- Never log sensitive data (IPs, circuit paths, authentication tokens).

### Naming Conventions

- **Exported functions/types**: PascalCase: `NewManager`, `TorInstance`.
- **Unexported functions/types**: camelCase: `findAvailablePort`, `writeConfig`.
- **Constants**: PascalCase (exported) or camelCase (unexported), not UPPER_SNAKE_CASE.
- **Acronyms**: `HTTP`, `URL`, `ID`, `TLS` — e.g., `HTTPTunnelPort`, `TLSEnabled`.
- **Interfaces**: named by behavior (`Runner`, `ConfigProvider`), not `Impl` or `Interface`.

### Code Organization

- No `init()` functions. No global mutable state.
- Prefer small, focused files. One primary type per file where practical.
- Interfaces are defined where they are consumed, not where they are implemented.
- Group related types and functions together within a package.

### Configuration Priority

1. CLI flags (highest priority)
2. Environment variables (`SPLITTER_*` prefix)
3. YAML config file (`configs/default.yaml`)
4. Compiled-in defaults (lowest priority)

### Concurrency

- Use goroutines for Tor instances, circuit renewal, country rotation.
- Every goroutine must have a cancellation mechanism (context or done channel).
- Use `sync.Mutex` or channels for shared state; no data races.
- Process management uses `os/exec` with context cancellation.

## Testing Conventions

### Test Structure

- Table-driven tests using `t.Run(name, func(t *testing.T))`.
- Test files live in the same package (`*_test.go`).
- Integration tests behind `//go:build integration` build tag.
- Use `t.TempDir()` for temporary files and directories in tests.
- Use `t.Setenv()` for environment variable tests.

### Test Patterns

```go
func TestFunctionName(t *testing.T) {
    tests := []struct {
        name    string
        input   InputType
        want    OutputType
        wantErr bool
    }{
        {name: "valid input", input: InputType{...}, want: OutputType{...}},
        {name: "invalid input", input: InputType{...}, wantErr: true},
    }
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            got, err := FunctionName(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("FunctionName() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if !reflect.DeepEqual(got, tt.want) {
                t.Errorf("FunctionName() = %v, want %v", got, tt.want)
            }
        })
    }
}
```

### Test Execution

- Run `go test ./...` before committing.
- Run `go vet ./...` before committing.
- Mock external dependencies (Tor binary, HAProxy, network) in unit tests.
- Integration tests require real tor/haproxy binaries and are skipped in CI.

## Key Patterns to Follow

1. **Never hardcode ports or paths** — use values from `configs/default.yaml`.
2. **Use `text/template` for config generation** — torrc, haproxy.cfg, privoxy.cfg all use Go templates in `templates/`.
3. **Use `net.Listen` for port availability checks** — no `netstat`, `ss`, or shell calls.
4. **Graceful shutdown** — `SIGTERM` -> wait 5s -> `SIGKILL` via `signal.NotifyContext`. Kill all child processes on exit.
5. **No logs by default** — enable via `--log` or `SPLITTER_LOG=1`. Never log sensitive data.
6. **Cookie auth for Tor control port** — `CookieAuthentication 1` in torrc, read `control_auth_cookie` file. No `expect` scripts.
7. **Kill previous instances before starting new ones** — clean up stale tor/haproxy/privoxy processes at startup.
8. **Randomize order** — use `math/rand` to shuffle backend lists and country selection for anti-correlation.
9. **Auto-detect Tor features** — parse `tor --version` at startup and conditionally enable Conflux, CGO, congestion control, HTTPTunnelPort.
10. **Config reload via SIGHUP** — reload `configs/default.yaml` without full restart. Restart Tor instances only if torrc parameters changed.

---

## Ralph Loop Workflow

This project uses the **Ralph Loop** methodology for autonomous AI-driven development.
Each iteration starts with fresh context, reads progress from files, implements one task,
tests, commits, and updates status.

### Files

| File | Purpose |
|------|---------|
| `RALPH_STATUS.md` | Tracks current progress across ROADMAP phases (DO NOT DELETE) |
| `ROADMAP.md` | Full development plan with ordered tasks |
| `.opencode/commands/ralph.md` | The `/ralph` command template |
| `.opencode/agents/ralph.md` | The Ralph worker agent definition |

### Usage

1. Run `/ralph` in OpenCode
2. The agent picks the next ROADMAP task, implements it, runs tests, commits
3. After each iteration, run `/ralph` again to continue
4. Each iteration starts with fresh context -- no context pollution

### Rules for Agents

- ONE task per `/ralph` iteration
- Always run `go test` before committing
- Follow ROADMAP task numbering strictly
- Update `RALPH_STATUS.md` after each iteration
- Use `@reviewer` subagent before committing complex changes
- Use `@security` subagent before committing Tor/network code

---

## OpenCode Agent Configuration

This project has 6 configured agents in `opencode.json`:

| Agent | Role | Model | Type |
|-------|------|-------|------|
| **build** | Development, writes code | GitHub Copilot GPT-5 Mini | primary |
| **plan** | Architecture, analysis, planning | deepseek/deepseek-chat | primary |
| **ralph** | Autonomous loop worker | deepseek/deepseek-chat | primary |
| **reviewer** | Code review (read-only) | GitHub Copilot GPT-5 Mini | subagent |
| **explorer** | Codebase search (read-only) | GitHub Copilot GPT-5 Mini | subagent |
| **security** | Tor/network security audit (read-only) | deepseek/deepseek-chat | subagent |

### Switching Agents

- Use **Tab** key to cycle between primary agents (build, plan, ralph)
- Use `@agent-name` to invoke subagents (e.g., `@reviewer check this code`)

### Commands

- `/ralph` - Start an autonomous Ralph Loop iteration
