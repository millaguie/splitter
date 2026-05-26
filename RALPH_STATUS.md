# Ralph Loop Status

## RALPH LOOP COMPLETE — ALL EXECUTABLE TASKS IMPLEMENTED

**Last update**: v2.0.0-beta-01 (squashed, pushed, tagged, released).

## ALL EXECUTABLE ROADMAP TASKS COMPLETE

Phase 1-7.3 implementation is feature-complete. Only Phase 7.2.6 (Docker seccomp), Phase 7.4 (Arti), and Phase 8 (future expansion) remain.

## Post-Rewrite Bugfixes

| Fix | Commit |
|-----|--------|
| HAProxy stats port conflict (63537→63539) | 1a97efb |
| HAProxy 3.x `stats admin` syntax | 1a97efb |
| HAProxy health checks (httpchk→tcp-check) | 1a97efb |
| Invalid CGOEnabled torrc option | 1a97efb |
| Obsolete OptimisticData torrc option | 1a97efb |
| SOCKS5 through HAProxy (mode tcp) | 8ff24df |
| Cache TTL 24h→12h (country + reputation) | 578c313 |

## Completed Phases

### Phase 1: Project Infrastructure ✅
### Phase 2: Go Project Structure ✅
### Phase 3: Go Implementation ✅
- 3.1 Foundation: Cobra CLI, config system, logging, process lifecycle
- 3.2 Core Services: Tor manager, HAProxy, proxy abstraction, country/circuit/port
- 3.3 Integration: Orchestrator, dependency detection, env overrides, status dashboard, SIGHUP reload

### Phase 4: Tests ✅
- Unit tests (15 packages, table-driven)
- Integration tests (`-tags=integration`)
- 3-layer privacy test suite (torrc assertions, config defaults, DNS leak, circuit rotation)
- Smoke test script (`tests/smoke.sh`)
- Makefile (`make check`, `make test-privacy`, `make smoke`)

### Phase 5: Docker and CI/CD ✅
- Multi-stage Dockerfile (alpine:3.21, Tor 0.4.9.6, HAProxy 3.0)
- docker-compose.yml (production) + docker-compose.dev.yml (development)
- GitHub Actions CI (lint + test + build + push)
- Release automation — Cross-compile 4 platforms, GitHub Release, GHCR push

### Phase 6: Documentation ✅
- README, AGENTS.md, Doc/MIGRATION.md
- ROADMAP.md with status markers
- SETTINGS_MAP.md (541+ lines settings.cfg → Go mapping)

### Phase 7.1: Speed Improvements ✅
### Phase 7.2: Security Improvements ✅ (5/6)
### Phase 7.3: New Features ✅ (9/9)
- Auto-update country lists (Tor Metrics API, 12h TTL cache)
- Stream Isolation, IPv6 dual-stack, Prometheus metrics
- Circuit fingerprinting resistance, exit node reputation, DNS leak test
- Configuration profiles (stealth, balanced, streaming, pentest)
- Bundled Tor Browser User-Agent list

## Pending

### Phase 7.2.6: Docker seccomp profile 🔄
Create `splitter.json` seccomp profile restricting syscalls to minimum required by tor + haproxy. The `stealth` profile enables `Sandbox 1` in torrc; the Docker profile is the complement.

### Phase 7.4: Arti (Rust Tor Client) ⏳ BLOCKED
Arti lacks GeoIP-based path selection — the core mechanism SPLITTER relies on.
Track: https://gitlab.torproject.org/tpo/core/arti — Re-evaluate quarterly.

### Phase 8: Future Expansion ⏳
- 8.1 Client mode (single-instance lightweight proxy)
- 8.2 TUI dashboard (bubbletea/tview)
- 8.3 REST API for remote management
- 8.4 Multi-node coordination (separate project)

## Resolved Decisions

| Question | Decision |
|----------|----------|
| License | BSD 3-Clause, inherited from original project |
| Architecture targets | `linux/amd64` + `linux/arm64` from day one |
| Prometheus metrics | Embedded in binary, enabled via `--metrics` (off by default) |
| Cache TTL | 12h for country lists and exit reputation |
| SOCKS5 through HAProxy | Fixed — explicit `mode tcp` in backend + frontend |
| Minimum Tor version | 0.4.8 (Conflux + HTTPTunnelPort); runtime uses 0.4.9.6 |

## How to Release

```bash
git tag v2.0.0-beta-02
git push origin v2.0.0-beta-02
```

This triggers `.github/workflows/release.yml` which:
1. Cross-compiles for linux/amd64, linux/arm64, darwin/amd64, darwin/arm64
2. Creates GitHub Release with binaries + checksums
3. Pushes multi-arch Docker image to GHCR
