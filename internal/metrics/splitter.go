package metrics

type SplitterMetrics struct {
	InstancesTotal       *metric
	InstancesActive      *metric
	InstanceState        *metric
	InstanceCountry      *metric
	CircuitsTotal        *metric
	CircuitRenewalsTotal *metric
	ErrorsTotal          *metric
	BootstrapProgress    *metric

	registry *Registry
}

func NewSplitterMetrics(r *Registry) *SplitterMetrics {
	return &SplitterMetrics{
		registry:             r,
		InstancesTotal:       r.NewGauge("splitter_instances_total", "Total number of tor instances"),
		InstancesActive:      r.NewGauge("splitter_instances_active", "Number of instances in ready state"),
		InstanceState:        r.NewGauge("splitter_instance_state", "Per-instance state (1=current state)"),
		InstanceCountry:      r.NewGauge("splitter_instance_country", "Per-instance country assignment"),
		CircuitsTotal:        r.NewCounter("splitter_circuits_total", "Total circuits created"),
		CircuitRenewalsTotal: r.NewCounter("splitter_circuit_renewals_total", "Per-instance NEWNYM count"),
		ErrorsTotal:          r.NewCounter("splitter_errors_total", "Total errors encountered"),
		BootstrapProgress:    r.NewGauge("splitter_bootstrap_progress", "Per-instance bootstrap progress 0-100"),
	}
}

func (sm *SplitterMetrics) SetInstanceCount(total, active int) {
	sm.InstancesTotal.Set(float64(total))
	sm.InstancesActive.Set(float64(active))
}

func (sm *SplitterMetrics) SetInstanceState(instanceID, state string) {
	sm.InstanceState.SetWithLabels(1, "instance_id", instanceID, "state", state)
}

func (sm *SplitterMetrics) SetInstanceCountry(instanceID, country string) {
	sm.InstanceCountry.SetWithLabels(1, "instance_id", instanceID, "country", country)
}

func (sm *SplitterMetrics) IncCircuits() {
	sm.CircuitsTotal.Inc()
}

func (sm *SplitterMetrics) IncCircuitRenewal(instanceID string) {
	sm.CircuitRenewalsTotal.IncWithLabels("instance_id", instanceID)
}

func (sm *SplitterMetrics) IncErrors() {
	sm.ErrorsTotal.Inc()
}

func (sm *SplitterMetrics) SetBootstrapProgress(instanceID string, progress float64) {
	sm.BootstrapProgress.SetWithLabels(progress, "instance_id", instanceID)
}

func (sm *SplitterMetrics) Registry() *Registry {
	return sm.registry
}
