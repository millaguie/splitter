# Settings.cfg to default.yaml Mapping

This document maps every variable found in `legacy/func/settings.cfg` (primary) and
`legacy/settings.cfg` (root variant) to their Go equivalents in `configs/default.yaml`.
This drives task 3.1.2 (Go config system implementation).

**Legend:**
- **Source**: `func` = `legacy/func/settings.cfg` (canonical, sourced by splitter.sh)
- **Source**: `root` = `legacy/settings.cfg` (variant, not directly sourced)
- **Decision**: PORTED = mapped to YAML; DROPPED = removed with justification; RUNTIME = Go runtime
- Differences between the two files are noted in the "Notes" column

---

## 1. Instance and CLI Arguments

These are set at runtime by `user_start_input.func` via CLI flags, not in settings.cfg.

| Bash Variable | Go YAML Key | Type | Default | Decision | Notes |
|---------------|-------------|------|---------|----------|-------|
| TOR_INSTANCES | instances.per_country | int | 2 | PORTED | Set via `-i` / `--instances` CLI flag |
| COUNTRIES | instances.countries | int | 6 | PORTED | Set via `-c` / `--countries` CLI flag |
| COUNTRY_LIST_CONTROLS | relay.enforce | string | "entry" | PORTED | Set via `-re` / `--relay-enforce`. Values: entry, exit, speed |
| LOAD_BALANCE_ALGORITHM | proxy.load_balance_algorithm | string | "roundrobin" | PORTED | Derived by loadbalancing_choice.func: entry/exit->roundrobin, speed->leastconn |
| HAPROXY_HTTP_REUSE | proxy.haproxy_http_reuse | string | "never" | PORTED | Derived: entry/exit->never, speed->safe |

---

## 2. Country Configuration

| Bash Variable | Go YAML Key | Type | Default (func) | Default (root) | Decision | Notes |
|---------------|-------------|------|----------------|----------------|----------|-------|
| MY_COUNTRY_LIST | country.selected | string | "RANDOM" | "RANDOM" | PORTED | "RANDOM" or comma-separated `{XX}` codes |
| ACCEPTED_COUNTRIES | country.accepted | list | 32 countries (see YAML) | same | PORTED | Converted from comma-separated `{XX}` string to YAML list |
| BLACKLIST_COUNTRIES | country.blacklisted | list | 67 entries (see YAML) | same | PORTED | Converted from comma-separated `{XX}` string to YAML list |
| CHANGE_COUNTRY_ONTHEFLY | country.rotation.enabled | bool | "YES" | "YES" | PORTED | String "YES"/"NO" mapped to bool |
| CHANGE_COUNTRY_INTERVAL | country.rotation.interval | int | 120 | 120 | PORTED | Seconds between country changes |
| TOTAL_COUNTRIES_TO_CHANGE | country.rotation.total_to_change | int | 10 | 10 | PORTED | Number of countries rotated per cycle |

---

## 3. Retry and Timeout

| Bash Variable | Go YAML Key | Type | Default (func) | Default (root) | Decision | Notes |
|---------------|-------------|------|----------------|----------------|----------|-------|
| RETRIES | instances.retries | int | 1000 | 100 | PORTED | **Values differ between files**. func=1000, root=100. Using func value. |
| MINIMUM_TIMEOUT | tor.minimum_timeout | int | 15 | 20 | PORTED | **Values differ**. Base for derived timeouts. func=15, root=20. |
| MAX_CONCURRENT_REQUEST | instances.max_concurrent_requests | int | 20 | 20 | PORTED | HAProxy backend maxconn per instance |

---

## 4. Port Allocation

