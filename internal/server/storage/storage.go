// реализация интерфейса хранилища метрик
package storage

import (
	"context"
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

func (storage *MemStorage) AddGauge(ctx context.Context, name string, guage float64) error {
	storage.Mutex.Lock()
	defer storage.Mutex.Unlock()
	storage.gauges[name] = guage
	return nil
}

func (storage *MemStorage) AddCounter(ctx context.Context, name string, counter int64) error {
	storage.Mutex.Lock()
	defer storage.Mutex.Unlock()
	storage.counters[name] += counter
	return nil
}

func (storage *MemStorage) GetMetric(ctx context.Context, metricType, name string) (string, error) {
	storage.Mutex.Lock()
	defer storage.Mutex.Unlock()

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

func (storage *MemStorage) GetAllMetrics(ctx context.Context) (string, error) {
	storage.Mutex.Lock()
	defer storage.Mutex.Unlock()

	var result string
	for name, val := range storage.gauges {
		result += fmt.Sprintf("%s: %g\n", name, val)
	}

	for name, val := range storage.counters {
		result += fmt.Sprintf("%s: %d\n", name, val)
	}
	return result, nil
}

func (storage *MemStorage) GetAllMetricsSlice(ctx context.Context) ([]repositories.Metric, error) {
	storage.Mutex.Lock()
	defer storage.Mutex.Unlock()

	result := make([]repositories.Metric, 0)
	for name, value := range storage.gauges {
		metric := repositories.Metric{
			ID:    name,
			MType: "gauge",
			Value: &value,
		}
		result = append(result, metric)
	}
	for name, delta := range storage.counters {
		metric := repositories.Metric{
			ID:    name,
			MType: "counter",
			Delta: &delta,
		}
		result = append(result, metric)
	}
	return result, nil
}

func (storage *MemStorage) AddMetricsFromSlice(ctx context.Context, metrics []repositories.Metric) error {
	if metrics == nil {
		return nil
	}

	for _, metric := range metrics {
		if metric.MType == "gauge" {
			if metric.Value == nil {
				return fmt.Errorf("invalid metric, value of gauge metric is nil")
			}
			err := storage.AddGauge(ctx, metric.ID, *metric.Value)
			if err != nil {
				return fmt.Errorf("add gauge error: %f", err)
			}
		} else if metric.MType == "counter" {
			if metric.Delta == nil {
				return fmt.Errorf("invalid metric, delta of counter metric is nil")
			}
			err := storage.AddCounter(ctx, metric.ID, *metric.Delta)
			if err != nil {
				return fmt.Errorf("add counter error: %f", err)
			}
		} else {
			return fmt.Errorf("invalid metric, undefined type of metric: %s", metric.MType)
		}
	}
	return nil
}

func (storage *MemStorage) Bootstrap(ctx context.Context) error {
	return nil
}

// Хранилище метрик -----------------------------------------------------------------------------------------
