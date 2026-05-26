# SPLITTER Modernization Roadmap

## Overview

Full rewrite of SPLITTER in Go, replacing the Bash codebase. The new version retains the same
architecture (Tor instances per country, HAProxy load balancing, geo-based anti-correlation rules)
but benefits from Go's concurrency model, single static binary distribution, and robust process
management. The Bash version is kept in `legacy/` for reference.

Existing short flags (`-i`, `-c`, `-re`) will remain as aliases alongside new long flags.

---

## Phase 1: Project Infrastructure ✅ DONE

| # | Task | Detail | Status |
|---|------|--------|--------|
| 1.1 | **Initialize Go module** | `go mod init github.com/user/splitter`. Add `go.sum`. | ✅ |
| 1.2 | **Add `.gitignore`** | Ignore `*.log`, `*.pid`, `/tmp/splitter/`, `force_new_circuit.sh`, Go build artifacts (`/bin/`, `*.exe`). | ✅ |
| 1.3 | **Move Bash to `legacy/`** | Move `splitter.sh`, `func/`, `settings.cfg` into `legacy/` directory. Update Dockerfile references. | ✅ |
| 1.6 | **Map `settings.cfg` → `configs/default.yaml`** | Audit all 541+ lines of `settings.cfg` and produce an explicit mapping of every parameter to its Go config equivalent. Parameters with no Go equivalent must be either ported or explicitly dropped with justification. This document drives 3.1.2. | ✅ `configs/SETTINGS_MAP.md` |
| 1.4 | **Add `docker-compose.yml`** | `splitter` service with volumes for config, healthcheck, restart policy, environment variables. | ✅ |
| 1.5 | **Add `.github/workflows/ci.yml`** | Pipeline: `go vet` + `go test` + `golangci-lint` + Docker build on PRs and pushes. | ✅ |

---

## Phase 2: Go Project Structure ✅ DONE

```
splitter/
  main.go                           # Entry point
  go.mod
  go.sum
  Makefile                          # Build, test, docker, smoke targets
  cmd/
    root.go                         # Root command (Cobra)
    run.go                          # `splitter run` subcommand
    status.go                       # `splitter status` - live dashboard
    test.go                         # `splitter test dns` / `splitter test exit-reputation`
    version.go                      # `splitter version` - detected Tor features
    reload.go                       # SIGHUP config reload handler
  internal/
    cli/                            # Cobra setup, flag bindings, input validation
    config/                         # Config loading: file, env vars (SPLITTER_*), defaults, profiles
    tor/                            # Tor instance lifecycle: spawn, config generation, signal, restart
    haproxy/                        # HAProxy config generation, process management
    proxy/                          # Proxy abstraction: HTTPTunnelPort (native) or Privoxy (legacy)
    country/                        # Country selection, rotation daemon, Tor Metrics API client
    circuit/                        # Circuit renewal, Conflux management, NEWNYM via control port
    process/                        # Process group lifecycle: spawn, graceful shutdown, SIGTERM->SIGKILL
    metrics/                        # Prometheus metrics endpoint
    health/                         # Health checks, DNS leak tests, exit node reputation
    network/                        # Port availability (net.Listen test), IPv4/IPv6 detection
    profile/                        # Predefined profiles: stealth, balanced, streaming, pentest
    template/                       # Go templates for torrc, haproxy.cfg, privoxy.cfg
  templates/
    torrc.gotmpl                    # Tor config template
    haproxy.cfg.gotmpl              # HAProxy config template
    privoxy.cfg.gotmpl              # Privoxy config template (legacy mode)
  configs/
    default.yaml                    # Default configuration (replaces settings.cfg)
    countries.yaml                  # Country lists, blacklist, Tor Metrics cache
    bridges.yaml                    # Bridge configuration (Snowflake, obfs4, WebTunnel)
    profiles.yaml                   # Profile definitions (stealth, balanced, streaming, pentest)
    useragents.yaml                 # Bundled Tor Browser User-Agent list
  tests/
    smoke.sh                        # Automated smoke test suite (Docker)
  Doc/
    MIGRATION.md                    # Bash -> Go migration guide
  legacy/                           # Original Bash version preserved for reference
    splitter.sh
    func/
    settings.cfg
  Dockerfile
  docker-compose.yml                # Production (uses ghcr.io image)
  docker-compose.dev.yml            # Development (builds from source)
  .github/
    workflows/
      ci.yml                        # CI pipeline
      release.yml                   # Release automation
```

