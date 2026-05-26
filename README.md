# SPLITTER

Go-based tool that creates and manages multiple Tor network instances load-balanced via HAProxy, with geolocation-based anti-correlation rules for relay selection. Each Tor instance is configured to enforce specific countries for entry or exit nodes, making traffic analysis and de-anonymization attacks significantly harder.

Licensed under the BSD License. Created by Rener Alberto (aka Gr1nch) -- DcLabs Security Team. The user accepts total responsibility for their actions while using this tool.

---

## Architecture

The TCP stream path depends on the proxy mode:

**Native mode (recommended, Tor 0.4.8+):**
```
User -> HAProxy -> Tor (HTTPTunnelPort) -> Tor Network -> Destination
```

**Legacy mode:**
```
User -> HAProxy -> Privoxy -> Tor -> Tor Network -> Destination
```

Native mode eliminates the Privoxy hop entirely, reducing latency and removing a dependency. HAProxy backends point directly at Tor's built-in HTTP CONNECT proxy listeners.

![SPLITTER - TCP STREAM PATH](Doc/01_TCP_STREAM_PATH.png)

---

## Quick Start

### Build

```bash
go build -o splitter .
```

### Run

```bash
# Default: 2 instances/country, 6 countries, entry enforcement
./splitter run

# Custom settings
./splitter run -i 3 -c 8 -r exit

# With a profile
./splitter run --profile stealth

# With bridges (censored networks)
./splitter run --bridge-type snowflake
```

### Docker

There are two compose files included:

- docker-compose.yml — pulls the published GHCR image (recommended for users).
- docker-compose.dev.yml — builds the image locally and mounts configs for development.

Run the published image (pulls ghcr.io/millaguie/splitter:v2.0.0-RC1 as configured):

```bash
docker compose up -d
```

Run the development compose (builds from source and mounts configs):

```bash
docker compose -f docker-compose.dev.yml up --build
```

Ports: `63536` (SOCKS), `63537` (HTTP), `63539` (stats), `63540` (status/healthz).

---

## Using the Proxy

### Ports & Endpoints

| Port | Protocol | Description |
|------|----------|-------------|
| **63536** | SOCKS5 | SOCKS5 proxy — point any SOCKS5-capable application here |
| **63537** | HTTP CONNECT | HTTP proxy — point your browser here |
| **63539** | HTTP | HAProxy stats dashboard (password shown in startup banner) |
| **63540** | HTTP | Status API (`/status`) and health check (`/healthz`) |
| **63541+** | TCP | Per-instance SOCKS ports (4999, 5000, ...) — internal use only |

### Browser Configuration (HTTP proxy)

This is the simplest option. All browser traffic goes through SPLITTER.

**Firefox / Chrome / Edge:**

1. Open Settings → Network / Proxy
2. Select "Manual proxy configuration"
3. Set:
   - **HTTP proxy**: `localhost` port `63537`
   - **SSL proxy**: `localhost` port `63537`
   - (also check "Use same proxy for all protocols")

**Verify it works:**
```bash
# Should show a Tor exit IP, not your real IP
curl -x http://localhost:63537 https://check.torproject.org/api/ip
```

**Note:** The HTTP proxy works with HTTPS sites because Tor's HTTPTunnelPort uses the CONNECT method (tunneling, not MITM). Your TLS connection remains end-to-end to the destination server.

### Browser Configuration (SOCKS5 proxy)

More control — applications that support SOCKS5 can use this port directly.

**Firefox:**

1. Open Settings → General → Network Settings
2. Click "Settings…" next to "Use a proxy"
3. Select "Manual proxy configuration"
4. Set:
   - **SOCKS Host**: `localhost`
   - **SOCKS Port**: `63536`
   - **SOCKS v5**: checked
5. Select "SOCKS v5" for DNS resolution through Tor

**curl:**
```bash
curl -x socks5h://localhost:63536 https://check.torproject.org/api/ip
```

**Python (requests):**
```python
import requests

proxies = {
    "http": "socks5h://localhost:63536",
    "https": "socks5h://localhost:63536",
}
r = requests.get("https://check.torproject.org/api/ip", proxies=proxies)
print(r.json())
```