| Bash Variable | Go YAML Key | Type | Default (func) | Default (root) | Decision | Notes |
|---------------|-------------|------|----------------|----------------|----------|-------|
| START_SOCKS_PORT | tor.start_socks_port | int | 4999 | 4999 | PORTED | First Tor SOCKS port, incremented per instance |
| START_CONTROL_PORT | tor.start_control_port | int | 5999 | 5999 | PORTED | First Tor control port, incremented per instance |
| START_DNS_PORT | tor.start_dns_port | int | — | 5299 | PORTED | Only in root version. Go port allocator manages this. |
| TOR_START_HTTP_PORT | tor.start_http_port | int | — | 5199 | PORTED | Only in root version. Go port allocator manages this. |
| TOR_START_TransPort | tor.start_transport_port | int | — | 5099 | PORTED | Only in root version. Go port allocator manages this. |
| PRIVOXY_START_PORT | privoxy.start_port | int | 6999 | 6999 | PORTED | First Privoxy port, incremented per instance |
| HIDDEN_START_PORT | tor.hidden_service.start_port | int | 3999 | — | PORTED | Only in func version. First hidden service port. |

---

## 5. Binary Paths

| Bash Variable | Go YAML Key | Type | Default (func) | Default (root) | Decision | Notes |
|---------------|-------------|------|----------------|----------------|----------|-------|
| TORPATH | tor.binary_path | string | "/usr/bin/tor" | "/usr/local/bin/tor" | PORTED | **Values differ**. Go will auto-detect from $PATH if not set. |
| HAPROXY_PATH | haproxy.binary_path | string | "/usr/sbin/haproxy" | "/usr/sbin/haproxy" | PORTED | Go will auto-detect from $PATH if not set. |
| PRIVOXY_PATH | privoxy.binary_path | string | "/usr/sbin/privoxy" | "/usr/sbin/privoxy" | PORTED | Only used in legacy proxy mode. Go auto-detects. |

---

## 6. User and Identity

| Bash Variable | Go YAML Key | Type | Default | Decision | Notes |
|---------------|-------------|------|---------|----------|-------|
| USER_ID | — | — | `$(id \| cut...)` | DROPPED | Go uses os/user package or auto-detects at runtime |
| USER_UID | — | — | `$(id \| sed...)` | DROPPED | Only in root version. Go uses os.Getuid() at runtime |
| USER_GID | — | — | `$(id \| sed...)` | DROPPED | Only in root version. Go uses os.Getgid() at runtime |

---

## 7. Paths and Directories

| Bash Variable | Go YAML Key | Type | Default (func) | Default (root) | Decision | Notes |
|---------------|-------------|------|----------------|----------------|----------|-------|
| TOR_TEMP_FILES | paths.temp_files | string | "/tmp/splitter" | "/tmp/splitter" | PORTED | Deleted and recreated on startup |
| HIDDEN_SERVICE_PATH | tor.hidden_service.base_path | string | "/tmp/splitter/hidden_service_" | — | PORTED | Only in func version. Instance number appended. |
| PRIVOXY_FILE | privoxy.config_file_prefix | string | "${TOR_TEMP_FILES}/privoxy_splitter_config_" | same | PORTED | Instance number appended |
| MASTER_PROXY_CFG | haproxy.config_file | string | "${TOR_TEMP_FILES}/splitter_master_proxy.cfg" | same | PORTED | Generated HAProxy config path |
| PROXYCHAINS_FILE | paths.proxychains_file | string | "${HOME}/.proxychains/proxychains.conf" | same | PORTED | Optional; may be dropped if proxychains not used in Go version |

---

## 8. Listen Addresses

| Bash Variable | Go YAML Key | Type | Default (func) | Default (root) | Decision | Notes |
|---------------|-------------|------|----------------|----------------|----------|-------|
| LISTEN_ADDR | tor.listen_addr | string | "0.0.0.0" | "0.0.0.0" | PORTED | Tor SOCKS/control port bind address |
| PRIVOXY_LISTEN | privoxy.listen | string | "0.0.0.0" | "0.0.0.0" | PORTED | Privoxy bind address |
| MASTER_PROXY_LISTEN | proxy.master.listen | string | "0.0.0.0" | "0.0.0.0" | PORTED | HAProxy frontend bind address |
| DNSDIST_SERVER_LISTEN | dns.dist_listen | string | — | "0.0.0.0" | PORTED | Only in root version. dnsdist not in func version. |
| TOR_DNS_LISTEN | dns.tor_listen | string | — | "0.0.0.0" | PORTED | Only in root version. |
| DNSDIST_SERVER_PORT | dns.dist_port | int | — | 5353 | PORTED | Only in root version. |

