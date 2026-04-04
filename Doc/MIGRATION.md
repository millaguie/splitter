# SPLITTER Migration Guide: Bash to Go

## 1. Overview

SPLITTER has been rewritten from Bash (~541 lines of shell scripts) to Go. The Bash version
is preserved in `legacy/` for reference. This guide helps existing users migrate to the Go
version.

**Quick start**: Most users can run `./splitter run --profile balanced` to get behavior
equivalent to the old defaults, then customize via flags or `configs/default.yaml`.

---

## 2. CLI Compatibility Matrix

| Bash Flag | Go Flag | Short | Notes |
|-----------|---------|-------|-------|
| `-i N` | `--instances N` | `-i` | Identical behavior |
| `-c N` | `--countries N` | `-c` | Identical behavior |
| `-re MODE` | `--relay-enforce MODE` | `-r` / `-re` | `-re` kept as legacy alias. New short flag `-r` also works. Modes: `entry`, `exit`, `speed` |
| — | `--profile NAME` | — | **NEW**. Quick config: `stealth`, `balanced`, `streaming`, `pentest` |
| — | `--proxy-mode MODE` | — | **NEW**. `native` (default, HTTPTunnelPort) or `legacy` (Privoxy) |
| — | `--bridge-type TYPE` | — | **NEW**. `snowflake`, `webtunnel`, `obfs4`, `none` |
| — | `--verbose` | — | **NEW**. Verbose output |
| — | `--log` | — | **NEW**. Enable logging (off by default) |
| — | `--log-level LEVEL` | — | **NEW**. `debug`, `info`, `warn`, `error` |
| — | `--auto-countries` | — | **NEW**. Fetch country list from Tor Metrics API |
| — | `--stream-isolation` | — | **NEW**. IsolateSOCKSAuth |
| — | `--ipv6` | — | **NEW**. ClientUseIPv6 |
| — | `--exit-reputation` | — | **NEW**. Check exit node reputation |

### Subcommands

The Go version introduces subcommands (Cobra CLI):

| Command | Description |
|---------|-------------|
| `splitter run` | Start SPLITTER (replaces `bash splitter.sh`) |
| `splitter status` | Live dashboard showing instance state, countries, circuits |
| `splitter test dns` | Verify all DNS queries go through Tor |
| `splitter test exit-reputation` | Check exit node reputation |
| `splitter version` | Show detected Tor version and feature support |

---

## 3. Config File Migration

### Format Change

| Aspect | Bash | Go |
|--------|------|-----|
| File | `func/settings.cfg` (shell variables) | `configs/default.yaml` (YAML) |
| Types | All strings, shell-evaluated | Proper types: int, bool, string, list |
| Overrides | Edit file only | CLI flags > env vars (`SPLITTER_*`) > YAML > defaults |
| Country format | Comma-separated `{XX}` string | YAML list of `{XX}` strings |

### Full Parameter Mapping

The complete mapping of all 81 Bash variables is documented in
[`configs/SETTINGS_MAP.md`](../configs/SETTINGS_MAP.md). Key highlights:

**62 variables ported** to YAML with proper typing. Examples:

```yaml
# Bash: TOR_INSTANCES=2
instances:
  per_country: 2

# Bash: COUNTRY_LIST_CONTROLS=entry
relay:
  enforce: "entry"

# Bash: ACCEPTED_COUNTRIES="{us},{de},{fr},..."
country:
  accepted:
    - "{us}"
    - "{de}"
    - "{fr}"
```

**15 variables dropped** (Go handles at runtime):

| Bash Variable | Reason Dropped |
|---------------|---------------|
| `USER_ID`, `USER_UID`, `USER_GID` | Go uses `os/user` and `os.Getuid()` |
| `RAND_PASS`, `TORPASS` | Go uses cookie auth; generates passwords with `crypto/rand` |
| `TOR_CURRENT_INSTANCE`, `TOR_CURRENT_SOCKS_PORT`, etc. | Go manages state in memory |
| `PRIVOXY_CURRENT_INSTANCE`, `PRIVOXY_CURRENT_PORT` | Go port allocator manages |
| `MASTER_PROXY_PASSWORD` | Was never defined (bug); Go auto-generates |
| `SPOOFED_USER_AGENT` | Go handles escaping at runtime |

