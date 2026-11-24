package drivers

import (
	"strings"
	"testing"

	"github.com/lxc/incus/v7/internal/server/metrics"
	"github.com/lxc/incus/v7/shared/api"
)

func TestQemuMetricSetFromAPICPUs(t *testing.T) {
	d := &qemu{common: common{name: "v1", project: api.Project{Name: "default"}}}

	tests := []struct {
		name     string
		metrics  metrics.Metrics
		expected string
	}{
		{
			name:     "counted from the online vCPUs",
			metrics:  metrics.Metrics{CPU: []metrics.CPUMetrics{{CPU: "cpu0"}, {CPU: "cpu1"}}},
			expected: `incus_cpu_effective_total{name="v1",project="default",type="virtual-machine"} 2`,
		},
		{
			name:     "reported by the source",
			metrics:  metrics.Metrics{CPUs: 4, CPU: []metrics.CPUMetrics{{CPU: "cpu0"}}},
			expected: `incus_cpu_effective_total{name="v1",project="default",type="virtual-machine"} 4`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			set, err := d.metricSetFromAPI(&tt.metrics)
			if err != nil {
				t.Fatal(err)
			}

			if !strings.Contains(set.String(), tt.expected) {
				t.Fatalf("Expected %q in:\n%s", tt.expected, set.String())
			}
		})
	}
}
