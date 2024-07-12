package repositories

import (
	"fmt"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
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
			if got := NewDefaultMemStorage(); !reflect.DeepEqual(*got, *tt.want) {
				t.Errorf("NewDefaultMemStorage() = %v, want %v", *got, *tt.want)
			}
		})
	}
}

func TestNewMemStorage(t *testing.T) {

	type args struct {
		gauges_   map[string]float64
		counters_ map[string]int64
	}
	tests := []struct {
		name string
		args args
		want *MemStorage
	}{
		{
			name: "Args function is nil",
			args: args{
				gauges_:   nil,
				counters_: nil,
			},
			want: &MemStorage{
				gauges:   map[string]float64{},
				counters: map[string]int64{},
			},
		},
		{
			name: "Args not nil #1",
			args: args{
				gauges_:   map[string]float64{"gauge1": 1.14},
				counters_: map[string]int64{"counter1": 5},
			},
			want: &MemStorage{
				gauges:   map[string]float64{"gauge1": 1.14},
				counters: map[string]int64{"counter1": 5},
			},
		},
		{
			name: "Left arg is nil #2",
			args: args{
				gauges_:   nil,
				counters_: map[string]int64{"counter1": 5},
			},
			want: &MemStorage{
				gauges:   map[string]float64{},
				counters: map[string]int64{"counter1": 5},
			},
		},
		{
			name: "Right args is nil #3",
			args: args{
				gauges_:   map[string]float64{"gauge1": 1.14},
				counters_: nil,
			},
			want: &MemStorage{
				gauges:   map[string]float64{"gauge1": 1.14},
				counters: map[string]int64{},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewMemStorage(tt.args.gauges_, tt.args.counters_); !reflect.DeepEqual(*got, *tt.want) {
				t.Errorf("NewMemStorage() = %v, want %v", *got, *tt.want)
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
			tt.args.stor.AddGauge(tt.args.name, tt.args.value)
			if !reflect.DeepEqual(*tt.args.stor, *tt.want) {
				t.Errorf("NewDefaultMemStorage() = %v, want %v", *tt.args.stor, *tt.want)
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
			tt.args.stor.AddCounter(tt.args.name, tt.args.value)
			if !reflect.DeepEqual(*tt.args.stor, *tt.want) {
				t.Errorf("NewDefaultMemStorage() = %v, want %v", *tt.args.stor, *tt.want)
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
			got, err := storage.GetMetric(tt.args.metricType, tt.args.name)
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
			want: "gauge1 17.77\n",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := &MemStorage{
				gauges:   tt.fields.gauges,
				counters: tt.fields.counters,
			}
			assert.Equal(t, tt.want, storage.GetAllMetrics())
		})
	}
}