---

## Phase 3: Go Implementation ✅ DONE

### 3.1 Foundation ✅

| # | Task | Detail | Status |
|---|------|--------|--------|
| 3.1.1 | **Go module + Cobra CLI** | Initialize `go.mod`. Set up Cobra with subcommands: `run`, `status`, `test`, `version`. Long flags: `--instances`, `--countries`, `--relay-enforce`. Legacy aliases: `-i`, `-c`. Note: `-re` is kept as a two-character alias for backward compat with the Bash version, but it is not a standard single-letter flag — document it clearly as a legacy alias. Add `--profile`, `--proxy-mode`, `--bridge-type`, `--verbose` flags. | ✅ |
| 3.1.2 | **Configuration system** | Load config from `configs/default.yaml`, override with env vars (`SPLITTER_*` prefix), override with CLI flags. Validate all values (instances > 0, valid relay modes, port ranges). Profile support: `--profile stealth` loads `profiles.yaml[stealth]` as base. | ✅ |
| 3.1.3 | **Structured logging** | Use `log/slog` (Go 1.21+). Levels: DEBUG, INFO, WARN, ERROR. **Logging is OFF by default** (philosophy: no logs, no crime). Enable via `--log` flag or `SPLITTER_LOG=1` env var. When enabled: JSON format for Docker (detected via `TERM=dumb` or `NO_COLOR`), text for terminal. `--log-level` controls verbosity (default INFO when logs are on). | ✅ |
| 3.1.4 | **Process lifecycle manager** | `internal/process/` package. Spawn child processes (tor, haproxy, privoxy) with `os/exec`. Track PIDs. Graceful shutdown: SIGTERM -> wait 5s -> SIGKILL. `trap` equivalent via `signal.NotifyContext`. Kill all children on exit. Clean up temp files. | ✅ |

### 3.2 Core Services ✅

| # | Task | Detail | Status |
|---|------|--------|--------|
| 3.2.1 | **Tor instance manager** | `internal/tor/` package. Each Tor instance is a goroutine-managed process. Generate torrc from `templates/torrc.gotmpl`. Auto-detect Tor version at startup (`tor --version`). Conditionally enable Conflux, CGO, congestion control, HTTPTunnelPort based on detected version. Track state: starting, bootstrapping, ready, failed. **Auto-restart on failure**: if a Tor process exits unexpectedly, the goroutine restarts it with backoff (1s, 2s, 4s, max 30s). Failure counter resets after successful bootstrap. **Hidden service per instance**: each Tor instance generates a hidden service on a unique port (`HiddenServiceDir`, `HiddenServicePort`), preserved from the Bash version. | ✅ |
| 3.2.2 | **HAProxy manager** | `internal/haproxy/` package. Generate config from `templates/haproxy.cfg.gotmpl`. Shuffle backend order (`math/rand`) for anti-correlation. Health check configuration (TCP checks). Start/stop/reload process. **Stats page** on port 63539 with random password generated at startup. **Fixed: HAProxy 3.x compatibility** (`stats auth admin:pw` instead of `stats admin pw`). **Fixed: TCP health checks** instead of httpchk (Tor HTTPTunnelPort is a CONNECT proxy). | ✅ |
| 3.2.3 | **Proxy abstraction** | `internal/proxy/` package. Two modes: `native` uses Tor's `HTTPTunnelPort` directly (no Privoxy), `legacy` generates Privoxy configs. Mode selected via `--proxy-mode` flag or profile. In native mode, HAProxy backends point directly at Tor HTTPTunnelPort listeners. | ✅ |
| 3.2.4 | **Country selection + rotation** | `internal/country/` package. Random selection without duplicates. Rotation daemon runs as a goroutine: periodically selects a random country, rewrites torrc, restarts the affected instances. Configurable interval with jitter (`--country-interval`, default 120s +- random). | ✅ |
| 3.2.5 | **Circuit renewal** | `internal/circuit/` package. Connect to Tor control port, authenticate via **cookie auth** (`CookieAuthentication 1` in torrc, read `control_auth_cookie` file), send `SIGNAL NEWNYM`. Replace `expect` scripts with Go's `net.Conn` + Tor control protocol. Randomized renewal intervals per instance (10-15s range). | ✅ |
| 3.2.6 | **Port allocation** | `internal/network/` package. Find available ports by attempting `net.Listen` on each port. No more `netstat` or `ss` calls. Thread-safe port allocator for concurrent instance startup. | ✅ |

