package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCollectMetrics(t *testing.T) {

	metrics := &MetricsStats{}

	tests := []struct {
		name string
		arg  *MetricsStats
		want int64
	}{
		{name: "Counter test #1",
			arg:  metrics,
			want: 1,
		},
		{name: "Counter test #2",
			arg:  metrics,
			want: 1,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.arg.CollectMetrics()
			assert.Equal(t, tt.want, tt.arg.PollCount)
		})
	}
}

func TestGetMetricString(t *testing.T) {
	metrics := NewMetricsStats()
	metrics.CollectMetrics()
	for _, metricName := range GaugeMetrics {
		t.Run(metricName, func(t *testing.T) {
			typeMetric, value, err := metrics.GetMetricString(metricName)
			assert.Equal(t, typeMetric, "gauge")
			assert.NotEqual(t, "", value)
			require.NoError(t, err)
		})
	}
	_, _, err := metrics.GetMetricString("wrong metric")
	require.Error(t, err)
}