**Docker usage note:** When running in Docker, replace `localhost` with your host's IP or `host.docker.internal` (macOS/Windows).

### Other Tools

**System-wide proxy (Linux):**
```bash
export http_proxy=http://localhost:63537
export https_proxy=http://localhost:63537
export HTTP_PROXY=http://localhost:63537
export HTTPS_PROXY=http://localhost:63537

# Test
curl https://check.torproject.org/api/ip
```

**wget:**
```bash
wget -e use_proxy=yes -e http_proxy=http://localhost:63537 https://example.com
```

**Git:**
```bash
git config --global http.proxy http://localhost:63537
git config --global https.proxy http://localhost:63537
```

### HAProxy Stats Dashboard

The stats page shows real-time backend health, request counts, and error rates.

```bash
# Get password from startup logs
docker logs splitter-dev 2>&1 | grep "HAProxy stats:"
# Example output: HAProxy stats: 0.0.0.0:63539/splitter_status (password: xYzAbC123)

# Open in browser
# http://localhost:63539/splitter_status
```

### Status & Health Check

```bash
# JSON status with instance count, Tor version, features
curl http://localhost:63540/status | python3 -m json.tool

# Health check (returns 200 when all instances are ready)
curl -sf http://localhost:63540/healthz
```

### Verifying It Works

```bash
# 1. Check that traffic goes through Tor
curl -x http://localhost:63537 https://check.torproject.org/api/ip
# Expected: {"IsTor":true,"IP":"45.x.x.x"}

# 2. Verify IP rotation (multiple requests should return different IPs)
for i in $(seq 1 6); do
  curl -s -x http://localhost:63537 https://check.torproject.org/api/ip
  sleep 1
done

# 3. Check your real IP (for comparison — should NOT match Tor exit IPs)
curl -s https://check.torproject.org/api/ip

# 4. DNS leak test
./splitter test dns

# 5. Exit node reputation check
./splitter test exit-reputation
```

---

## Environment Variables

All configuration values can be overridden via environment variables with the `SPLITTER_` prefix. Variables override YAML config file defaults but are overridden by CLI flags.

### Core

| Variable | Default | Description |
|----------|---------|-------------|
| `SPLITTER_INSTANCES` | `2` | Tor instances per country |
| `SPLITTER_COUNTRIES` | `6` | Number of countries to select |
| `SPLITTER_RELAY_ENFORCE` | `entry` | Relay mode: `entry`, `exit`, `speed` |
| `SPLITTER_PROXY_MODE` | `native` | Proxy mode: `native` or `legacy` |
| `SPLITTER_PROFILE` | `""` | Configuration profile: `stealth`, `balanced`, `streaming`, `pentest` |

### Features

| Variable | Default | Description |
|----------|---------|-------------|
| `SPLITTER_LOG` | `0` | Enable logging (1 = on) |
| `SPLITTER_LOG_LEVEL` | `info` | Log level: `debug`, `info`, `warn`, `error` |
| `SPLITTER_AUTO_COUNTRIES` | `0` | Auto-fetch country list from Tor Metrics API |
| `SPLITTER_STREAM_ISOLATION` | `0` | Enable stream isolation via SOCKS5 auth |
| `SPLITTER_IPV6` | `0` | Enable IPv6 dual-stack relay selection |
| `SPLITTER_EXIT_REPUTATION` | `0` | Check exit node reputation via Onionoo API |
| `SPLITTER_METRICS` | `0` | Enable Prometheus metrics endpoint |
| `SPLITTER_BRIDGE_TYPE` | `none` | Bridge type: `snowflake`, `webtunnel`, `obfs4`, `none` |
| `SPLITTER_COUNTRY_INTERVAL` | `120` | Country rotation interval in seconds |

### Ports

