package builder

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/repositories"
)

func TestBuild(t *testing.T) {
	deltaPointer := func(delta int64) *int64 {
		return &delta
	}
	valuePointer := func(value float64) *float64 {
		return &value
	}
	type args struct {
		typeMetric  string
		nameMetric  string
		valueMetric string
	}
	tests := []struct {
		name       string
		args       args
		wantMetric repositories.Metric
		wantErr    bool
	}{
		{
			name: "success counter1",
			args: args{
				typeMetric:  "counter",
				nameMetric:  "counter1",
				valueMetric: "95738",
			},
			wantMetric: repositories.Metric{
				ID:    "counter1",
				MType: "counter",
				Delta: deltaPointer(95738),
				Value: nil,
			},
			wantErr: false,
		},
		{
			name: "error counter2",
			args: args{
				typeMetric:  "counter",
				nameMetric:  "counter2",
				valueMetric: "errorString",
			},
			wantMetric: repositories.Metric{},
			wantErr:    true,
		},
		{
			name: "success gauge1",
			args: args{
				typeMetric:  "gauge",
				nameMetric:  "gauge1",
				valueMetric: "95738.23598",
			},
			wantMetric: repositories.Metric{
				ID:    "gauge1",
				MType: "gauge",
				Value: valuePointer(95738.23598),
				Delta: nil,
			},
			wantErr: false,
		},
		{
			name: "error gauge2",
			args: args{
				typeMetric:  "gauge",
				nameMetric:  "gauge2",
				valueMetric: "errorString",
			},
			wantMetric: repositories.Metric{},
			wantErr:    true,
		},
		{
			name: "error wrongType",
			args: args{
				typeMetric:  "wrongType",
				nameMetric:  "counter1",
				valueMetric: "95738",
			},
			wantMetric: repositories.Metric{},
			wantErr:    true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotMetric, err := Build(tt.args.typeMetric, tt.args.nameMetric, tt.args.valueMetric)
			if tt.wantErr == true {
				require.Error(t, err)
				return
			}
			assert.Equal(t, gotMetric, tt.wantMetric)
		})
	}
}

func TestBuildSlice(t *testing.T) {
	// Тест с непроинициализированной структурой
	slice := BuildSlice(nil)
	assert.Equal(t, []repositories.Metric{}, slice)
}