**12 variables had conflicting defaults** between `legacy/func/settings.cfg` and
`legacy/settings.cfg`. The Go version uses the `func/` values (the ones actually sourced
by the script):

| Variable | func/ value | root/ value | Go default |
|----------|-------------|-------------|------------|
| `RETRIES` | 1000 | 100 | 1000 |
| `MINIMUM_TIMEOUT` | 15 | 20 | 15 |
| `CircuitsAvailableTimeout` | 5 | 360 | 5 |
| `CircuitStreamTimeout` | 20 | 30 | 20 |
| `ConnectionPadding` | 0 | 1 | 0 |
| `TrackHostExitsExpire` | 10 | 120 | 10 |
| `HEALTH_CHECK_INTERVAL` | 12 | 3 | 12 |
| `MASTER_PROXY_STAT_PORT` | 63537 | 63539 | 63537 |

### Derived Timeouts

The Bash version computed some timeouts via shell arithmetic. The Go version computes
them at runtime from the base settings:

| Bash Expression | Go Computation |
|-----------------|---------------|
| `PRIVOXY_TIMEOUT = CircuitStreamTimeout + MINIMUM_TIMEOUT` | Same formula, runtime |
| `SocksTimeout = CircuitStreamTimeout + MINIMUM_TIMEOUT` | Same formula, runtime |
| `MASTER_PROXY_CLIENT_TIMEOUT = RETRIES * SERVER_TIMEOUT * COUNTRIES` | Same formula, runtime |

### New YAML Fields (No Bash Equivalent)

| YAML Key | Purpose |
|----------|---------|
| `tor.conflux_enabled` | Conflux multi-leg circuits (Tor 0.4.8+) |
| `tor.congestion_control_auto` | Congestion control (Tor 0.4.7+) |
| `tor.cgo_enabled` | CGO relay encryption (Tor 0.4.9+) |
| `tor.post_quantum_enabled` | ML-KEM768 key exchange (Tor 0.4.8.17+) |
| `tor.sandbox` | seccomp-bpf sandbox (Linux) |
| `tor.stream_isolation` | IsolateSOCKSAuth |
| `tor.client_use_ipv6` | IPv6 dual-stack |
| `country.auto_fetch` | Dynamic country lists from Tor Metrics API |
| `health.exit_reputation` | Exit node reputation checking |
| `metrics.enabled` | Prometheus `/metrics` endpoint |
| `user_agent.bundle_file` | Path to bundled UA list (`configs/useragents.yaml`) |

---

## 4. Dependency Changes

| Dependency | Bash | Go | Notes |
|------------|------|----|-------|
| **tor** | Required | Required | Go auto-detects version and conditionally enables features |
| **haproxy** | Required | Required | No change |
| **privoxy** | Required | **Optional** | Only in `--proxy-mode legacy`. Native mode uses HTTPTunnelPort |
| **expect** | Required | **Removed** | Go uses native Tor control protocol for NEWNYM |
| **proxychains** | Optional | **Removed** | Go handles proxy chaining natively |
| **netstat / ss** | Required | **Removed** | Go uses `net.Listen` for port availability checks |
| **shuf / sort -R** | Required | **Removed** | Go uses `math/rand` |
| **bash** | Required | **Removed** | Single static Go binary, no interpreter |

**Result**: The Go version requires only `tor` and `haproxy` installed (plus `privoxy` only
if using legacy proxy mode).

---

## 5. Feature Parity Matrix

### Core Features (Carried Forward)