| Variable | Default | Description |
|----------|---------|-------------|
| `SPLITTER_SOCKS_PORT` | `63536` | SOCKS5 proxy listen port |
| `SPLITTER_HTTP_PORT` | `63537` | HTTP proxy listen port |
| `SPLITTER_STATS_PORT` | `63539` | HAProxy stats page port |
| `SPLITTER_STATUS_PORT` | `63540` | Status/healthz HTTP port |

### Docker-Specific

```bash
docker run -d --name splitter \
  -p 63536:63536 \
  -p 63537:63537 \
  -p 63539:63539 \
  -p 63540:63540 \
  -e SPLITTER_INSTANCES=2 \
  -e SPLITTER_COUNTRIES=6 \
  -e SPLITTER_RELAY_ENFORCE=exit \
  -e SPLITTER_LOG=1 \
  splitter
```

> **Security note:** Do NOT expose ports 63536-63537 publicly. These are unauthenticated proxies. Use Docker's port mapping to bind to `127.0.0.1` only, or restrict access with a firewall. Exposing them on `0.0.0.0` means anyone who can reach your server can use your Tor exit as an open proxy.

To bind to localhost only:

```bash
docker run -d --name splitter \
  -p 127.0.0.1:63536:63536 \
  -p 127.0.0.1:63537:63537 \
  -p 127.0.0.1:63539:63539 \
  -p 127.0.0.1:63540:63540 \
  -e SPLITTER_INSTANCES=2 \
  -e SPLITTER_COUNTRIES=6 \
  -e SPLITTER_RELAY_ENFORCE=exit \
  splitter
```

Or with docker-compose.yml:

```yaml
services:
  splitter:
    image: ghcr.io/millaguie/splitter:v2.0.0-beta-02
    ports:
      - "127.0.0.1:63536:63536"
      - "127.0.0.1:63537:63537"
      - "127.0.0.1:63539:63539"
      - "127.0.0.1:63540:63540"
```

### Custom Configuration

Mount your own config file to override defaults:

```bash
docker run -d --name splitter \
  -v ./my-config.yaml:/splitter/configs/default.yaml:ro \
  -p 63536:63536 -p 63537:63537 -p 63539:63539 -p 63540:63540 \
  splitter
```

---

## CLI Reference

### Commands

| Command | Description |
|---------|-------------|
| `splitter run` | Start SPLITTER with Tor instances and HAProxy |
| `splitter status` | Show live dashboard of instances, countries, circuits |
| `splitter test dns` | Run DNS leak test through Tor |
| `splitter test exit-reputation` | Check exit node reputation |
| `splitter version` | Show version and detected Tor features |

### Flags

| Flag | Short | Default | Description |
|------|-------|---------|-------------|
| `--instances` | `-i` | 2 | Tor instances per country |
| `--countries` | `-c` | 6 | Number of countries to select |
| `--relay-enforce` | `-r` | entry | Relay mode: entry, exit, speed |
| `-re` | -- | -- | Legacy alias for `--relay-enforce` |
| `--profile` | -- | "" | Configuration profile: stealth, balanced, streaming, pentest |
| `--proxy-mode` | -- | native | Proxy mode: native (HTTPTunnelPort), legacy (Privoxy) |
| `--bridge-type` | -- | none | Bridge type: snowflake, webtunnel, obfs4, none |
| `--verbose` | -- | false | Enable verbose output |
| `--log` | -- | false | Enable logging (off by default: no logs, no crime) |
| `--log-level` | -- | info | Log level: debug, info, warn, error |
| `--auto-countries` | -- | false | Auto-fetch country list from Tor Metrics API |
| `--stream-isolation` | -- | false | Enable stream isolation via SOCKS5 auth |
| `--ipv6` | -- | false | Enable IPv6 dual-stack relay selection |
| `--exit-reputation` | -- | false | Check exit node reputation via Onionoo API |

---

## Configuration Profiles

Predefined profiles for common use cases. Set with `--profile <name>`.

### stealth

Maximum security. Aggressive rotation, many instances, all hardening enabled.

| Parameter | Value |
|-----------|-------|
| Instances/country | 3 |
| Countries | 8 |
| Relay enforce | entry |
| Circuit rotation | 10s |
| Load balance | roundrobin |
| Conflux | enabled |
| Congestion control | enabled |
| Connection padding | enabled |
| Sandbox | enabled |
| Circuit fingerprinting resistance | enabled |
| Logging | off |

