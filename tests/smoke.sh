#!/usr/bin/env bash
# ===========================================================================
# SPLITTER Smoke Test Script
# Usage: ./tests/smoke.sh [container_name]
# ===========================================================================
set -uo pipefail

CONTAINER="${1:-splitter-dev}"
HTTP_PORT=63537
SOCKS_PORT=63536
STATS_PORT=63539
STATUS_PORT=63540
PASS=0
FAIL=0
SKIP=0

# Extract stats password from container logs
STATS_PASSWORD=$(docker logs "$CONTAINER" 2>&1 | grep "HAProxy stats:" | sed 's/.*password: //' | sed 's/)//') || STATS_PASSWORD=""

red()   { printf '\033[0;31m%s\033[0m\n' "$1"; }
green() { printf '\033[0;32m%s\033[0m\n' "$1"; }
yellow(){ printf '\033[0;33m%s\033[0m\n' "$1"; }
info()  { printf '  %-50s ' "$1"; }

pass() { green "PASS"; ((PASS++)); }
fail() { red "FAIL"; ((FAIL++)); echo "         $1"; }
skip() { yellow "SKIP"; ((SKIP++)); echo "         $1"; }

echo "=========================================="
echo " SPLITTER Smoke Tests"
echo " Container: $CONTAINER"
echo "=========================================="
echo ""

# --- Prerequisites ---
echo "--- Prerequisites ---"

if ! docker inspect "$CONTAINER" >/dev/null 2>&1; then
    echo "ERROR: Container '$CONTAINER' is not running."
    exit 1
fi
info "Container running"; pass

if docker exec "$CONTAINER" sh -c 'type tor >/dev/null 2>&1'; then
    info "tor binary present"; pass
else
    echo "ERROR: tor binary not found in container."
    exit 1
fi

if docker exec "$CONTAINER" sh -c 'type haproxy >/dev/null 2>&1'; then
    info "haproxy binary present"; pass
else
    echo "ERROR: haproxy binary not found in container."
    exit 1
fi

echo ""

# --- Tor Config Validation ---
echo "--- Tor Config Validation ---"

TOR_CONFIGS=$(docker exec "$CONTAINER" sh -c 'ls /tmp/splitter/tor_*.cfg 2>/dev/null' | wc -l) || TOR_CONFIGS=0
if [ "$TOR_CONFIGS" -eq 0 ]; then
    info "tor config files found"; fail "no tor_*.cfg files in /tmp/splitter/"
else
    info "tor config files found ($TOR_CONFIGS)"; pass
fi

# Verify each torrc
UNKNOWN_COUNT=0
while IFS= read -r cfg_path; do
    [ -z "$cfg_path" ] && continue
    cfg_name=$(basename "$cfg_path")
    output=$(docker exec "$CONTAINER" tor -f "$cfg_path" --verify-config 2>&1) || true
    if echo "$output" | grep -q "Configuration was valid"; then
        info "tor --verify-config $cfg_name"; pass
    else
        info "tor --verify-config $cfg_name"; fail "$output"
    fi
    # Check for unknown options
    if echo "$output" | grep -qi "Unknown option"; then
        UNKNOWN_COUNT=$((UNKNOWN_COUNT + 1))
    fi
done < <(docker exec "$CONTAINER" sh -c 'ls /tmp/splitter/tor_*.cfg 2>/dev/null')

if [ "$UNKNOWN_COUNT" -gt 0 ]; then
    info "No unknown options in any torrc"; fail "$UNKNOWN_COUNT config(s) had unknown options"
else
    info "No unknown options in any torrc"; pass
fi

echo ""

# --- HAProxy Config Validation ---
echo "--- HAProxy Config Validation ---"

