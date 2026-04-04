# ===========================================================================
# SPLITTER Makefile
# ===========================================================================
#
# Quick reference:
#   make                  build the binary
#   make test             unit tests
#   make test-all         unit + integration tests
#   make vet              static analysis
#   make lint             golangci-lint (if installed)
#   make check            vet + test + build (CI-equivalent)
#   make clean            remove build artifacts
#
# Docker:
#   make docker-build     build Docker image (splitter:test)
#   make docker-up        build + start dev container
#   make docker-down      stop and remove dev container
#   make docker-logs      tail container logs
#   make docker-restart   rebuild + restart (code change iteration)
#
# Smoke tests:
#   make smoke            run smoke tests against running container
#   make smoke-proxy      quick HTTP proxy connectivity check
#
# ===================================================================

# ── Variables ──────────────────────────────────────────────────────────────

BINARY     := splitter
IMAGE      := splitter:test
CONTAINER  := splitter-dev
COMPOSE    := docker compose -f docker-compose.dev.yml
GOFLAGS    := -trimpath
LDFLAGS    := -s -w
TESTPKGS   := ./...
TESTTAGS   :=

# Ports (must match docker-compose.dev.yml)
HTTP_PORT  := 63537
SOCKS_PORT := 63536
STATS_PORT := 63539
STATUS_PORT:= 63540

# ── Build ───────────────────────────────────────────────────────────────────

.PHONY: build
build: ## Build the Go binary
	go build $(GOFLAGS) -ldflags "$(LDFLAGS)" -o $(BINARY) .

.PHONY: build-race
build-race: ## Build with race detector
	go build -race -o $(BINARY) .

.PHONY: clean
clean: ## Remove build artifacts and temp files
	rm -f $(BINARY)
	go clean

# ── Testing ─────────────────────────────────────────────────────────────────

.PHONY: test
test: ## Run unit tests
	go test -count=1 $(TESTPKGS)

.PHONY: test-verbose
test-verbose: ## Run unit tests with verbose output
	go test -v -count=1 $(TESTPKGS)

.PHONY: test-race
test-race: ## Run unit tests with race detector
	go test -race -count=1 $(TESTPKGS)

.PHONY: test-coverage
test-coverage: ## Run tests and generate coverage report
	go test -coverprofile=coverage.out $(TESTPKGS)
	go tool cover -func=coverage.out
	@echo "---"
	@echo "HTML report: go tool cover -html=coverage.out"

.PHONY: test-integration
test-integration: ## Run integration tests (requires tor, haproxy installed)
	go test -tags=integration -v -count=1 ./internal/tor/...

.PHONY: test-all
test-all: ## Run unit + integration tests
	@echo "=== Unit tests ==="
	go test -count=1 $(TESTPKGS)
	@echo ""
	@echo "=== Integration tests ==="
	go test -tags=integration -v -count=1 ./internal/tor/...

.PHONY: test-privacy
test-privacy: ## Run only privacy-related tests
	@echo "=== Torrc privacy assertions ==="
	go test -v -count=1 -run "Privacy|Hardcoded|Configurable|CannotBe|ControlPort|Deprecated|SecurityDirective" ./internal/tor/...
	@echo ""
	@echo "=== Config privacy defaults ==="
	go test -v -count=1 -run "Privacy|ControlAuth|ReducedConnection|EntryGuard|CircuitTimeout|StreamIsolation" ./internal/config/...

# ── Static analysis ────────────────────────────────────────────────────────

.PHONY: vet
vet: ## Run go vet
	go vet ./...

.PHONY: fmt
fmt: ## Run gofmt (write changes)
	gofmt -w .

.PHONY: fmt-check
fmt-check: ## Check formatting without changes
	@test -z "$$(gofmt -l .)" || (echo "files need formatting:"; gofmt -l .; exit 1)

.PHONY: lint
lint: ## Run golangci-lint (install: go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest)
	golangci-lint run ./...

.PHONY: check
check: vet test build ## vet + test + build (CI-equivalent)

# ── Docker ──────────────────────────────────────────────────────────────────

.PHONY: docker-build
docker-build: ## Build Docker image
	docker build -t $(IMAGE) .

.PHONY: docker-up
docker-up: ## Build image and start dev container
	docker build -t $(IMAGE) . && \
	$(COMPOSE) up -d && \
	echo "" && \
	echo "Waiting for bootstrap..." && \
	sleep 10 && \
	echo "" && \
	docker logs --tail 20 splitter-dev-1 2>&1 || docker logs --tail 20 $(CONTAINER) 2>&1

.PHONY: docker-down
docker-down: ## Stop and remove dev container
	-$(COMPOSE) down 2>/dev/null || true
	-docker rm -f $(CONTAINER) 2>/dev/null || true

.PHONY: docker-logs
docker-logs: ## Tail container logs
	docker logs -f --tail 50 $(CONTAINER)