### balanced

Good tradeoff between security and performance. Default-like behavior.

| Parameter | Value |
|-----------|-------|
| Instances/country | 2 |
| Countries | 6 |
| Relay enforce | exit |
| Circuit rotation | 15s |
| Load balance | roundrobin |
| Congestion control | enabled |
| Logging | off |

### streaming

Optimized for throughput and media consumption.

| Parameter | Value |
|-----------|-------|
| Instances/country | 1 |
| Countries | 4 |
| Relay enforce | speed |
| Circuit rotation | 300s |
| Load balance | leastconn |
| Conflux | enabled |
| Congestion control | enabled |
| IPv6 | enabled |
| Logging | off |

### pentest

Extreme rotation and randomization for penetration testing scenarios.

| Parameter | Value |
|-----------|-------|
| Instances/country | 5 |
| Countries | 10 |
| Relay enforce | exit |
| Circuit rotation | 10s |
| Load balance | roundrobin |
| Stream isolation | enabled |
| Circuit fingerprinting resistance | enabled |
| Exit reputation | enabled |
| Logging | DEBUG |

---

## Docker Usage

### docker-compose.yml

```yaml
version: "3.8"

services:
  splitter:
    image: ghcr.io/millaguie/splitter:v2.0.0-beta-02
    ports:
      - "63536:63536"   # SOCKS5 proxy
      - "63537:63537"   # HTTP proxy
      - "63539:63539"   # HAProxy stats
      - "63540:63540"   # Status / healthz
    environment:
      - SPLITTER_INSTANCES=2
      - SPLITTER_COUNTRIES=6
      - SPLITTER_RELAY_ENFORCE=exit
      - SPLITTER_LOG=1
    volumes:
      - ./configs:/splitter/configs:ro
      - splitter-data:/splitter/data
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-sf", "http://localhost:63540/status"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 60s

volumes:
  splitter-data:
```

The Docker image uses a multi-stage build: Go compilation in `golang:1.23-alpine`, runtime in `alpine:3.21` with Tor, HAProxy, and Privoxy installed. Runs as a non-root user.

---

## Modern Tor Features

SPLITTER auto-detects Tor version at startup and enables features conditionally:

- **Conflux** (Tor 0.4.8+) -- Multi-leg circuits that split traffic across multiple paths simultaneously, multiplying throughput and resilience
- **HTTPTunnelPort** (Tor 0.4.8+) -- Native HTTP CONNECT proxy, eliminating the need for Privoxy
- **Congestion Control** (Tor 0.4.7+) -- Dramatically improves throughput on long-distance circuits
- **Post-Quantum Key Exchange** (Tor 0.4.8.17+ with OpenSSL 3.5.0+) -- ML-KEM768 protection against harvest-now-decrypt-later attacks
- **CGO Encryption** (Tor 0.4.9+) -- Counter Galois Onion relay cryptography with improved resistance to tagging attacks
- **Bridge Support** -- Snowflake, WebTunnel, and obfs4 pluggable transports for censored networks (`--bridge-type`)
- **Sandboxing** -- seccomp-bpf sandbox via `Sandbox 1` in generated torrc
- **Circuit Fingerprinting Resistance** -- Adaptive circuit rotation based on traffic patterns, defeating timing correlation attacks
- **Exit Node Reputation** -- Checks exit relay flags, uptime, and bandwidth via Onionoo API before use
- **DNS Leak Testing** -- Verifies all DNS queries go exclusively through Tor (`splitter test dns`)
- **Prometheus Metrics** -- `/metrics` and `/healthz` HTTP endpoints for monitoring and alerting
- **Stream Isolation** -- Per-destination circuit separation via SOCKS5 auth (`IsolateSOCKSAuth`)
- **IPv6 Dual-Stack** -- `ClientUseIPv6 1` for broader relay selection

---

## Building from Source

