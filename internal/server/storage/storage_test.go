package storage

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/repositories"
)

func TestNewDefaultMemStorage(t *testing.T) {

	tests := []struct {
		name string
		want *MemStorage
	}{
		{
			name: "Default test #1",
			want: &MemStorage{
				gauges:   map[string]float64{},
				counters: map[string]int64{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewDefaultMemStorage(); !reflect.DeepEqual(got.counters, tt.want.counters) {
				t.Errorf("NewDefaultMemStorage() = %v, want %v", got.counters, tt.want.counters)
			}
			if got := NewDefaultMemStorage(); !reflect.DeepEqual(got.gauges, tt.want.gauges) {
				t.Errorf("NewDefaultMemStorage() = %v, want %v", got.gauges, tt.want.gauges)
			}
		})
	}
}

func TestNewMemStorage(t *testing.T) {

	type args struct {
		gaugesArg   map[string]float64
		countersArg map[string]int64
	}
	tests := []struct {
		name string
		args args
		want *MemStorage
	}{
		{
			name: "Args function is nil",
			args: args{
				gaugesArg:   nil,
				countersArg: nil,
			},
			want: &MemStorage{
				gauges:   map[string]float64{},
				counters: map[string]int64{},
			},
		},
		{
			name: "Args not nil #1",
			args: args{
				gaugesArg:   map[string]float64{"gauge1": 1.14},
				countersArg: map[string]int64{"counter1": 5},
			},
			want: &MemStorage{
				gauges:   map[string]float64{"gauge1": 1.14},
				counters: map[string]int64{"counter1": 5},
			},
		},
		{
			name: "Left arg is nil #2",
			args: args{
				gaugesArg:   nil,
				countersArg: map[string]int64{"counter1": 5},
			},
			want: &MemStorage{
				gauges:   map[string]float64{},
				counters: map[string]int64{"counter1": 5},
			},
		},
		{
			name: "Right args is nil #3",
			args: args{
				gaugesArg:   map[string]float64{"gauge1": 1.14},
				countersArg: nil,
			},
			want: &MemStorage{
				gauges:   map[string]float64{"gauge1": 1.14},
				counters: map[string]int64{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewMemStorage(tt.args.gaugesArg, tt.args.countersArg); !reflect.DeepEqual(got.counters, tt.want.counters) {
				t.Errorf("NewMemStorage() = %v, want %v", got.counters, tt.want.counters)
			}
			if got := NewMemStorage(tt.args.gaugesArg, tt.args.countersArg); !reflect.DeepEqual(got.gauges, tt.want.gauges) {
				t.Errorf("NewMemStorage() = %v, want %v", got.gauges, tt.want.gauges)
			}
		})
	}
}

func TestMemStorageAddGauge(t *testing.T) {
	type args struct {
		stor  *MemStorage
		name  string
		value float64
	}
	tests := []struct {
		name string
		args args
		want *MemStorage
	}{
		{
			name: "Add test #1",
			args: args{
				stor: &MemStorage{
					gauges:   map[string]float64{"gauge1": 1.14},
					counters: map[string]int64{"counter1": 5},
				},
				name:  "gauge2",
				value: 3.14,
			},
			want: &MemStorage{
				gauges:   map[string]float64{"gauge1": 1.14, "gauge2": 3.14},
				counters: map[string]int64{"counter1": 5},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.args.stor.AddGauge(context.Background(), tt.args.name, tt.args.value)
			require.NoError(t, err)
			if !reflect.DeepEqual(tt.args.stor.counters, tt.want.counters) {
				t.Errorf("NewDefaultMemStorage() = %v, want %v", tt.args.stor.counters, tt.want.counters)
			}
			if !reflect.DeepEqual(tt.args.stor.gauges, tt.want.gauges) {
				t.Errorf("NewDefaultMemStorage() = %v, want %v", tt.args.stor.gauges, tt.want.gauges)
			}
		})
	}
}

func TestMemStorageAddCounter(t *testing.T) {
	type args struct {
		stor  *MemStorage
		name  string
		value int64
	}
	tests := []struct {
		name string
		args args
		want *MemStorage
	}{
		{
			name: "Add test #1",
			args: args{
				stor: &MemStorage{
					gauges:   map[string]float64{"gauge1": 1.14},
					counters: map[string]int64{"counter1": 5},
				},
				name:  "counter2",
				value: 6,
			},
			want: &MemStorage{
				gauges:   map[string]float64{"gauge1": 1.14},
				counters: map[string]int64{"counter1": 5, "counter2": 6},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.args.stor.AddCounter(context.Background(), tt.args.name, tt.args.value)
			require.NoError(t, err)
			if !reflect.DeepEqual(tt.args.stor.counters, tt.want.counters) {
				t.Errorf("NewDefaultMemStorage() = %v, want %v", tt.args.stor.counters, tt.want.counters)
			}
			if !reflect.DeepEqual(tt.args.stor.gauges, tt.want.gauges) {
				t.Errorf("NewDefaultMemStorage() = %v, want %v", tt.args.stor.gauges, tt.want.gauges)
			}
		})
	}
}

func TestMemStorageGetMetric(t *testing.T) {
	type fields struct {
		gauges   map[string]float64
		counters map[string]int64
	}
	type args struct {
		metricType string
		name       string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
	}{
		{
			name: "Correct get gauge #1",
			fields: fields{
				gauges:   map[string]float64{"gauge1": 17.77, "gauge2": 14.9},
				counters: map[string]int64{"counter1": 100, "counter2": 7},
			},
			args: args{
				metricType: "gauge",
				name:       "gauge1",
			},
			want:    fmt.Sprintf("%g", 17.77),
			wantErr: false,
		},
		{
			name: "Correct get gauge #2",
			fields: fields{
				gauges:   map[string]float64{"gauge1": 17.77, "gauge2": 14.9},
				counters: map[string]int64{"counter1": 100, "counter2": 7},
			},
			args: args{
				metricType: "gauge",
				name:       "gauge2",
			},
			want:    fmt.Sprintf("%g", 14.9),
			wantErr: false,
		},
		{
			name: "Whrong get gauge #1",
			fields: fields{
				gauges:   map[string]float64{"gauge1": 17.77, "gauge2": 14.9},
				counters: map[string]int64{"counter1": 100, "counter2": 7},
			},
			args: args{
				metricType: "gauge",
				name:       "gauge3",
			},
			want:    "",
			wantErr: true,
		},
		{
			name: "Correct get counter #1",
			fields: fields{
				gauges:   map[string]float64{"gauge1": 17.77, "gauge2": 14.9},
				counters: map[string]int64{"counter1": 100, "counter2": 7},
			},
			args: args{
				metricType: "counter",
				name:       "counter1",
			},
			want:    "100",
			wantErr: false,
		},
		{
			name: "Whrong get counter #1",
			fields: fields{
				gauges:   map[string]float64{"gauge1": 17.77, "gauge2": 14.9},
				counters: map[string]int64{"counter1": 100, "counter2": 7},
			},
			args: args{
				metricType: "counter",
				name:       "counter100",
			},
			want:    "",
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := &MemStorage{
				gauges:   tt.fields.gauges,
				counters: tt.fields.counters,
			}
			got, err := storage.GetMetric(context.Background(), tt.args.metricType, tt.args.name)
			if !tt.wantErr {
				assert.NoError(t, err)
			}
			assert.Equal(t, got, tt.want)
		})
	}
}

func TestMemStorageGetAllMetrics(t *testing.T) {
	type fields struct {
		gauges   map[string]float64
		counters map[string]int64
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		{
			name: "Correct get metrics #1",
			fields: fields{
				gauges:   map[string]float64{"gauge1": 17.77},
				counters: map[string]int64{},
			},
			want: "gauge1: 17.77\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := &MemStorage{
				gauges:   tt.fields.gauges,
				counters: tt.fields.counters,
			}
			res, err := storage.GetAllMetrics(context.Background())
			require.NoError(t, err)
			assert.Equal(t, tt.want, res)
		})
	}
}