---

## 9. Master Proxy Ports

| Bash Variable | Go YAML Key | Type | Default (func) | Default (root) | Decision | Notes |
|---------------|-------------|------|----------------|----------------|----------|-------|
| MASTER_PROXY_PORT | proxy.master.port | int | 63536 | — | PORTED | Only in func version. Single unified port. |
| MASTER_PROXY_SOCKS_PORT | proxy.master.socks_port | int | — | 63536 | PORTED | Only in root version. Separate SOCKS port. |
| MASTER_PROXY_HTTP_PORT | proxy.master.http_port | int | — | 63537 | PORTED | Only in root version. Separate HTTP port. |
| MASTER_PROXY_TRANSPARENT_PORT | proxy.master.transparent_port | int | — | 63538 | PORTED | Only in root version. Separate transparent port. |
| MASTER_PROXY_STAT_LISTEN | proxy.stats.listen | string | "0.0.0.0" | "0.0.0.0" | PORTED | HAProxy stats page bind address |
| MASTER_PROXY_STAT_PORT | proxy.stats.port | int | 63537 | 63539 | PORTED | **Values differ**. func=63537, root=63539. |
| MASTER_PROXY_STAT_URI | proxy.stats.uri | string | "/splitter_status" | "/splitter_status" | PORTED | HAProxy stats page URL path |
| MASTER_PROXY_STAT_PWD | proxy.stats.password | string | "${RAND_PASS}" | "${RAND_PASS}" | PORTED | Auto-generated random password at startup |

---

## 10. Passwords (Runtime-Generated)

| Bash Variable | Go YAML Key | Type | Default | Decision | Notes |
|---------------|-------------|------|---------|----------|-------|
| RAND_PASS | — | — | `$(dd if=/dev/urandom...)` | DROPPED | Go generates random password at runtime using crypto/rand |
| TORPASS | — | — | `$(tor --hash-password...)` | DROPPED | Go uses cookie auth by default, or hashes password at runtime |
| MASTER_PROXY_PASSWORD | — | — | undefined | BUG | Referenced in pre_loading.func:78 but never defined in either settings.cfg. Go version will use auto-generated password. |

---

## 11. Logging

| Bash Variable | Go YAML Key | Type | Default (func) | Default (root) | Decision | Notes |
|---------------|-------------|------|----------------|----------------|----------|-------|
| LOGDIR | logging.dir | string | "${TOR_TEMP_FILES}" | "${TOR_TEMP_FILES}" | PORTED | Defaults to temp_files path |
| LOGNAME | logging.name_prefix | string | "tor_log_" | "tor_log_" | PORTED | Instance number appended |

---

## 12. Health Check

| Bash Variable | Go YAML Key | Type | Default (func) | Default (root) | Decision | Notes |
|---------------|-------------|------|----------------|----------------|----------|-------|
| HEALTH_CHECK_URL | health_check.url | string | "https://www.google.com/" | "www.google.com" | PORTED | **Values differ**: func includes scheme, root doesn't. Using func version. |
| HEALTH_CHECK_INTERVAL | health_check.interval | int | 12 | 3 | PORTED | **Values differ**. func=12s, root=3. Using func value. Root used bare int; func had "12s" suffix. |
| HEALTH_CHECK_MAX_FAIL | health_check.max_fail | int | 1 | 1 | PORTED | Fail count before instance marked DOWN |
| HEALTH_CHECK_MININUM_SUCESS | health_check.minimum_success | int | 1 | 1 | PORTED | Note: typo in Bash name (MININUM). Corrected in Go. |

---

## 13. User Agent