```bash
# Standard build
go build -o splitter .

# Optimized binary (smaller, no debug info)
go build -ldflags="-s -w" -o splitter .

# Cross-compile
GOOS=linux GOARCH=arm64 go build -ldflags="-s -w" -o splitter .
```

### Dependencies

Runtime dependencies must be in `$PATH`:
- `tor` -- Tor standalone client
- `haproxy` -- HAProxy load balancer
- `privoxy` -- Only required in legacy proxy mode

Notes about GHCR image and authentication:

- The default docker-compose.yml references the published GHCR image ghcr.io/millaguie/splitter:v2.0.0-RC1. If that package is public you can pull it without authentication. If GHCR authentication is required for this repository, login with a Personal Access Token (PAT) that has `read:packages`:

```bash
echo "${GHCR_PAT}" | docker login ghcr.io -u <your-username> --password-stdin
```

The release workflow sets the binary version at build-time via ldflags (cmd.Version). The Release workflow is triggered by pushing tags like `v2.0.0-RC1` and will build binaries and push container images to GHCR.

SPLITTER detects available binaries and Tor version at startup, failing fast with a clear message if anything is missing.

---

## Migration from Bash Version

The original Bash version is preserved in `legacy/` for reference.

| Aspect | Bash | Go |
|--------|------|----|
| Configuration | `settings.cfg` | `configs/default.yaml` |
| CLI flags | `-i`, `-c`, `-re` | Same short flags + long flags (`--instances`, `--countries`, `--relay-enforce`) |
| Circuit renewal | `expect` scripts | Native Go Tor control protocol (cookie auth) |
| Port allocation | `netstat` parsing | `net.Listen` test |
| Process management | Shell background jobs | Go process groups with graceful shutdown |
| Logging | Always on | Off by default (`--log` to enable) |
| Profiles | Manual config editing | `--profile stealth\|balanced\|streaming\|pentest` |
| Proxy mode | Privoxy always | Native HTTPTunnelPort or legacy Privoxy |

Key changes:
- No more `expect` dependency -- circuit renewal connects to the Tor control port directly in Go
- No more `netstat` / `ss` -- port allocation uses the Go standard library
- No more `proxychains` dependency
- The `-re` flag is preserved as a legacy alias for `--relay-enforce`

---

## Anti-Correlation Theory

### The Problem

Tor-related de-anonymization techniques rely on traffic analysis, correlation, and statistical attacks [1, 2, 3, 4, 5, 6, 10, 20, 23, 28]. These techniques exploit the ability to observe traffic patterns at both ends of a Tor circuit -- the user's entry point and the exit point -- and correlate them.

### SPLITTER's Approach

SPLITTER defeats these attacks through three mechanisms:

**1. Geolocation-based relay enforcement.** Each Tor instance is configured to use a specific country for either the entry node or exit node (never the same country for both). This forces an adversary to compromise nodes in multiple jurisdictions simultaneously to capture both ends of a circuit [8, 16, 21, 22, 24, 26, 27, 29].

**2. Instance lifecycle management.** Tor instances are periodically killed and restarted with fresh configurations pointing to different countries. This interrupts long-lived TCP streams and prevents an adversary from accumulating enough traffic data over time to perform meaningful correlation [1, 2, 3, 4, 5, 6, 10, 20, 23, 28].

**3. Randomized circuit rotation.** SPLITTER introduces random intervals in circuit creation to avoid predictable timing patterns that could be exploited by timing correlation attacks.

### Relay Enforcement Modes

| Mode | Behavior | Use Case |
|------|----------|----------|
| **entry** | Enforces a specific country as entry node; exit is random from a different country | Maximum security (default) |
| **exit** | Enforces a specific country as exit node; entry is random from a different country | GeoIP bypass, geographic control of exit |
| **speed** | Enforces the same country for entry, middle, and exit nodes | Maximum throughput, restricted circuit geography |

#### Entry Mode

The load balancing algorithm is round-robin. For a given country enforced as the entry node, SPLITTER selects a different random country for the exit node, ensuring the two never overlap. This controls Tor's normally free random selection of relay countries [8, 27].