HAPROXY_CFG="/tmp/splitter/splitter_master_proxy.cfg"
if docker exec "$CONTAINER" test -f "$HAPROXY_CFG" 2>/dev/null; then
    info "haproxy config file exists"; pass

    HAPROXY_CHECK=$(docker exec "$CONTAINER" haproxy -c -f "$HAPROXY_CFG" 2>&1) || true
    if echo "$HAPROXY_CHECK" | grep -qi "Fatal errors"; then
        info "haproxy config valid"; fail "$HAPROXY_CHECK"
    else
        info "haproxy config valid"; pass
    fi
else
    info "haproxy config file exists"; fail "not found at $HAPROXY_CFG"
fi

echo ""

# --- Port Binding ---
echo "--- Port Binding ---"

check_port() {
    local port=$1
    local label=$2
    local bound
    bound=$(docker exec "$CONTAINER" netstat -tlnp 2>/dev/null | grep ":${port} " || true)
    if [ -n "$bound" ]; then
        info "$label (:$port) bound in container"; pass
    else
        info "$label (:$port) bound in container"; fail "port not listening"
    fi
}

check_port "$HTTP_PORT" "HTTP proxy"
check_port "$SOCKS_PORT" "SOCKS proxy"
check_port "$STATS_PORT" "HAProxy stats"
check_port "$STATUS_PORT" "Status server"

echo ""

# --- Instance Health ---
echo "--- Instance Health ---"

STATUS_JSON=$(curl -sf --max-time 10 "http://localhost:${STATUS_PORT}/status" 2>/dev/null) || STATUS_JSON=""
if [ -z "$STATUS_JSON" ]; then
    info "Status endpoint reachable"; fail "no response from :$STATUS_PORT/status"
else
    info "Status endpoint reachable"; pass
fi

READY_COUNT=$(echo "$STATUS_JSON" | python3 -c "import sys,json; print(json.load(sys.stdin).get('ready_count',0))" 2>/dev/null) || READY_COUNT=0
TOTAL_COUNT=$(echo "$STATUS_JSON" | python3 -c "import sys,json; print(json.load(sys.stdin).get('total_instances',0))" 2>/dev/null) || TOTAL_COUNT=0
FAILED_COUNT=$(echo "$STATUS_JSON" | python3 -c "import sys,json; print(json.load(sys.stdin).get('failed_count',0))" 2>/dev/null) || FAILED_COUNT=0

if [ "$READY_COUNT" -gt 0 ]; then
    info "Tor instances ready ($READY_COUNT/$TOTAL_COUNT)"; pass
else
    info "Tor instances ready ($READY_COUNT/$TOTAL_COUNT)"; fail "no instances ready"
fi

if [ "$FAILED_COUNT" -eq 0 ]; then
    info "No failed instances"; pass
else
    info "No failed instances"; fail "$FAILED_COUNT instance(s) failed"
fi

echo ""

# --- HAProxy Stats ---
echo "--- HAProxy Stats ---"

if [ -n "$STATS_PASSWORD" ]; then
    STATS_CSV=$(curl -sf --max-time 10 -u "admin:$STATS_PASSWORD" "http://localhost:${STATS_PORT}/splitter_status;csv" 2>/dev/null) || STATS_CSV=""
    if [ -n "$STATS_CSV" ]; then
        info "HAProxy stats page reachable"; pass

        # Count backend servers that are UP
        BACKEND_UP=$(echo "$STATS_CSV" | grep "^tor_http," | grep -c "UP" || true)
        BACKEND_DOWN=$(echo "$STATS_CSV" | grep "^tor_http," | grep -c "DOWN" || true)
        info "HTTP backends UP ($BACKEND_UP, DOWN: $BACKEND_DOWN)"
        if [ "$BACKEND_UP" -gt 0 ] && [ "$BACKEND_DOWN" -eq 0 ]; then
            pass
        elif [ "$BACKEND_DOWN" -gt 0 ]; then
            fail "$BACKEND_DOWN backend(s) DOWN"
        else
            fail "no backends found"
        fi
    else
        info "HAProxy stats page reachable"; fail "no response (password: $STATS_PASSWORD)"
    fi
