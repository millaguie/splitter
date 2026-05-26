package health

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/user/splitter/internal/process"
	"github.com/user/splitter/internal/tor"
)

const DefaultStatusPort = 63540

type InstanceStatus struct {
	ID          int    `json:"id"`
	Country     string `json:"country"`
	State       string `json:"state"`
	SocksPort   int    `json:"socks_port"`
	ControlPort int    `json:"control_port"`
	HTTPPort    int    `json:"http_port"`
}

type SystemStatus struct {
	Timestamp        string           `json:"timestamp"`
	TorVersion       string           `json:"tor_version"`
	Features         map[string]bool  `json:"features"`
	Instances        []InstanceStatus `json:"instances"`
	TotalInstances   int              `json:"total_instances"`
	ReadyCount       int              `json:"ready_count"`
	FailedCount      int              `json:"failed_count"`
	Processes        int              `json:"process_count"`
	ProcessBreakdown map[string]int   `json:"process_breakdown,omitempty"`
}

func CollectStatus(torMgr *tor.TorManager, procMgr *process.Manager) *SystemStatus {
	status := &SystemStatus{
		Timestamp: time.Now().Format("2006-01-02 15:04:05"),
		Features:  make(map[string]bool),
	}

	if v := torMgr.GetVersion(); v != nil {
		status.TorVersion = v.String()
		status.Features["conflux"] = v.SupportsConflux()
		status.Features["http_tunnel"] = v.SupportsHTTPTunnel()
		status.Features["congestion_control"] = v.SupportsCongestionControl()
		status.Features["cgo"] = v.SupportsCGO()
	}

	instances := torMgr.GetInstances()
	status.TotalInstances = len(instances)
	status.Instances = make([]InstanceStatus, len(instances))

	for i, inst := range instances {
		status.Instances[i] = InstanceStatus{
			ID:          inst.ID,
			Country:     inst.Country,
			State:       inst.GetState().String(),
			SocksPort:   inst.SocksPort,
			ControlPort: inst.ControlPort,
			HTTPPort:    inst.HTTPPort,
		}
		switch inst.GetState() {
		case tor.StateReady:
			status.ReadyCount++
		case tor.StateFailed:
			status.FailedCount++
		}
	}

	procs := procMgr.List()
	status.Processes = len(procs)
	status.ProcessBreakdown = categorizeProcesses(procs)

	return status
}

func categorizeProcesses(procs []*process.Process) map[string]int {
	breakdown := make(map[string]int)
	for _, p := range procs {
		name := p.Name
		if idx := strings.Index(name, "-"); idx >= 0 {
			name = name[:idx]
		}
		breakdown[name]++
	}
	return breakdown
}

func StatusHandler(torMgr *tor.TorManager, procMgr *process.Manager) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		status := CollectStatus(torMgr, procMgr)
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(status); err != nil {
			http.Error(w, fmt.Sprintf("StatusHandler: %v", err), http.StatusInternalServerError)
		}
	}
}
