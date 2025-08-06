package checker

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/repositories"
)

func TestEqualFloat(t *testing.T) {
	tests := []struct {
		name string
		n1   float64
		n2   float64
		want bool
	}{
		{
			name: "#1",
			n1:   13535.346,
			n2:   13535.346,
			want: true,
		},
		{
			name: "#2",
			n1:   13535.346,
			n2:   346.12,
			want: false,
		},
		{
			name: "#3",
			n1:   13535.346,
			n2:   -13535.346,
			want: false,
		},
		{
			name: "#4",
			n1:   -13535.346,
			n2:   -13535.346,
			want: true,
		},
		{
			name: "#5",
			n1:   0.0,
			n2:   0.0,
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := EqualFloat(tt.n1, tt.n2)
			require.Equal(t, tt.want, res)
		})
	}
}

func TestEqual(t *testing.T) {
	deltaPointer := func(delta int64) *int64 {
		return &delta
	}
	valuePointer := func(value float64) *float64 {
		return &value
	}

	tests := []struct {
		name string
		m1   repositories.Metric
		m2   repositories.Metric
		want bool
	}{
		{
			name: "#1",
			m1: repositories.Metric{
				ID:    "metric1",
				MType: "counter",
				Delta: deltaPointer(324324),
			},
			m2: repositories.Metric{
				ID:    "metric1",
				MType: "counter",
				Delta: deltaPointer(324324),
			},
			want: true,
		},
		{
			name: "#2",
			m1: repositories.Metric{
				ID:    "metric1",
				MType: "counter",
				Delta: deltaPointer(324324),
			},
			m2: repositories.Metric{
				ID:    "different ID",
				MType: "counter",
				Delta: deltaPointer(324324),
			},
			want: false,
		},
		{
			name: "#3",
			m1: repositories.Metric{
				ID:    "metric1",
				MType: "counter",
				Delta: deltaPointer(324324),
			},
			m2: repositories.Metric{
				ID:    "metric1",
				MType: "different type",
				Delta: deltaPointer(324324),
			},
			want: false,
		},
		{
			name: "#4",
			m1: repositories.Metric{
				ID:    "metric1",
				MType: "counter",
				Delta: deltaPointer(324324),
			},
			m2: repositories.Metric{
				ID:    "metric1",
				MType: "counter",
				Delta: deltaPointer(235),
			},
			want: false,
		},
		{
			name: "#5",
			m1: repositories.Metric{
				ID:    "metric1",
				MType: "gauge",
				Value: valuePointer(235.3253),
			},
			m2: repositories.Metric{
				ID:    "metric1",
				MType: "gauge",
				Value: valuePointer(235.3253),
			},
			want: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			res := Equal(tt.m1, tt.m2)
			require.Equal(t, tt.want, res)
		})
	}
}
