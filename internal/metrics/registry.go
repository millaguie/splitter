package metrics

import (
	"fmt"
	"sort"
	"strings"
	"sync"
)

type metricType int

const (
	typeCounter metricType = iota
	typeGauge
)

func (t metricType) String() string {
	switch t {
	case typeCounter:
		return "counter"
	case typeGauge:
		return "gauge"
	default:
		return "untyped"
	}
}

type labeledValue struct {
	labels map[string]string
	value  float64
}

type metric struct {
	name    string
	help    string
	mtype   metricType
	mu      sync.RWMutex
	self    float64
	labeled []labeledValue
}

func (m *metric) Inc() {
	m.mu.Lock()
	m.self++
	m.mu.Unlock()
}

func (m *metric) Add(v float64) {
	m.mu.Lock()
	m.self += v
	m.mu.Unlock()
}

func (m *metric) Set(v float64) {
	m.mu.Lock()
	m.self = v
	m.mu.Unlock()
}

func (m *metric) IncWithLabels(pairs ...string) {
	key := labelKey(pairs)
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.labeled {
		if labelKeyFromMap(m.labeled[i].labels) == key {
			m.labeled[i].value++
			return
		}
	}
	m.labeled = append(m.labeled, labeledValue{
		labels: labelMap(pairs),
		value:  1,
	})
}

func (m *metric) SetWithLabels(v float64, pairs ...string) {
	key := labelKey(pairs)
	m.mu.Lock()
	defer m.mu.Unlock()
	for i := range m.labeled {
		if labelKeyFromMap(m.labeled[i].labels) == key {
			m.labeled[i].value = v
			return
		}
	}
	m.labeled = append(m.labeled, labeledValue{
		labels: labelMap(pairs),
		value:  v,
	})
}

func (m *metric) render() string {
	var b strings.Builder
	fmt.Fprintf(&b, "# HELP %s %s\n", m.name, m.help)
	fmt.Fprintf(&b, "# TYPE %s %s\n", m.name, m.mtype)

	m.mu.RLock()
	if m.self != 0 || len(m.labeled) == 0 {
		renderValue(&b, m.name, nil, m.self)
	}
	for _, lv := range m.labeled {
		renderValue(&b, m.name, lv.labels, lv.value)
	}
	m.mu.RUnlock()

	return b.String()
}

func renderValue(b *strings.Builder, name string, labels map[string]string, v float64) {
	if len(labels) == 0 {
		fmt.Fprintf(b, "%s %g\n", name, v)
		return
	}
	keys := make([]string, 0, len(labels))
	for k := range labels {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	pairs := make([]string, 0, len(keys))
	for _, k := range keys {
		pairs = append(pairs, fmt.Sprintf("%s=%q", k, labels[k]))
	}
	fmt.Fprintf(b, "%s{%s} %g\n", name, strings.Join(pairs, ","), v)
}

func labelMap(pairs []string) map[string]string {
	m := make(map[string]string, len(pairs)/2)
	for i := 0; i+1 < len(pairs); i += 2 {
		m[pairs[i]] = pairs[i+1]
	}
	return m
}

func labelKey(pairs []string) string {
	m := labelMap(pairs)
	return labelKeyFromMap(m)
}

func labelKeyFromMap(m map[string]string) string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	parts := make([]string, 0, len(keys))
	for _, k := range keys {
		parts = append(parts, k+"="+m[k])
	}
	return strings.Join(parts, ",")
}

type Registry struct {
	mu      sync.RWMutex
	metrics []*metric
	order   []string
}

func NewRegistry() *Registry {
	return &Registry{}
}

func (r *Registry) newMetric(name, help string, mt metricType) *metric {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, existing := range r.metrics {
		if existing.name == name {
			return existing
		}
	}
	m := &metric{
		name:  name,
		help:  help,
		mtype: mt,
	}
	r.metrics = append(r.metrics, m)
	r.order = append(r.order, name)
	return m
}

func (r *Registry) NewCounter(name, help string) *metric {
	return r.newMetric(name, help, typeCounter)
}

func (r *Registry) NewGauge(name, help string) *metric {
	return r.newMetric(name, help, typeGauge)
}

func (r *Registry) Render() string {
	r.mu.RLock()
	sorted := make([]*metric, len(r.metrics))
	copy(sorted, r.metrics)
	r.mu.RUnlock()

	var b strings.Builder
	for _, m := range sorted {
		b.WriteString(m.render())
	}
	return b.String()
}