| Bash Variable | Go YAML Key | Type | Default | Decision | Notes |
|---------------|-------------|------|---------|----------|-------|
| TOR_BROWNSER_USER_AGENT | user_agent.tor_browser | string | "Mozilla/5.0 (Windows NT 6.1; rv:52.0) Gecko/20100101 Firefox/52.0" | PORTED | Outdated (2018). Go version should use bundled list updated at release time. |
| SPOOFED_USER_AGENT | — | — | derived from TOR_BROWNSER_USER_AGENT | DROPPED | Go handles escaping at runtime. Bash used sed to escape spaces. |

---

## 14. Proxy Exclusion

| Bash Variable | Go YAML Key | Type | Default (func) | Default (root) | Decision | Notes |
|---------------|-------------|------|----------------|----------------|----------|-------|
| DO_NOT_PROXY | proxy.do_not_proxy | list | 6 entries | 16 RFC1918 entries | PORTED | **Values differ significantly**. func=local network IPs; root=full RFC1918 range. Converted to YAML list. |
| INCLUDE_SECURITY_HEADERS_IN_HTTP_RESPONSE | proxy.include_security_headers | bool | "YES" | — | PORTED | Only in func version. String "YES"/"NO" mapped to bool. Set false for pentest. |

---

## 15. Tor Tuning Parameters

These map directly to torrc directives. The "Tor default" column shows what Tor uses if not specified.

| Bash Variable | Go YAML Key | Type | Default (func) | Default (root) | Tor Default | Decision | Notes |
|---------------|-------------|------|----------------|----------------|-------------|----------|-------|
| RejectPlaintextPorts | tor.reject_plaintext_ports | string | "" (commented out) | "" (commented out) | None | PORTED | Empty = disabled. Uncomment to block unencrypted ports. |
| WarnPlaintextPorts | tor.warn_plaintext_ports | string | "21,23,25,80,109,110,143" | "21,23,25,80,109,110,143" | "23,109,110,143" | PORTED | Extended beyond Tor default to include FTP, SMTP, HTTP. |
| CircuitBuildTimeout | tor.circuit_build_timeout | int | 60 | 60 | 60 | PORTED | Seconds. No difference between files. |
| CircuitsAvailableTimeout | tor.circuits_available_timeout | int | 5 | 360 | 1800 | PORTED | **Values differ drastically**. func=5s (aggressive), root=360s (6min). Using func value. |
| LearnCircuitBuildTimeout | tor.learn_circuit_build_timeout | int | 1 | 1 | 1 | PORTED | 0=disable adaptive learning. |
| CircuitStreamTimeout | tor.circuit_stream_timeout | int | 20 | 30 | 0 | PORTED | **Values differ**. func=20s, root=30s. 0=use Tor internal schedule. |
| ClientOnly | tor.client_only | int | 0 | 0 | 0 | PORTED | 1=don't run as relay. |
| ConnectionPadding | tor.connection_padding | int | 0 | 1 | auto | PORTED | **Values differ**. func=0 (off), root=1 (on). Using func value. |
| ReducedConnectionPadding | tor.reduced_connection_padding | int | 1 | 1 | 0 | PORTED | Less padding, shorter connections. |
| GeoIPExcludeUnknown | tor.geoip_exclude_unknown | int | 1 | 1 | auto | PORTED | 1=exclude all unknown-country nodes. |
| StrictNodes | tor.strict_nodes | int | 1 | 1 | 0 | PORTED | 1=enforce exclusions strictly even if it breaks functionality. |
| FascistFirewall | tor.fascist_firewall | int | 0 | 0 | 0 | PORTED | 1=only connect to FirewallPorts. |
| FirewallPorts | tor.firewall_ports | list | [80, 443] | [80, 443] | [80, 443] | PORTED | Converted from string to list. |
| LongLivedPorts | tor.long_lived_ports | list | [1, 2] | [1, 2] | [21,22,706,...] | PORTED | Minimal override: only ports 1,2 to avoid routing through high-uptime nodes. |
| NewCircuitPeriod | tor.new_circuit_period | int | 30 | 30 | 30 | PORTED | Seconds between considering new circuit. |
| MaxCircuitDirtiness | tor.max_circuit_dirtiness | int | 15 | 15 | 600 | PORTED | Seconds. Go randomizes 10..value per instance (Bash uses `shuf`). |
| MaxClientCircuitsPending | tor.max_client_circuits_pending | int | 1024 | 1024 | 32 | PORTED | Max 1024. Set high so circuits are always available. |
| SocksTimeout | tor.socks_timeout | int | derived: CircuitStreamTimeout+MINIMUM_TIMEOUT | same | 120 | PORTED | **Derived value**. func=35s, root=50s. Go computes at runtime. |
| TrackHostExitsExpire | tor.track_host_exits_expire | int | 10 | 120 | 1800 | PORTED | **Values differ**. func=10s (very aggressive), root=120s. Using func value. |
| UseEntryGuards | tor.use_entry_guards | int | 1 | 1 | 1 | PORTED | Keep as 1 for security. |
| NumEntryGuards | tor.num_entry_guards | int | 1 | 1 | 0 | PORTED | 0=use consensus. 1=always use exactly one guard. |
| SafeSocks | tor.safe_socks | int | 1 | 1 | 0 | PORTED | Reject unsafe SOCKS to prevent DNS leaks. |
| TestSocks | tor.test_socks | int | 1 | 1 | 0 | PORTED | Log SOCKS safety test results. |
| AllowNonRFC953Hostnames | tor.allow_non_rfc953_hostnames | int | 0 | 0 | 0 | PORTED | Block illegal hostname characters. |
| ClientRejectInternalAddresses | tor.client_reject_internal_addresses | int | 1 | 1 | 1 | PORTED | Reject connections to internal/private IPs. |
| DownloadExtraInfo | tor.download_extra_info | int | 0 | 0 | 0 | PORTED | Extra bandwidth cost, keep off. |
| OptimisticData | tor.optimistic_data | string | "auto" | "auto" | "auto" | PORTED | Send data before exit confirms connection. |
| AutomapHostsSuffixes | tor.automap_hosts_suffixes | string | ".exit,.onion" | ".exit,.onion" | ".exit,.onion" | PORTED | Domain suffixes for address mapping. |