### 3.3 Integration ✅

| # | Task | Detail | Status |
|---|------|--------|--------|
| 3.3.1 | **Orchestrator** | `cmd/run.go` wires everything together: parse config -> kill previous -> create temp dirs -> start tor instances -> start HAProxy -> start circuit renewal -> start country rotation daemon -> block until context cancelled. | ✅ |
| 3.3.2 | **Dependency detection** | At startup, check that `tor`, `haproxy` (and `privoxy` if legacy mode) exist in `$PATH`. Detect Tor version and feature support (Conflux, HTTPTunnelPort, CGO). Fail fast with clear message if anything is missing. | ✅ |
| 3.3.3 | **Environment variable overrides** | Any config value can be overridden via `SPLITTER_*` env vars. E.g., `SPLITTER_INSTANCES=10`, `SPLITTER_COUNTRIES=6`, `SPLITTER_RELAY_ENFORCE=exit`. Priority: CLI flag > env var > config file > default. | ✅ |
| 3.3.4 | **Status dashboard** | `cmd/status.go` - terminal UI showing live instance state, countries, circuit count, health. Uses `fmt` + ANSI escape codes (no external TUI dependency). Exposed as HTTP at `/status`. | ✅ |
| 3.3.5 | **SIGHUP config reload** | Handle `SIGHUP` via `signal.NotifyContext` to reload `configs/default.yaml` without full restart. On reload: update country lists, rotation intervals, and HAProxy config. Tor instances are not restarted unless a parameter affecting torrc changes. Print reload summary to stdout. | ✅ |

---

## Phase 4: Tests ✅ DONE

| # | Task | Detail | Status |
|---|------|--------|--------|
| 4.1 | **Unit tests** | `go test ./internal/...` for each package (15 packages). Mock external dependencies (tor binary, network). | ✅ |
| 4.2 | **Test CLI parsing** | Test valid args, invalid args, missing flags, short/long flag equivalence, profile loading. | ✅ |
| 4.3 | **Test country selection** | Test random selection without duplicates, empty list handling, rotation logic. | ✅ |
| 4.4 | **Test port allocation** | Test concurrent port allocation, handling of occupied ports. | ✅ |
| 4.5 | **Test config generation** | Test torrc, haproxy.cfg, privoxy.cfg templates with various relay modes and profiles. Verify generated output matches expected structure. | ✅ |
| 4.6 | **Test relay config** | Test EntryNodes/ExitNodes/ExcludeNodes for each mode (entry/exit/speed). Test Conflux/CGO flags appear conditionally based on Tor version. | ✅ |
| 4.7 | **Test process lifecycle** | Test graceful shutdown, SIGTERM timeout, child process cleanup. | ✅ |
| 4.8 | **Integration tests** | `go test -tags=integration ./...` — tests that spawn real tor/haproxy processes. Skipped in CI without dependencies. | ✅ |
| 4.9 | **Privacy test suite** | 3-layer privacy testing: Layer 1 (unit: torrc security assertions, config defaults), Layer 2 (integration: DNS leak, circuit rotation, cookie auth), Layer 3 (smoke: log safety, header leaks, security options). | ✅ |
| 4.10 | **Smoke test script** | `tests/smoke.sh` — automated end-to-end validation: torrc verify-config, HAProxy config, port binding, instance health, proxy functionality, IP rotation, privacy checks. | ✅ |
| 4.11 | **Makefile** | `Makefile` with targets: `make check`, `make test`, `make test-privacy`, `make docker-up`, `make smoke`, etc. | ✅ |

---

## Phase 5: Docker and CI/CD ✅ DONE