.PHONY: docker-restart
docker-restart: ## Rebuild image and restart container (code change iteration)
	$(MAKE) docker-down
	$(MAKE) docker-up

.PHONY: docker-shell
docker-shell: ## Shell into running container
	docker exec -it $(CONTAINER) /bin/sh

.PHONY: docker-verify
docker-verify: ## Verify tor configs inside running container
	@echo "=== Verifying tor configs ==="
	@for cfg in $$(docker exec $(CONTAINER) sh -c 'ls /tmp/splitter/tor_*.cfg 2>/dev/null'); do \
		echo -n "  $$cfg: "; \
		docker exec $(CONTAINER) tor -f "$$cfg" --verify-config 2>&1 | grep -o "Configuration was valid\|Unknown option.*"; \
	done
	@echo ""
	@echo "=== Verifying HAProxy config ==="
	@docker exec $(CONTAINER) haproxy -c -f /tmp/splitter/splitter_master_proxy.cfg 2>&1 || true
	@echo ""
	@echo "=== Listening ports ==="
	@docker exec $(CONTAINER) netstat -tlnp 2>/dev/null | grep "6353" || true

# ── Smoke tests ─────────────────────────────────────────────────────────────

.PHONY: smoke
smoke: ## Run full smoke test suite against running container
	@./tests/smoke.sh $(CONTAINER)

.PHONY: smoke-proxy
smoke-proxy: ## Quick HTTP proxy connectivity check
	@echo "Checking HTTP proxy on :$(HTTP_PORT)..."
	@RESULT=$$(curl -sf --max-time 15 -x http://localhost:$(HTTP_PORT) https://check.torproject.org/api/ip 2>/dev/null) || RESULT=""; \
	if echo "$$RESULT" | grep -q '"IsTor":true'; then \
		IP=$$(echo "$$RESULT" | python3 -c "import sys,json; print(json.load(sys.stdin)['IP'])" 2>/dev/null); \
		echo "  Tor exit IP: $$IP"; \
	else \
		echo "  FAIL: no response or not Tor (is the container running?)"; \
		exit 1; \
	fi

.PHONY: smoke-rotation
smoke-rotation: ## Check IP rotation across multiple requests
	@echo "Checking IP rotation (6 requests)..."
	@IPS=""; \
	for i in 1 2 3 4 5 6; do \
		R=$$(curl -sf --max-time 15 -x http://localhost:$(HTTP_PORT) https://check.torproject.org/api/ip 2>/dev/null) || R=""; \
		IP=$$(echo "$$R" | python3 -c "import sys,json; print(json.load(sys.stdin).get('IP','TIMEOUT'))" 2>/dev/null) || IP="TIMEOUT"; \
		IPS="$$IPS $$IP"; \
		sleep 1; \
	done; \
	UNIQUE=$$(echo "$$IPS" | tr ' ' '\n' | grep -v '^$$' | sort -u | wc -l); \
	echo "  Unique IPs: $$UNIQUE / 6 | IPs:$$IPS"

.PHONY: smoke-status
smoke-status: ## Check status API
	@curl -sf --max-time 5 http://localhost:$(STATUS_PORT)/status 2>/dev/null \
		| python3 -c 'import sys,json; d=json.load(sys.stdin); print(f"  Instances: {d[\"ready_count\"]}/{d[\"total_instances\"]} ready, {d[\"failed_count\"]} failed"); print(f"  Tor:      {d[\"tor_version\"]}"); print(f"  Features: {\" \".join(k for k,v in d[\"features\"].items() if v)}")' \
		|| echo "  FAIL: status endpoint unreachable"

# ── Convenience ─────────────────────────────────────────────────────────────

.PHONY: status
status: smoke-status ## Alias for smoke-status

.PHONY: logs
logs: docker-logs ## Alias for docker-logs

.PHONY: run
run: build ## Build and run locally
	./$(BINARY) run --log

# ── Help ────────────────────────────────────────────────────────────────────

.PHONY: help
help: ## Show this help
	@echo ""
	@echo "SPLITTER — available targets:"
	@echo ""
	@grep -E '^[a-zA-Z_-]+:.*##' $(MAKEFILE_LIST) | \
		awk 'BEGIN {FS = ":.*## "}; {printf "  \033[36m%-22s\033[0m %s\n", $$1, $$2}'
	@echo ""
	@echo "Docker shortcuts:"
	@echo "  make docker-restart   rebuild + restart (after code changes)"
	@echo "  make smoke            full test suite against container"
	@echo "  make smoke-proxy      quick Tor connectivity check"
	@echo ""
	@echo "Ports:"
	@echo "  :$(SOCKS_PORT)  SOCKS5 proxy"
	@echo "  :$(HTTP_PORT)  HTTP proxy"
	@echo "  :$(STATS_PORT)  HAProxy stats"
	@echo "  :$(STATUS_PORT) Status API"
	@echo ""

.DEFAULT_GOAL := help
