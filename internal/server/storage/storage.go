package storage

import (
	"fmt"
	"sync"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/repositories"
)

// Хранилище метрик ------------------------------------------------------------------------------------

type MemStorage struct {
	sync.Mutex
	gauges   map[string]float64
	counters map[string]int64
}

func NewDefaultMemStorage() *MemStorage {
	return &MemStorage{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

func NewMemStorage(gaugesArg map[string]float64, countersArg map[string]int64) *MemStorage {
	if gaugesArg == nil {
		gaugesArg = make(map[string]float64)
	}
	if countersArg == nil {
		countersArg = make(map[string]int64)
	}
	return &MemStorage{
		gauges:   gaugesArg,
		counters: countersArg,
	}
}

func (storage *MemStorage) AddGauge(name string, guage float64) {
	storage.gauges[name] = guage
}

func (storage *MemStorage) AddCounter(name string, counter int64) {
	storage.counters[name] += counter
}

func (storage *MemStorage) GetMetric(metricType, name string) (string, error) {

	if metricType == "gauge" {
		val, ok := storage.gauges[name]
		if !ok {
			return "", fmt.Errorf("metric %s of type gauge not found", name)
		}
		return fmt.Sprintf("%g", val), nil
	}

	if metricType == "counter" {
		val, ok := storage.counters[name]
		if !ok {
			return "", fmt.Errorf("metric %s of type counter not found", name)
		}
		return fmt.Sprintf("%d", val), nil
	}
	return "", fmt.Errorf("whrong type of metric")
}

func (storage *MemStorage) GetAllMetrics() string {
	var result string

	for name, val := range storage.gauges {
		result += fmt.Sprintf("%s: %g\n", name, val)
	}

	for name, val := range storage.counters {
		result += fmt.Sprintf("%s: %d\n", name, val)
	}
	return result
}

func (storage *MemStorage) AddMetricsFromSlice(metrics []repositories.Metrics) error {
	if metrics == nil {
		return nil
	}

	for _, metric := range metrics {
		if metric.MType == "gauge" {
			if metric.Value == nil {
				return fmt.Errorf("invalid metric, value of gauge metric is nil")
			}
			storage.AddGauge(metric.ID, *metric.Value)
		} else if metric.MType == "counter" {
			if metric.Delta == nil {
				return fmt.Errorf("invalid metric, delta of counter metric is nil")
			}
			storage.AddCounter(metric.ID, *metric.Delta)
		} else {
			return fmt.Errorf("invalid metric, undefined type of metric: %s", metric.MType)
		}
	}
	return nil
}

func (storage *MemStorage) GetCounters() map[string]int64{
	return storage.counters
}
func (storage *MemStorage) GetGauges() map[string]float64{
	return storage.gauges
}

// Хранилище метрик -----------------------------------------------------------------------------------------