| # | Task | Detail | Status |
|---|------|--------|--------|
| 5.1 | **Multi-stage Dockerfile** | Stage 1 (`golang:1.23-alpine`): compile static binary. Stage 2 (`alpine:3.21`): copy binary + install tor (0.4.9.6) + haproxy (3.0). HEALTHCHECK hitting `/healthz` endpoint. Non-root user. | ✅ |
| 5.2 | **docker-compose.yml** | Production compose (`docker-compose.yml`) uses ghcr.io image. Dev compose (`docker-compose.dev.yml`) builds from source with volume-mounted configs. Optional `prometheus` + `grafana` services for monitoring. | ✅ |
| 5.3 | **GitHub Actions CI** | Jobs: `lint` (`golangci-lint run`), `test` (`go test ./...`), `build` (`docker build`), `push` (only on tags). Cross-compile for `linux/amd64` and `linux/arm64`. | ✅ |
| 5.4 | **Release automation** | Release v2.0.0-beta-01 created. GitHub release with notes, Docker image tagged. `release.yml` workflow for future releases. | ✅ |

---

## Phase 6: Documentation ✅ DONE

| # | Task | Detail | Status |
|---|------|--------|--------|
| 6.1 | **Update README** | Go build instructions, new CLI reference, docker-compose usage, profiles, migration from Bash. | ✅ |
| 6.2 | **Update AGENTS.md** | Reflect Go structure, `go test` commands, Go code style conventions, lint commands. | ✅ |
| 6.3 | **Migration guide** | Document differences between Bash and Go versions. Config mapping (settings.cfg -> default.yaml). Feature parity matrix. | ✅ `Doc/MIGRATION.md` |

---

## Phase 7: Modern Tor Features 🔄 PARTIAL

Features based on Tor 0.4.7-0.4.9 (2023-2026) changelog analysis and current Tor ecosystem.

### 7.1 Speed Improvements ✅

| # | Task | Detail | Status |
|---|------|--------|--------|
| 7.1.1 | **Conflux (multi-leg circuits)** | Tor 0.4.8 introduced Conflux: traffic is split across multiple circuit legs simultaneously at the protocol level. This is what SPLITTER approximates manually via HAProxy load balancing. Enabling `ConfluxEnabled 1` in each tor instance config would let Tor natively aggregate bandwidth across legs. Combined with SPLITTER's per-country instance strategy, this would multiply throughput and resilience. Requires Tor >= 0.4.8. | ✅ Auto-detected and conditionally enabled |
| 7.1.2 | **Replace Privoxy with HTTPTunnelPort** | Tor 0.4.8+ has native HTTP CONNECT proxy support via `HTTPTunnelPort`. This eliminates the Privoxy layer entirely. The TCP path simplifies from User -> HAProxy -> Privoxy -> Tor to User -> HAProxy -> Tor. Removes one hop, reduces latency, eliminates a dependency, and removes Privoxy config generation complexity. HAProxy backend targets become Tor HTTPTunnelPort listeners directly. Add a `--proxy-mode` flag: `native` (HTTPTunnelPort, recommended) or `legacy` (Privoxy, backward compat). | ✅ Default mode is `native` |
| 7.1.3 | **Congestion Control tuning** | Tor 0.4.7+ has congestion control (`CongestionControlAuto`). Enable it by default and expose tuning parameters in config. This dramatically improves throughput on long-distance circuits. Set `CongestionControlAuto 1` in generated tor configs. | ✅ Auto-detected and conditionally enabled |

### 7.2 Security Improvements 🔄

