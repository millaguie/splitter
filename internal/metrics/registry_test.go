package metrics

import (
	"strings"
	"testing"
)

func TestCounterIncrement(t *testing.T) {
	r := NewRegistry()
	c := r.NewCounter("test_counter", "A test counter")
	c.Inc()
	c.Inc()
	c.Inc()

	out := r.Render()
	if !strings.Contains(out, "test_counter 3") {
		t.Fatalf("expected counter value 3, got:\n%s", out)
	}
	if !strings.Contains(out, "# TYPE test_counter counter") {
		t.Fatalf("missing TYPE line:\n%s", out)
	}
	if !strings.Contains(out, "# HELP test_counter A test counter") {
		t.Fatalf("missing HELP line:\n%s", out)
	}
}

func TestCounterAdd(t *testing.T) {
	r := NewRegistry()
	c := r.NewCounter("test_add", "Add test")
	c.Add(5)
	c.Add(3)

	out := r.Render()
	if !strings.Contains(out, "test_add 8") {
		t.Fatalf("expected counter value 8, got:\n%s", out)
	}
}

func TestGaugeSet(t *testing.T) {
	r := NewRegistry()
	g := r.NewGauge("test_gauge", "A test gauge")
	g.Set(42)
	g.Set(7)

	out := r.Render()
	if !strings.Contains(out, "test_gauge 7") {
		t.Fatalf("expected gauge value 7, got:\n%s", out)
	}
	if !strings.Contains(out, "# TYPE test_gauge gauge") {
		t.Fatalf("missing TYPE line:\n%s", out)
	}
}

func TestLabeledGauge(t *testing.T) {
	r := NewRegistry()
	g := r.NewGauge("labeled_gauge", "With labels")
	g.SetWithLabels(1, "instance_id", "1", "state", "ready")
	g.SetWithLabels(0, "instance_id", "2", "state", "failed")

	out := r.Render()
	if !strings.Contains(out, `labeled_gauge{instance_id="1",state="ready"} 1`) {
		t.Fatalf("expected labeled value for instance 1, got:\n%s", out)
	}
	if !strings.Contains(out, `labeled_gauge{instance_id="2",state="failed"} 0`) {
		t.Fatalf("expected labeled value for instance 2, got:\n%s", out)
	}
}

func TestLabeledCounter(t *testing.T) {
	r := NewRegistry()
	c := r.NewCounter("labeled_counter", "Labeled counter")
	c.IncWithLabels("instance_id", "1")
	c.IncWithLabels("instance_id", "1")
	c.IncWithLabels("instance_id", "2")

	out := r.Render()
	if !strings.Contains(out, `labeled_counter{instance_id="1"} 2`) {
		t.Fatalf("expected labeled counter 2 for instance 1, got:\n%s", out)
	}
	if !strings.Contains(out, `labeled_counter{instance_id="2"} 1`) {
		t.Fatalf("expected labeled counter 1 for instance 2, got:\n%s", out)
	}
}

func TestLabeledGaugeOverwrite(t *testing.T) {
	r := NewRegistry()
	g := r.NewGauge("overwrite_gauge", "Overwrite test")
	g.SetWithLabels(50, "instance_id", "1")
	g.SetWithLabels(100, "instance_id", "1")

	out := r.Render()
	if !strings.Contains(out, `overwrite_gauge{instance_id="1"} 100`) {
		t.Fatalf("expected overwritten value 100, got:\n%s", out)
	}
}

func TestRegistryMultipleMetrics(t *testing.T) {
	r := NewRegistry()
	c := r.NewCounter("counter_a", "Counter A")
	g := r.NewGauge("gauge_b", "Gauge B")
	c.Inc()
	g.Set(99)

	out := r.Render()
	if !strings.Contains(out, "counter_a 1") {
		t.Fatalf("expected counter_a, got:\n%s", out)
	}
	if !strings.Contains(out, "gauge_b 99") {
		t.Fatalf("expected gauge_b, got:\n%s", out)
	}
}

func TestRegistryDuplicateName(t *testing.T) {
	r := NewRegistry()
	c1 := r.NewCounter("dup", "First")
	c2 := r.NewCounter("dup", "Second")
	c1.Inc()
	c2.Inc()

	out := r.Render()
	if !strings.Contains(out, "dup 2") {
		t.Fatalf("expected combined value 2 for duplicate metric, got:\n%s", out)
	}
}

func TestRegistryEmpty(t *testing.T) {
	r := NewRegistry()
	out := r.Render()
	if out != "" {
		t.Fatalf("expected empty output, got:\n%s", out)
	}
}

func TestGaugeUnlabeledRender(t *testing.T) {
	r := NewRegistry()
	g := r.NewGauge("simple", "Simple gauge")
	g.Set(0)

	out := r.Render()
	if !strings.Contains(out, "simple 0") {
		t.Fatalf("expected value 0, got:\n%s", out)
	}
}

func TestCounterUnlabeledWithLabeled(t *testing.T) {
	r := NewRegistry()
	c := r.NewCounter("mixed", "Mixed counter")
	c.Inc()
	c.Inc()
	c.IncWithLabels("instance_id", "1")

	out := r.Render()
	if !strings.Contains(out, "mixed 2") {
		t.Fatalf("expected unlabeled value 2, got:\n%s", out)
	}
	if !strings.Contains(out, `mixed{instance_id="1"} 1`) {
		t.Fatalf("expected labeled value 1, got:\n%s", out)
	}
}