---

## 16. Derived Timeout Values

These are computed from other settings. In Go, they are calculated at runtime.

| Bash Variable | Go YAML Key | Type | Expression (func) | Expression (root) | Decision | Notes |
|---------------|-------------|------|--------------------|--------------------|----------|-------|
| PRIVOXY_TIMEOUT | privoxy.timeout | int | CircuitStreamTimeout + MINIMUM_TIMEOUT = 35 | same = 50 | PORTED | Computed at runtime from tor settings |
| MASTER_PROXY_SERVER_TIMEOUT | proxy.master.server_timeout | int | PRIVOXY_TIMEOUT = 35 | CircuitStreamTimeout + MINIMUM_TIMEOUT = 50 | PORTED | **Different expressions** between files. func uses PRIVOXY_TIMEOUT directly. |
| MASTER_PROXY_CLIENT_TIMEOUT | proxy.master.client_timeout | int | RETRIES * SERVER_TIMEOUT * COUNTRIES | same formula, different values | PORTED | Computed at runtime. Not stored in YAML; formula applied in Go code. |

---

## 17. Runtime State Variables (DO NOT CHANGE section)

These are internal counters managed by the Bash script at runtime. In Go, they are managed
by the port allocator and instance manager — NOT stored in config.

| Bash Variable | Go Equivalent | Type | Default | Decision | Notes |
|---------------|--------------|------|---------|----------|-------|
| TOR_START_INSTANCE | — | — | 0 | DROPPED | Go uses 0-based indexing internally |
| TOR_CURRENT_INSTANCE | — | — | ${TOR_START_INSTANCE} | DROPPED | Go tracks instance state in memory |
| TOR_CURRENT_SOCKS_PORT | — | — | ${START_SOCKS_PORT} | DROPPED | Go port allocator manages this |
| TOR_CURRENT_CONTROL_PORT | — | — | ${START_CONTROL_PORT} | DROPPED | Go port allocator manages this |
| TOR_CURRENT_HTTP_PORT | — | — | ${TOR_START_HTTP_PORT} | DROPPED | Only in root. Go port allocator manages this. |
| TOR_CURRENT_TransPort | — | — | ${TOR_START_TransPort} | DROPPED | Only in root. Go port allocator manages this. |
| PRIVOXY_CURRENT_INSTANCE | — | — | 0 | DROPPED | Go tracks in memory |
| PRIVOXY_CURRENT_PORT | — | — | ${PRIVOXY_START_PORT} | DROPPED | Go port allocator manages this |
| HIDDEN_SERVICE_CURRENT_PORT | — | — | ${HIDDEN_START_PORT} | DROPPED | Only in func. Go port allocator manages this. |
| COUNT_CURRENT_INSTANCE | — | — | 0 | DROPPED | Only in root. Go tracks in memory. |
| DNSPORT | — | — | ${START_DNS_PORT} | DROPPED | Only in root. Go port allocator manages this. |
| NodeFamily | — | — | "" | DROPPED | Not used in any func file. |