#### Exit Mode

Gives the user control over which country the destination server sees as the traffic origin. Suitable for bypassing GeoIP restrictions [29]. Specific use cases:

- **Fixed country**: Set countries to 1 and include only the desired country. Adjust instances per country for stability.
- **Random countries**: Set the desired number of countries. Each instance rotates through a random selection.

#### Speed Mode

All three relays (entry, middle, exit) are constrained to the same country. This minimizes the geographic distance packets must traverse, maximizing throughput. The first anti-correlation rule is relaxed but still observed [8].

### Instance Lifecycle

The total number of simultaneous active Tor instances is:

**(_X_ instances per country) x (_Y_ countries) = _total instances_**

Each instance follows this lifecycle:

1. SPLITTER selects a random country and writes a torrc configuration file enforcing the selected relay mode
2. The Tor process starts and creates circuits following SPLITTER's rules
3. Random jitter is applied to circuit creation intervals to avoid timing patterns
4. When the instance lifetime expires, SPLITTER kills the process, deletes temporary files, and restarts with a new country

![SPLITTER - TOR INSTANCE LIFE CIRCLE](Doc/03_INSTANCE_LIFECIRCLE.png)

### HAProxy Health Checking

SPLITTER uses HAProxy to perform health checks on each Tor instance before routing traffic through it. A specific website is checked at configurable intervals. If a circuit fails to respond or the exit node cannot resolve the requested address, the instance is marked down and traffic is routed to another instance.

The order of instances in the HAProxy configuration is randomized to prevent consecutive requests from going through the same country when multiple instances share a country.

![SPLITTER - HAPROXY HEALTH CHECK](Doc/04_HAPROXY_HEALTH_CHECK.png)

![SPLITTER - LOAD BALANCE OVERVIEW](Doc/02_LOADBALANCE_OVERVIEW.png)

---

## SPLITTER NETWORK

For maximum effectiveness of the anti-correlation approach, a low-cost private VPS and VPN chain should be considered. This globally distributed network infrastructure makes traffic analysis harder and prevents direct association between the Tor network and the user.

### Architecture

The user connects to a VPS via VPN. The VPS runs SPLITTER inside a Docker container and routes all outbound traffic through a public VPN service before entering the Tor network.

![SPLITTER NETWORK - TCP STREAM PATH](Doc/05_SPLITTER_NETWORK_TCP_STREAM_PATH.png)

### Components

**1. The VPS acts as both VPN server and VPN client:**

- The user connects to the VPS through a VPN service. All user traffic is forwarded to the VPS. The user points their browser to the HAProxy port exposed by the Docker container.
- The VPS is also connected to a public VPN service. All outbound traffic from the VPS uses this connection, so Tor connections originate from the public VPN's IP address [38, 39, 40].

**2. VPS firewall prevents leaks:**

- Inbound: Only VPN server traffic is allowed. All other inbound traffic is blocked.
- Outbound: Only DNS resolution for the public VPN, connections to the public VPN service, and HAProxy port traffic are allowed. The user's only outbound route is through Tor inside the Docker container.

**3. Docker container isolation:**

SPLITTER runs inside a Docker container. The public VPN connection is established at the VPS level and transferred to the container so it becomes the default gateway. The container exposes only the HAProxy port.

If an adversary compromises the container, they are trapped inside it with no route to the VPS operating system or the connected user [44, 45].

### Global Scale

Multiple VPS instances can be distributed globally using different providers and VPN services. Each runs SPLITTER in a container connected to a different VPN provider. An additional HAProxy layer can load-balance across all VPS nodes, further distributing traffic and increasing correlation difficulty.

![SPLITTER NETWORK - OVERVIEW](Doc/06_SPLITTER_NETWORK_OVERVIEW.png)

---

## References

- [1] Sambuddho Chakravarty, Marco V. Barbera, Georgios Portokalidis, Michalis Polychronakis and Angelos D. Keromytis -- "On the Effectiveness of Traffic Analysis Against Anonymity Networks Using Flow Records".
  https://mice.cs.columbia.edu/getTechreport.php?techreportID=1545&format=pdf