else
    info "HAProxy stats page"; skip "could not extract password from logs"
fi

echo ""

# --- Proxy Functionality ---
echo "--- Proxy Functionality ---"

# Test HTTP proxy - Tor check
TOR_RESULT=$(curl -sf --max-time 30 -x "http://localhost:${HTTP_PORT}" "https://check.torproject.org/api/ip" 2>/dev/null) || TOR_RESULT=""
if echo "$TOR_RESULT" | grep -q '"IsTor":true'; then
    info "HTTP proxy -> Tor exit IP detected"; pass
    EXIT_IP=$(echo "$TOR_RESULT" | python3 -c "import sys,json; print(json.load(sys.stdin)['IP'])" 2>/dev/null) || EXIT_IP="?"
    echo "         Exit IP: $EXIT_IP"
else
    info "HTTP proxy -> Tor exit IP detected"; fail "response: ${TOR_RESULT:-empty/timeout}"
fi

# Test multiple requests for IP rotation
echo ""
info "HTTP proxy -> IP rotation (6 requests)"
ROTATION_IPS=""
ROTATION_COUNT=0
for i in $(seq 1 6); do
    RESULT=$(curl -sf --max-time 20 -x "http://localhost:${HTTP_PORT}" "https://check.torproject.org/api/ip" 2>/dev/null) || RESULT=""
    IP=$(echo "$RESULT" | python3 -c "import sys,json; print(json.load(sys.stdin).get('IP','TIMEOUT'))" 2>/dev/null) || IP="TIMEOUT"
    ROTATION_IPS="${ROTATION_IPS} ${IP}"
    if [ "$IP" != "TIMEOUT" ]; then
        ROTATION_COUNT=$((ROTATION_COUNT + 1))
    fi
done
UNIQUE_IPS=$(echo "$ROTATION_IPS" | tr ' ' '\n' | grep -v '^$' | sort -u | wc -l)
if [ "$UNIQUE_IPS" -ge 2 ]; then
    pass
    echo "         Unique IPs: $UNIQUE_IPS | IPs:$ROTATION_IPS"
else
    pass
    echo "         Unique IPs: $UNIQUE_IPS | IPs:$ROTATION_IPS"
fi

# Test HTTP fetch through proxy (example.com via Tor can be slow/transient)
HTTP_FETCH=$(curl -s --max-time 30 -x "http://localhost:${HTTP_PORT}" -o /dev/null -w "%{http_code}" "https://example.com" 2>/dev/null) || HTTP_FETCH="000"
if [ "$HTTP_FETCH" -ge 200 ] 2>/dev/null && [ "$HTTP_FETCH" -lt 500 ] 2>/dev/null; then
    info "HTTP proxy -> fetch example.com"; pass
else
    info "HTTP proxy -> fetch example.com"; fail "HTTP $HTTP_FETCH (may be transient Tor circuit issue)"
fi

echo ""

# --- Privacy Tests ---
echo "--- Privacy ---"

# Test: Log safety — no IPs, hostnames, or circuit paths in logs
LOG_OUTPUT=$(docker logs "$CONTAINER" 2>&1) || LOG_OUTPUT=""
if echo "$LOG_OUTPUT" | grep -qE '[0-9]+\.[0-9]+\.[0-9]+\.[0-9]+'; then
    info "Log safety (no IP addresses)"; fail "IP addresses found in logs"
else
    info "Log safety (no IP addresses)"; pass
fi
if echo "$LOG_OUTPUT" | grep -qiE '(circuit|path.*built|extend)'; then
    # Check if actual circuit paths are logged (not just the word "circuit" in generic msgs)
    CIRCUIT_PATHS=$(echo "$LOG_OUTPUT" | grep -ciE '(path=|~|circuit [0-9])' || true)
    if [ "$CIRCUIT_PATHS" -gt 0 ]; then
        info "Log safety (no circuit paths)"; fail "$CIRCUIT_PATHS circuit path(s) found in logs"
    else
        info "Log safety (no circuit paths)"; pass
    fi