---

## 18. Hardcoded Torrc Values (in boot_tor_instances.func)

These are hardcoded in the torrc template generation, not configurable via settings.cfg.
In Go, they can be exposed in an `advanced` section or kept as template defaults.

| Torrc Directive | Value | Go YAML Key | Decision | Notes |
|-----------------|-------|-------------|----------|-------|
| RunAsDaemon | 1 | — | DROPPED | Go manages process lifecycle; daemonization not needed |
| CookieAuthentication | 0 | tor.control_auth | PORTED | Changed to "cookie" (1) by default in Go version per ROADMAP |
| SafeLogging | 1 | — | HARDCODED | Always on for security. No reason to change. |
| DirCache | 1 | — | HARDCODED | Keep enabled for performance. |
| DisableDebuggerAttachment | 1 | — | HARDCODED | Security feature, always on. |
| NoExec | 1 | — | HARDCODED | Security feature, always on. |
| ProtocolWarnings | 1 | — | HARDCODED | Useful for debugging, always on. |
| TruncateLogFile | 1 | — | DROPPED | Only relevant when logs are enabled. |
| KeepBindCapabilities | auto | — | HARDCODED | Auto is the right default. |
| HardwareAccel | 0 | — | HARDCODED | Hardware acceleration rarely available/needed. |
| AvoidDiskWrites | 0 | — | HARDCODED | Left at Tor default. |
| CircuitPriorityHalflife | 1 | — | HARDCODED | Tuning parameter, could be exposed if needed. |
| ExtendByEd25519ID | auto | — | HARDCODED | Auto is the right default. |
| EnforceDistinctSubnets | 1 | — | HARDCODED | Security feature, always on. |
| TransPort | 0 | — | HARDCODED | Disabled; not used in this architecture. |
| NATDPort | 0 | — | HARDCODED | Disabled; not used. |
| ConstrainedSockSize | 8192 | — | HARDCODED | Socket buffer size tuning. |
| UseGuardFraction | auto | — | HARDCODED | Auto is the right default. |
| UseMicrodescriptors | auto | — | HARDCODED | Auto is the right default. |
| ClientUseIPv4 | 1 | — | PORTED | Could be exposed for IPv6 dual-stack (ROADMAP 7.3.3). |
| ClientUseIPv6 | 0 | — | PORTED | Could be exposed for IPv6 dual-stack (ROADMAP 7.3.3). |
| ClientPreferIPv6ORPort | auto | — | HARDCODED | Auto is the right default. |
| PathsNeededToBuildCircuits | -1 | — | HARDCODED | -1 = use consensus. |
| ClientBootstrapConsensusAuthorityDownloadSchedule | "6, 11, ..." | — | HARDCODED | Rarely needs tuning. |
| ClientBootstrapConsensusFallbackDownloadSchedule | "0, 1, 4, ..." | — | HARDCODED | Rarely needs tuning. |
| ClientBootstrapConsensusAuthorityOnlyDownloadSchedule | "0, 3, 7, ..." | — | HARDCODED | Rarely needs tuning. |
| ClientBootstrapConsensusMaxInProgressTries | 3 | — | HARDCODED | Rarely needs tuning. |
| NumDirectoryGuards | 0 | — | HARDCODED | 0 = use consensus. |
| GuardLifetime | 0 | — | HARDCODED | 0 = use consensus. |
| AutomapHostsOnResolve | 0 | — | HARDCODED | Disabled; not needed for this use case. |
| HiddenServiceMaxStreams | 0 | tor.hidden_service.max_streams | PORTED | 0 = unlimited. |
| HiddenServiceMaxStreamsCloseCircuit | 0 | tor.hidden_service.max_streams_close_circuit | PORTED | Close circuit on max streams. |
| HiddenServiceDirGroupReadable | 0 | tor.hidden_service.dir_group_readable | PORTED | Security: don't allow group read. |
| HiddenServiceNumIntroductionPoints | 3 | tor.hidden_service.num_introduction_points | PORTED | Standard value. |
| DataDirectoryGroupReadable | 0 | — | HARDCODED | Security: don't allow group read. |
| CacheDirectoryGroupReadable | 0 | — | HARDCODED | Security: don't allow group read. |
| FetchDirInfoEarly | 0 | — | HARDCODED | Not needed for client-only use. |
| FetchDirInfoExtraEarly | 0 | — | HARDCODED | Not needed for client-only use. |
| FetchHidServDescriptors | 1 | — | HARDCODED | Needed for hidden services. |
| FetchServerDescriptors | 1 | — | HARDCODED | Needed for normal operation. |
| FetchUselessDescriptors | 0 | — | HARDCODED | Save bandwidth. |
| KeepalivePeriod | ${MINIMUM_TIMEOUT} | — | DERIVED | Set from minimum_timeout at runtime. |

