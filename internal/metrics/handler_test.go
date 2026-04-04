package metrics

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestHandlerMetricsEndpoint(t *testing.T) {
	r := NewRegistry()
	c := r.NewCounter("test_requests", "Total requests")
	c.Inc()
	h := NewHandler(r)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if ct != "text/plain; version=0.0.4; charset=utf-8" {
		t.Fatalf("unexpected content type: %s", ct)
	}
	body := rec.Body.String()
	if !strings.Contains(body, "test_requests 1") {
		t.Fatalf("expected metric in body, got:\n%s", body)
	}
}

func TestHandlerHealthzEndpoint(t *testing.T) {
	r := NewRegistry()
	h := NewHandler(r)

	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	rec := httptest.NewRecorder()
	h.Healthz(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	ct := rec.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Fatalf("unexpected content type: %s", ct)
	}

	var result map[string]string
	if err := json.NewDecoder(rec.Body).Decode(&result); err != nil {
		t.Fatalf("failed to decode healthz response: %v", err)
	}
	if result["status"] != "ok" {
		t.Fatalf("expected status ok, got %s", result["status"])
	}
}

func TestHandlerEmptyRegistry(t *testing.T) {
	r := NewRegistry()
	h := NewHandler(r)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", rec.Code)
	}
	if rec.Body.String() != "" {
		t.Fatalf("expected empty body, got:\n%s", rec.Body.String())
	}
}

func TestHandlerFullPrometheusFormat(t *testing.T) {
	r := NewRegistry()
	g := r.NewGauge("splitter_instances_total", "Total instances")
	g.Set(3)
	c := r.NewCounter("splitter_errors_total", "Total errors")
	c.Inc()
	c.Inc()
	h := NewHandler(r)

	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	body := rec.Body.String()
	lines := strings.Split(strings.TrimSpace(body), "\n")
	expected := []string{
		"# HELP splitter_instances_total Total instances",
		"# TYPE splitter_instances_total gauge",
		"splitter_instances_total 3",
		"# HELP splitter_errors_total Total errors",
		"# TYPE splitter_errors_total counter",
		"splitter_errors_total 2",
	}
	if len(lines) != len(expected) {
		t.Fatalf("expected %d lines, got %d:\n%s", len(expected), len(lines), body)
	}
	for i, exp := range expected {
		if lines[i] != exp {
			t.Fatalf("line %d: expected %q, got %q", i, exp, lines[i])
		}
	}
}

func TestSplitterMetricsIntegration(t *testing.T) {
	r := NewRegistry()
	sm := NewSplitterMetrics(r)

	sm.SetInstanceCount(4, 3)
	sm.SetInstanceState("1", "ready")
	sm.SetInstanceState("2", "ready")
	sm.SetInstanceState("3", "ready")
	sm.SetInstanceState("4", "bootstrapping")
	sm.SetInstanceCountry("1", "us")
	sm.SetInstanceCountry("2", "de")
	sm.IncCircuits()
	sm.IncCircuitRenewal("1")
	sm.IncCircuitRenewal("1")
	sm.IncErrors()
	sm.SetBootstrapProgress("4", 75.5)

	out := r.Render()

	if !strings.Contains(out, "splitter_instances_total 4") {
		t.Fatalf("expected 4 total instances, got:\n%s", out)
	}
	if !strings.Contains(out, "splitter_instances_active 3") {
		t.Fatalf("expected 3 active instances, got:\n%s", out)
	}
	if !strings.Contains(out, `splitter_instance_state{instance_id="1",state="ready"} 1`) {
		t.Fatalf("expected instance 1 state, got:\n%s", out)
	}
	if !strings.Contains(out, `splitter_instance_country{country="de",instance_id="2"} 1`) {
		t.Fatalf("expected instance 2 country, got:\n%s", out)
	}
	if !strings.Contains(out, "splitter_circuits_total 1") {
		t.Fatalf("expected 1 circuit, got:\n%s", out)
	}
	if !strings.Contains(out, `splitter_circuit_renewals_total{instance_id="1"} 2`) {
		t.Fatalf("expected 2 renewals for instance 1, got:\n%s", out)
	}
	if !strings.Contains(out, "splitter_errors_total 1") {
		t.Fatalf("expected 1 error, got:\n%s", out)
	}
	if !strings.Contains(out, `splitter_bootstrap_progress{instance_id="4"} 75.5`) {
		t.Fatalf("expected bootstrap progress 75.5, got:\n%s", out)
	}
}

func TestServerStartAndShutdown(t *testing.T) {
	r := NewRegistry()
	h := NewHandler(r)
	srv := NewServer("127.0.0.1:0", h)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- srv.Start(ctx)
	}()

	for range 50 {
		resp, err := http.Get("http://" + srv.server.Addr + "/healthz")
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				break
			}
		}
	}

	cancel()
	if err := <-done; err != nil {
		t.Fatalf("server returned error: %v", err)
	}
}
