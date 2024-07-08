package agenthandlers

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
			want: 2,
		},
		{name: "Counter test #3",
			arg:  metrics,
			want: 3,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			CollectMetrics(tt.arg)
			assert.Equal(t, tt.want, tt.arg.PollCount)
		})
	}
}