func TestMemStorage_Bootstrap(t *testing.T) {
	stor := NewDefaultMemStorage()
	ctx := context.Background()
	require.NoError(t, stor.Bootstrap(ctx))
}

func TestMemStorage_Clean(t *testing.T) {
	stor := NewDefaultMemStorage()
	ctx := context.Background()

	err := stor.AddCounter(ctx, "first counter", 3252)
	require.NoError(t, err)

	err = stor.AddGauge(ctx, "first gauge", 785723.3242)
	require.NoError(t, err)
	stor.Clean(ctx)

	_, err = stor.GetMetric(ctx, "counter", "first counter")
	require.Error(t, err)
	_, err = stor.GetMetric(ctx, "gauge", "first gauge")
	require.Error(t, err)
}

func TestAddMetricsFromSlice(t *testing.T) {
	stor := NewDefaultMemStorage()

	// Error: metrics is nil
	err := stor.AddMetricsFromSlice(context.Background(), nil)
	require.NoError(t, err)

	// Gauge value is nil
	err = stor.AddMetricsFromSlice(context.Background(), []repositories.Metric{{
		MType: "gauge",
		ID:    "test gauge",
	}})
	require.Error(t, err)

	// Counter delta is nil
	err = stor.AddMetricsFromSlice(context.Background(), []repositories.Metric{{
		MType: "counter",
		ID:    "test counter",
	}})
	require.Error(t, err)
}