| # | Task | Detail | Status |
|---|------|--------|--------|
| 7.2.1 | **CGO - Counter Galois Onion encryption** | Tor 0.4.9 introduced CGO relay cryptography with improved resistance to tagging attacks, better forward secrecy, and better forgery resistance. SPLITTER should auto-detect Tor version and enable CGO when available. Detect via `tor --version` and conditionally set `CGOEnabled 1` (or equivalent config) in generated torrc. | ✅ Detection via `Version.SupportsCGO()`. **Note**: CGO is not a torrc option — detection is used for informational/logging only. Previously emitted invalid `CGOEnabled 1` torrc directive; removed after `tor --verify-config` failure. |
| 7.2.2 | **Post-Quantum Key Exchange (ML-KEM768)** | Tor 0.4.8.17+ supports post-quantum key agreement via ML-KEM768 when built with OpenSSL 3.5.0+. This protects against "harvest now, decrypt later" attacks. SPLITTER should check if the Tor binary supports it (`tor --dump-config` or version check) and enable it when available. | ✅ Detection via `Version.SupportsPostQuantum()` and TLS/OpenSSL version check. Runtime verification shows `X25519MLKEM768` in TLS handshake. |
| 7.2.3 | **Bridge / Pluggable Transport support** | Add support for Tor bridges (Snowflake, WebTunnel, obfs4) via `--bridge` flag. This solves the main threat from the README: connecting to Tor from networks that block it. Instead of requiring a VPS + VPN chain, users in censored regions can use a Snowflake bridge. Config via `Bridge` lines in torrc. Add `--bridge-type snowflake|webtunnel|obfs4|none` flag and `configs/bridges.yaml` file. | ✅ |
| 7.2.4 | **Happy Families awareness** | Tor 0.4.9 introduced "happy families" for relay grouping. When selecting entry/exit countries, SPLITTER should avoid circuits where multiple relays belong to the same operator. Enable by ensuring generated torrc respects family restrictions. No direct config needed on the client side, but document that upgrading to Tor 0.4.9+ enables this automatically. | ✅ Detection via `Version.SupportsHappyFamilies()`. Automatic — no torrc directive needed. |
| 7.2.5 | **TLS 1.3 enforcement** | Tor 0.4.9 now requires TLS 1.2 minimum and recommends TLS 1.3. SPLITTER should verify the Tor binary supports TLS 1.3 and document this as a requirement for the Docker image. | ✅ Detection via `Version.SupportsTLS13()`. Verified in Docker image (TLS 1.3 with X25519MLKEM768). |
| 7.2.6 | **Sandboxing** | Enable `Sandbox 1` in generated torrc (Tor's built-in seccomp-bpf sandbox, available on Linux). Add a seccomp profile to the Docker image (`--security-opt seccomp=splitter.json`) restricting syscalls to the minimum required by tor + haproxy. The `stealth` profile enables both by default; other profiles document the tradeoff (sandbox adds ~5% latency). Verify sandbox compatibility with the target Tor version at startup — some older builds have sandbox bugs. | 🔄 Template supports `Sandbox 1` (conditional on `SandboxEnabled`). Docker seccomp profile not yet created. |

### 7.3 New Features ✅

| # | Task | Detail | Status |
|---|------|--------|--------|
| 7.3.1 | **Auto-update country lists from Tor Metrics** | Replace the hardcoded 32-country list from 2018 with dynamic fetching from the Tor Metrics API. Query `https://metrics.torproject.org/rs/search` to find countries with active exit/guard relays. Cache results locally with TTL (e.g., 24h). Fallback to `configs/countries.yaml` when offline. Add `--auto-countries` flag to enable. | ✅ Tor Metrics API client implemented (`internal/country/metrics.go`). Cache with TTL. Fallback to YAML. |
| 7.3.2 | **Stream Isolation via SOCKS5 auth** | Use Tor's `IsolateSOCKSAuth` feature to isolate streams by destination without creating extra instances. Each destination gets its own circuit via SOCKS5 username/password isolation. This provides finer-grained separation than the current per-instance model. Add `--stream-isolation` flag. | ✅ Enabled via `--stream-isolation` or `pentest` profile. Renders `IsolateSOCKSAuth` in torrc. |
| 7.3.3 | **IPv6 dual-stack support** | Add IPv6 relay selection alongside IPv4. Many countries now have IPv6 Tor relays. Add `ClientUseIPv6 1` to torrc, and allow country selection to consider IPv6 relay availability. Add `--ipv6` flag to enable. | ✅ Disabled by default (`ClientUseIPv6 0`). Enable via `--ipv6` flag. Template tested for both states. |
| 7.3.4 | **Prometheus metrics endpoint** | Expose a `/metrics` HTTP endpoint using Go's `net/http` + Prometheus client library. Metrics: active instances, circuits per instance, country distribution, latency percentiles, error rates, bandwidth usage, bootstrap progress. This enables Grafana dashboards and alerting. | ✅ `/metrics` endpoint. Custom Prometheus registry. |
| 7.3.5 | **Circuit fingerprinting resistance** | Adaptive circuit rotation based on traffic patterns. If a burst of requests is detected, randomize circuit selection more aggressively. For steady traffic, rotate at variable intervals (not fixed 10s). This defeats timing correlation attacks that exploit predictable rotation patterns. | ✅ Adaptive circuit rotation (`internal/circuit/adaptive.go`). Burst/moderate/idle detection with variance. |
| 7.3.6 | **Exit node reputation checking** | Before assigning an exit node, check its reputation against public datasets. Query the Tor Metrics API for exit relay flags, uptime, and bandwidth. Optionally cross-reference with community blocklists. Skip exit nodes that are newly appeared (possible honeypots) or have been flagged. Add `--exit-reputation` flag. | ✅ Onionoo API client (`internal/health/reputation.go`). Score computation, cache, filtering. |
| 7.3.7 | **DNS leak test** | Built-in test that verifies all DNS queries go exclusively through Tor. Run at startup and periodically. Use a known test domain that resolves to a unique address; if the address differs from what Tor resolves, flag a leak. Expose result in metrics and logs. Accessible via `splitter test dns`. | ✅ DNS leak detection (`internal/health/dnsleak.go`). SOCKS5 resolution + IP comparison. `splitter test dns` command. |
| 7.3.8 | **Configuration profiles** | Predefined profiles in `configs/profiles.yaml`: `stealth` (max security, many instances, aggressive rotation, Conflux enabled, no speed mode), `balanced` (current default behavior, good tradeoff), `streaming` (Conflux + congestion control + speed mode, for media), `pentest` (extreme rotation, randomized User-Agent per request, stream isolation). Add `--profile` flag. | ✅ 4 profiles: stealth, balanced, streaming, pentest. |
| 7.3.9 | **Bundled Tor Browser User-Agent list** | Replace the hardcoded 2018 Firefox UA (`Mozilla/5.0 (Windows NT 6.1; rv:52.0) Gecko/20100101 Firefox/52.0`) with a curated list in `configs/useragents.yaml`, updated at release time. **Do not fetch at runtime** — an outbound HTTP call at startup is a privacy risk (logged by the remote, timing side-channel). The list is rotated randomly per instance. Update the bundled list as part of the release process. | ✅ `configs/useragents.yaml` bundled. Random rotation per instance. |

### 7.4 Arti (Rust Tor Client) Support ⏳ DEFERRED

> **Note**: Arti currently lacks GeoIP-based path selection, which is the core mechanism SPLITTER relies on for anti-correlation. Until that feature lands, Arti cannot be a functional backend for SPLITTER. This phase is deferred until Arti reaches parity on that specific feature. Track progress at [gitlab.torproject.org/tpo/core/arti](https://gitlab.torproject.org/tpo/core/arti). Re-evaluate quarterly.

When GeoIP path selection is available in Arti, the implementation plan is:

| # | Task | Detail | Status |
|---|------|--------|--------|
| 7.4.1 | **Dual backend support** | Add `--tor-backend c-tor\|arti` flag. When `arti` selected, generate Arti TOML config instead of torrc. Feature-gate: hidden services, bridges, pluggable transports, and conflux remain c-tor only until Arti supports them. | ⏳ Deferred |
| 7.4.2 | **Feature parity tracking** | Maintain a compatibility matrix in docs showing which SPLITTER features work with each backend. Key blocker: GeoIP-based path selection. Secondary blockers: conflux, pluggable transports, hidden services. | ⏳ Deferred |

---

## Phase 8: Future Expansion ⏳ NOT STARTED

| # | Task | Detail | Status |
|---|------|--------|--------|
| 8.1 | **Client mode** | `splitter client` - lightweight proxy for individual use without HAProxy. Single Tor instance with circuit rotation. No load balancing, no multi-instance overhead. For when you just want a rotating Tor proxy. | ⏳ |
| 8.2 | **TUI dashboard** | Terminal UI using `bubbletea` or `tview` for real-time monitoring of instances, circuits, countries, bandwidth. Interactive: swap country, force new circuit, restart instance, all from the TUI. | ⏳ |
| 8.3 | **REST API** | HTTP API for remote management: `GET /api/instances`, `POST /api/instances/{id}/rotate`, `GET /api/countries`, etc. Enables integration with external tools and scripts. | ⏳ |
| 8.4 | **Multi-node coordination** | *(Out of scope for this roadmap — would require a separate distributed control plane project. Tracked separately.)* | ⏳ |

---

## Bugfixes Applied (v2.0.0-beta-01)

| Fix | Detail |
|-----|--------|
| **HAProxy stats port conflict** | Stats port default changed from `63537` (same as HTTPPort) to `63539`. HAProxy was failing to start due to duplicate bind. |
| **HAProxy 3.x `stats admin` syntax** | Changed `stats admin <pw>` to `stats auth admin:<pw>` + `stats admin if TRUE`. HAProxy 3.x requires `if`/`unless` condition. |
| **HAProxy health checks** | Replaced `option httpchk GET https://google.com/` with `option tcp-check` + `tcp-check connect`. Tor's HTTPTunnelPort is a CONNECT proxy, not an HTTP server — httpchk would never succeed. |
| **Invalid CGOEnabled torrc option** | Removed `CGOEnabled 1` from torrc template. CGO is not a torrc directive — it's a compile-time Tor feature. Caused `tor --verify-config` to fail with "Unknown option". |
| **Obsolete OptimisticData torrc option** | Removed `OptimisticData` from torrc template. Obsolete in Tor 0.4.9.x, caused warnings and potential parse failures. |
| **SOCKS5 through HAProxy** | Fixed: added explicit `mode tcp` to `backend tor_socks` in HAProxy template. The backend was inheriting `mode http` from defaults, causing SOCKS5 handshakes to be misinterpreted as HTTP. |

---

## Suggested Execution Order

1. ~~**Phase 1** (infrastructure: Go module, move Bash to legacy/, CI scaffold)~~ ✅
2. ~~**Phase 2** (project structure: directories, Go module layout)~~ ✅
3. ~~**Phase 3.1** (foundation: Cobra CLI, config system, logging, process manager)~~ ✅
4. ~~**Phase 5** (Docker/CI/CD) -> CI pipeline active from day one, catches regressions as core services are built~~ ✅
5. ~~**Phase 4** (tests for foundation packages) -> safety net~~ ✅
6. ~~**Phase 3.2** (core services: tor manager, HAProxy, proxy, country, circuit, port allocation)~~ ✅
7. ~~**Phase 3.3** (integration: orchestrator, dependency detection, status dashboard)~~ ✅
8. ~~**Phase 7.1** (speed: Conflux, HTTPTunnelPort, congestion control) -> biggest user-visible impact~~ ✅
9. ~~**Phase 7.2** (security: CGO, post-quantum, bridges)~~ ✅ (partial: Docker seccomp profile pending)
10. ~~**Phase 7.3** (new features: auto-countries, metrics, profiles, etc.)~~ ✅
11. ~~**Phase 6** (documentation)~~ ✅
12. **Phase 7.2.6** (Docker seccomp profile) 🔄
13. **Phase 7.4** (Arti support, when GeoIP parity lands) + **Phase 8** (future expansion) ⏳

---

## Open Questions

### Resolved

- **Hidden services per instance**: yes, maintained. Each Tor instance generates a hidden service on a unique port (`HiddenServiceDir`, `HiddenServicePort`).
- **Logging strategy**: off by default (`--log` / `SPLITTER_LOG=1` to enable). Philosophy: no logs, no crime.
- **HAProxy stats page**: preserved on port 63539 with random password generated at startup.
- **Control port auth**: cookie auth (`CookieAuthentication 1` in torrc, read `control_auth_cookie` file).
- **Minimum Tor version**: 0.4.8 (Conflux + HTTPTunnelPort). Runtime image uses 0.4.9.6.
- **Privoxy support**: kept as legacy fallback via `--proxy-mode legacy`. Default is `native` (HTTPTunnelPort).
- **License**: BSD 3-Clause, inherited from the original project. Must remain BSD since SPLITTER is a derivative work.
- **Architecture targets**: `linux/amd64` and `linux/arm64` from day one.
- **Prometheus metrics**: embedded in the splitter binary, enabled via `--metrics` flag (off by default).
- **Cache TTL**: 12h for Tor Metrics country lists and Onionoo exit reputation.
- **Arti support**: deferred until GeoIP path selection is available. Re-evaluate quarterly. Does not block releases.