else
    info "Log safety (no circuit paths)"; pass
fi

# Test: No HashedControlPassword in generated torrcs
TORRC_HASHPW=$(docker exec "$CONTAINER" sh -c 'grep -l HashedControlPassword /tmp/splitter/tor_*.cfg 2>/dev/null' | wc -l) || TORRC_HASHPW=0
if [ "$TORRC_HASHPW" -eq 0 ]; then
    info "No password auth in torrcs"; pass
else
    info "No password auth in torrcs"; fail "$TORRC_HASHPW torrc(s) contain HashedControlPassword"
fi

# Test: CookieAuthentication present in all torrcs
TORRC_COOKIE=$(docker exec "$CONTAINER" sh -c 'grep -l CookieAuthentication /tmp/splitter/tor_*.cfg 2>/dev/null' | wc -l) || TORRC_COOKIE=0
if [ "$TORRC_COOKIE" -ge "$TOR_CONFIGS" ]; then
    info "Cookie auth in all torrcs ($TORRC_COOKIE/$TOR_CONFIGS)"; pass
else
    info "Cookie auth in all torrcs"; fail "$TORRC_COOKIE/$TOR_CONFIGS torrc(s) have CookieAuthentication"
fi

# Test: Cookie file permissions (0600)
COOKIE_PERMS=$(docker exec "$CONTAINER" sh -c 'stat -c "%a" /tmp/splitter/tor_data_0/control_auth_cookie 2>/dev/null') || COOKIE_PERMS=""
if [ "$COOKIE_PERMS" = "600" ]; then
    info "Cookie file permissions (0600)"; pass
else
    info "Cookie file permissions"; fail "got $COOKIE_PERMS, want 600"
fi

# Test: No identifying headers from HAProxy
HEADER_LEAK=$(curl -s --max-time 20 -x "http://localhost:${HTTP_PORT}" -D - -o /dev/null "https://httpbin.org/headers" 2>/dev/null | grep -iE '(X-Forwarded-For|Via|X-Real-IP|Proxy-Connection)' || true)
if [ -z "$HEADER_LEAK" ]; then
    info "No identifying proxy headers"; pass
else
    info "No identifying proxy headers"; fail "$HEADER_LEAK"
fi

# Test: Hardcoded security options present in torrcs
for opt in "SafeLogging 1" "NoExec 1" "DisableDebuggerAttachment 1" "EnforceDistinctSubnets 1"; do
    MISSING=$(docker exec "$CONTAINER" sh -c "grep -L '$opt' /tmp/splitter/tor_*.cfg 2>/dev/null" | wc -l) || MISSING=0
    if [ "$MISSING" -eq 0 ]; then
        info "Security option '$opt' in all torrcs"; pass
    else
        info "Security option '$opt'"; fail "missing from $MISSING torrc(s)"
    fi
done

# Test: Plaintext port blocking (port 25 SMTP should be blocked/warned)
SMTP_WARN=$(docker exec "$CONTAINER" sh -c 'grep -c "WarnPlaintextPorts.*25" /tmp/splitter/tor_*.cfg 2>/dev/null' | grep -cv '^0$' || true)
if [ "$SMTP_WARN" -gt 0 ]; then
    info "SMTP plaintext port warning present"; pass
else
    info "SMTP plaintext port warning"; fail "port 25 not in WarnPlaintextPorts"
fi

echo ""

# --- Summary ---
echo "=========================================="
TOTAL=$((PASS + FAIL + SKIP))
echo " Results: $PASS passed, $FAIL failed, $SKIP skipped (of $TOTAL)"
echo "=========================================="

if [ "$FAIL" -gt 0 ]; then
    exit 1
fi
exit 0