| Feature | Bash | Go | Differences |
|---------|------|-----|-------------|
| Multiple Tor instances | ✅ | ✅ | Go adds auto-restart with exponential backoff |
| HAProxy load balancing | ✅ | ✅ | Identical; roundrobin or leastconn based on mode |
| Country-based entry/exit enforcement | ✅ | ✅ | entry, exit, speed modes |
| Country rotation daemon | ✅ | ✅ | Go adds configurable interval with jitter |
| Circuit renewal (NEWNYM) | ✅ | ✅ | Go uses cookie auth instead of `expect` + hashed password |
| Randomized circuit intervals | ✅ | ✅ | Go adds adaptive modes: burst, moderate, idle |
| Hidden service per instance | ✅ | ✅ | HiddenServiceDir + HiddenServicePort per instance |
| HAProxy stats page | ✅ | ✅ | Random password generated at startup |
| Health checks | ✅ | ✅ | URL-based with configurable interval and thresholds |
| Privoxy HTTP-to-SOCKS bridge | ✅ | ✅ | Legacy mode only (`--proxy-mode legacy`) |
| Docker support | ✅ | ✅ | Multi-stage build, smaller image, healthcheck |

### New Features (Go Only)

| Feature | Flag / Config | Tor Version | Notes |
|---------|--------------|-------------|-------|
| **HTTPTunnelPort (native proxy)** | `--proxy-mode native` | 0.4.8+ | Eliminates Privoxy. Default mode. |
| **Configuration profiles** | `--profile stealth\|balanced\|streaming\|pentest` | — | Predefined tuning presets |
| **Conflux (multi-leg circuits)** | Auto (tor.conflux_enabled) | 0.4.8+ | Traffic split across parallel circuit legs |
| **Congestion control** | Auto (tor.congestion_control_auto) | 0.4.7+ | Dramatic throughput improvement |
| **Post-quantum key exchange** | Auto (tor.post_quantum_enabled) | 0.4.8.17+ | ML-KEM768, requires OpenSSL 3.5+ |
| **CGO encryption** | Auto (tor.cgo_enabled) | 0.4.9+ | Improved relay cryptography |
| **Bridge / Pluggable Transport** | `--bridge-type snowflake\|webtunnel\|obfs4` | — | Censorship circumvention |
| **Sandboxing** | Profile: stealth | — | seccomp-bpf via `Sandbox 1` in torrc |
| **IPv6 dual-stack** | `--ipv6` | — | ClientUseIPv6 |
| **Stream isolation** | `--stream-isolation` | — | IsolateSOCKSAuth per destination |
| **Auto country lists** | `--auto-countries` | — | Tor Metrics API with 24h cache |
| **Exit node reputation** | `--exit-reputation` | — | Onionoo API scoring |
| **DNS leak test** | `splitter test dns` | — | Verify DNS goes through Tor |
| **Prometheus metrics** | metrics.enabled | — | `/metrics` + `/healthz` endpoints |
| **Circuit fingerprinting resistance** | Auto | — | Adaptive rotation based on traffic patterns |
| **Bundled User-Agent list** | configs/useragents.yaml | — | Rotated randomly per instance, updated at release |
| **SIGHUP config reload** | Signal-based | — | Reload YAML without full restart |
| **Status dashboard** | `splitter status` | — | Live terminal UI with ANSI codes |
| **Version detection** | `splitter version` | — | Show Tor version + feature support |
| **Single static binary** | — | — | No interpreter, no shell dependencies |
| **Subcommand CLI** | Cobra | — | `run`, `status`, `test`, `version` |
| **Environment variable overrides** | `SPLITTER_*` prefix | — | Full config override via env |

---

## 6. Docker Migration

### Bash Docker

```bash
docker build -t splitter .
docker run -d -p 63536:63536 -p 63537:63537 splitter
# Entry: bash splitter.sh -i 15 -c 6 -re exit
```

### Go Docker

```bash
# Recommended: docker compose
docker compose up -d

# Manual build
docker build -t splitter .
docker run -d \
  -p 63536:63536 \
  -p 63537:63537 \
  -p 63539:63539 \
  -p 63540:63540 \
  splitter
```

### Key Docker Differences

| Aspect | Bash | Go |
|--------|------|-----|
| Build | Single-stage | Multi-stage (golang build → alpine runtime) |
| Image size | Larger (bash + all deps) | Smaller (static binary + tor + haproxy only) |
| User | root | Non-root (`splitter` user) |
| Healthcheck | None | HTTP `/status` endpoint |
| Ports | 63536, 63537 | 63536 (HTTP proxy), 63537 (SOCKS), 63539 (stats), 63540 (status/healthz) |
| Config | CMD args only | Environment variables + mounted YAML |

---