- [2] Nathan S. Evans, Roger Dingledine and Christian Grothoff -- "A Practical Congestion Attack on Tor Using Long Paths".
  https://www.usenix.org/legacy/event/sec09/tech/full_papers/evans.pdf

- [3] Matthew Wright, Micah Adler, Brian N. Levine and Clay Shields -- "An analysis of the degradation of anonymous protocols".
  http://people.cs.georgetown.edu/~clay/research/pubs/wright.ndss01.pdf

- [4] Nicholas Hopper, Eeugene Y. Vasserman, and Eric Chan-Tin -- "How Much Anonymity does Network Latency Leak?".
  https://www-users.cs.umn.edu/~hoppernj/tissec-latency-leak.pdf

- [5] Sebastian Zander and Steven J. Murdoch -- "An Improved Clock-skew Measurement Technique for Revealing Hidden Services".
  https://www.usenix.org/legacy/event/sec08/tech/full_papers/zander/zander.pdf

- [6] Kevin Bauer, Damon McCoy, Dirk Grunwald, Tadayoshi Kohno and Douglas Sicker -- "Low-Resource Routing Attacks Against Anonymous Systems".
  http://www.cs.colorado.edu/department/publications/reports/docs/CU-CS-1025-07.pdf

- [7] TOR project official web site. https://www.torproject.org/

- [8] TOR project overview. https://www.torproject.org/about/overview.html.en

- [9] "Statistical Analysis Handbook". http://www.statsref.com/StatsRefSample.pdf

- [10] Steven J. Murdoch and George Danezis -- "Low-Cost Traffic Analysis of Tor".
  https://murdoch.is/papers/oakland05torta.pdf

- [11] FBI Official web site -- "Dozens of Online 'Dark Markets' Seized Pursuant to Forfeiture Complaint Filed in Manhattan Federal Court in Conjunction with the Arrest of the Operator of Silk Road 2.0".
  https://www.fbi.gov/contact-us/field-offices/newyork/news/press-releases/dozens-of-online-dark-markets-seized-pursuant-to-forfeiture-complaint-filed-in-manhattan-federal-court-in-conjunction-with-the-arrest-of-the-operator-of-silk-road-2.0

- [12] FBI Official web site -- "Operator of Silk Road 2.0 Website Charged in Manhattan Federal Court".
  https://www.fbi.gov/contact-us/field-offices/newyork/news/press-releases/operator-of-silk-road-2.0-website-charged-in-manhattan-federal-court

- [13] FBI Special Agent: Thomas M. Dalton report about the hoax bomb in Harvard University resulting in the prison of Eldo Kim.
  https://cbsboston.files.wordpress.com/2013/12/kimeldoharvard.pdf

- [13.1] FBI Official web site -- "Harvard Student Charged with Bomb Hoax".
  https://archives.fbi.gov/archives/boston/press-releases/2013/harvard-student-charged-with-bomb-hoax

- [14] FBI Official web site -- "Six Hackers in the United States and Abroad Charged for Crimes Affecting Over One Million Victims".
  https://archives.fbi.gov/archives/newyork/press-releases/2012/six-hackers-in-the-united-states-and-abroad-charged-for-crimes-affecting-over-one-million-victims

- [15] Adrian Crenshaw -- "Dropping Docs on Darknets: How People Got Caught - Defcon 22".
  https://www.youtube.com/watch?v=7G1LjQSYM5Q

- [16] TOR Official Project web site -- Metrics about TOR network.
  https://metrics.torproject.org/networksize.html

- [17] TOR Official Project web site -- "Tor: Onion Service Protocol".
  https://www.torproject.org/docs/onion-services.html.en

- [18] Steven J. Murdoch -- "Hot or Not: Revealing Hidden Services by their Clock Skew".
  https://murdoch.is/papers/ccs06hotornot.pdf

- [19] TOR Official Project web site -- "Who Uses Tor?".
  https://www.torproject.org/about/torusers.html.en