---

## 19. Variables Referenced But Never Defined (Bugs)

| Bash Variable | Where Used | Issue | Go Resolution |
|---------------|-----------|-------|---------------|
| MASTER_PROXY_PASSWORD | pre_loading.func:78 (HAProxy userlist) | Never defined in either settings.cfg. `insecure-password ${MASTER_PROXY_PASSWORD}` will expand to empty string. | Auto-generated at startup and stored in memory. |

---

## Summary Statistics

| Category | Count |
|----------|-------|
| Total Bash variables identified | 81 |
| PORTED to Go YAML config | 62 |
| DROPPED (Go runtime / not needed) | 15 |
| Hardcoded torrc values | 29 |
| Bugs found | 1 (MASTER_PROXY_PASSWORD undefined) |
| Variables with different defaults between files | 12 |

### Variables with Conflicting Defaults Between Files

These 12 variables have different default values between `legacy/func/settings.cfg` and
`legacy/settings.cfg`. The `func/` version takes precedence as it is the one actually
sourced by the script.

| Variable | func/settings.cfg | settings.cfg (root) | Go Default |
|----------|-------------------|---------------------|------------|
| RETRIES | 1000 | 100 | 1000 |
| MINIMUM_TIMEOUT | 15 | 20 | 15 |
| TORPATH | /usr/bin/tor | /usr/local/bin/tor | (auto-detect) |
| CircuitsAvailableTimeout | 5 | 360 | 5 |
| CircuitStreamTimeout | 20 | 30 | 20 |
| ConnectionPadding | 0 | 1 | 0 |
| TrackHostExitsExpire | 10 | 120 | 10 |
| HEALTH_CHECK_URL | https://www.google.com/ | www.google.com | https://www.google.com/ |
| HEALTH_CHECK_INTERVAL | 12 | 3 | 12 |
| DO_NOT_PROXY | 6 local IPs | 16 RFC1918 ranges | 6 local IPs |
| MASTER_PROXY_STAT_PORT | 63537 | 63539 | 63537 |
| MASTER_PROXY_SERVER_TIMEOUT expr | PRIVOXY_TIMEOUT | CircuitStreamTimeout+MINIMUM_TIMEOUT | PRIVOXY_TIMEOUT |
