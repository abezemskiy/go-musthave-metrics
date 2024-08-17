package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