- [20] Rob Jansen, Marc Juarez, Rafa Galvez, Tariq Elahi and Claudia Diaz -- "Inside Job: Applying Traffic Analysis to Measure Tor from Within".
  https://www.robgjansen.com/publications/insidejob-ndss2018.pdf

- [21] TOR project official web site -- FAQ: "What are Entry Guards?".
  https://www.torproject.org/docs/faq#EntryGuards

- [22] TOR project official blog -- "Improving Tor's anonymity by changing guard parameters".
  https://blog.torproject.org/improving-tors-anonymity-changing-guard-parameters

- [23] Free Haven -- Online Anonymity Papers Library.
  https://www.freehaven.net/anonbib/

- [24] TOR project official blog -- "Research problem: better guard rotation parameters".
  https://blog.torproject.org/research-problem-better-guard-rotation-parameters

- [25] Nick Mathewson -- "Cryptographic Challenges in and around Tor".
  https://crypto.stanford.edu/RealWorldCrypto/slides/tor.pdf

- [26] TOR project official web site -- FAQ: "How often does Tor change its paths?".
  https://www.torproject.org/docs/faq#ChangePaths

- [27] TOR project official web site -- TOR MANUAL.
  https://www.torproject.org/docs/tor-manual.html.en

- [28] Milad Nasr, Amir Houmansadr and Arya Mazumdar -- "Compressive Traffic Analysis: A New Paradigm for Scalable Traffic Analysis".
  https://people.cs.umass.edu/~milad/papers/compress_CCS.pdf

- [29] ISACA -- "Geolocation: Risk, Issues and Strategies".
  https://www.isaca.org/Groups/Professional-English/wireless/GroupDocuments/Geolocation_WP.pdf

- [30] Eugene Gorelik -- "Cloud Computing Models".
  https://web.mit.edu/smadnick/www/wp/2013-01.pdf

- [31] Alexa Huth and James Cebula -- "The Basics of Cloud Computing".
  https://www.us-cert.gov/sites/default/files/publications/CloudComputingHuthCebula.pdf

- [32] Jason A. Donenfeld -- "WireGuard: Next Generation Kernel Network Tunnel".
  https://www.wireguard.com/papers/wireguard.pdf

- [33] HAProxy -- Official Web Site. https://www.haproxy.org/

- [34] Privoxy -- Official Web Site. http://www.privoxy.org/

- [35] TOR Standalone Linux version Download Page.
  https://www.torproject.org/download/download-unix.html.en

- [36] HAProxy -- Documentation. https://www.haproxy.org/#docs

- [37] Privoxy -- Official User Manual. http://www.privoxy.org/user-manual/index.html

- [38] King, Kevin, "Personal Jurisdiction, Internet Commerce, and Privacy: The Pervasive Legal Consequences of Geolocation Technologies," Albany Law Journal of Science and Technology, January 2011.

- [39] Viviane Reding -- "Digital Sovereignty: Europe at a Crossroads".
  http://institute.eib.org/wp-content/uploads/2016/01/Digital-Sovereignty-Europe-at-a-Crossroads.pdf

- [40] Tim Maurer, Robert Morgus, Isabel Skierka, Mirko Hohmann -- "Technological Sovereignty: Missing the Point?".
  http://www.digitaldebates.org/fileadmin/media/cyber/Maurer-et-al_2014_Tech-Sovereignty-Europe.pdf

- [41] BSD License Definition. http://www.linfo.org/bsdlicense.html

- [42] Joel Reardon and Ian Goldberg -- "Improving Tor using a TCP-over-DTLS Tunnel".
  https://www.usenix.org/legacy/event/sec09/tech/full_papers/reardon.pdf

- [43] TOR Metrics -- Official Web Site.
  https://metrics.torproject.org/rs.html#search/country:ES%20flag:exit

- [44] Docker -- Official Documentation. https://docs.docker.com/

- [45] Docker -- Official Documentation "Expose (incoming ports)".
  https://docs.docker.com/engine/reference/run/#expose-incoming-ports

---

## License

BSD License [41]. Do whatever you want with this tool, but take the responsibility.