## 7. Environment Variable Overrides

The Go version supports `SPLITTER_*` environment variables. The Bash version had no
environment variable support.

```bash
# Core settings
SPLITTER_INSTANCES=10
SPLITTER_COUNTRIES=6
SPLITTER_RELAY_ENFORCE=exit
SPLITTER_PROXY_MODE=native

# New features
SPLITTER_AUTO_COUNTRIES=true
SPLITTER_STREAM_ISOLATION=true
SPLITTER_IPV6=true
SPLITTER_EXIT_REPUTATION=true

# Logging (off by default)
SPLITTER_LOG=1
SPLITTER_LOG_LEVEL=debug
```

**Priority order**: CLI flags > environment variables > YAML config file > compiled defaults.

---

## 8. Breaking Changes

### 1. Default proxy mode is native (HTTPTunnelPort)

The Go version defaults to `--proxy-mode native`, which uses Tor's built-in `HTTPTunnelPort`
instead of Privoxy. This eliminates one network hop and the Privoxy dependency.

**Migration**: If you need Privoxy, use `--proxy-mode legacy`.

### 2. Control port authentication changed to cookie auth

The Bash version used `expect` scripts with `tor --hash-password` for control port
authentication. The Go version uses `CookieAuthentication 1` and reads the
`control_auth_cookie` file.

**Migration**: No action needed. Cookie auth is more secure and automatic.

### 3. Logging is OFF by default

The Bash version logged by default (philosophy: audit trail). The Go version disables
logging by default (philosophy: no logs, no crime).

**Migration**: Use `--log` or `SPLITTER_LOG=1` to enable logging.

### 4. HAProxy stats port

The `func/` version used port 63537 for both the SOCKS proxy and stats page. The Go
version separates these:
- 63536: HTTP proxy frontend
- 63537: SOCKS proxy frontend
- 63539: HAProxy stats page
- 63540: Status/healthz endpoint

**Migration**: Update any scripts or monitoring that referenced the old stats port.

### 5. No shell dependency

The Go version is a single static binary. `expect`, `proxychains`, `netstat`, `shuf`, and
`bash` are no longer needed.

**Migration**: Remove these from your Docker images or deployment scripts.

### 6. Subcommand structure

The Bash version was invoked as `bash splitter.sh [flags]`. The Go version uses
subcommands:

```bash
# Old
bash splitter.sh -i 15 -c 6 -re exit

# New
./splitter run --instances 15 --countries 6 --relay-enforce exit
# Short flags still work:
./splitter run -i 15 -c 6 -re exit
```

**Migration**: Update any wrapper scripts to use `splitter run` with the new flag format.

---

## 9. Configuration Profiles

The Go version introduces profiles that preset multiple configuration values. Use
`--profile` instead of specifying individual flags:

| Profile | Instances | Countries | Relay Mode | Rotation | Special |
|---------|-----------|-----------|------------|----------|---------|
| `balanced` | 2 | 6 | entry | 120s | Default equivalent to old behavior |
| `stealth` | 5 | 12 | entry | 60s | Sandbox on, connection padding, no speed mode |
| `streaming` | 3 | 4 | speed | 300s | Conflux + congestion control, leastconn |
| `pentest` | 8 | 15 | exit | 30s | Aggressive rotation, stream isolation, random UA |

---

## 10. Common Migration Scenarios

### "I just want the same behavior as before"

```bash
./splitter run --profile balanced
```

### "I was using `-i 15 -c 6 -re exit`"

```bash
./splitter run -i 15 -c 6 -re exit
# or
./splitter run --instances 15 --countries 6 --relay-enforce exit
```

### "I need Privoxy (my Tor version is < 0.4.8)"

```bash
./splitter run --proxy-mode legacy
```

### "I want to use the old settings.cfg values as a starting point"

1. Copy `configs/default.yaml`
2. Adjust values per the mapping in `configs/SETTINGS_MAP.md`
3. Run `./splitter run --config /path/to/my-config.yaml`

### "I was running in Docker"

```bash
docker compose up -d
# Customize via environment variables:
SPLITTER_INSTANCES=10 SPLITTER_COUNTRIES=6 docker compose up -d
```
