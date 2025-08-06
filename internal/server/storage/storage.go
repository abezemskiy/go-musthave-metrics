// реализация интерфейса хранилища метрик
package storage

import (
	"context"
	"fmt"
	"sync"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/repositories"
)

// Хранилище метрик ------------------------------------------------------------------------------------

// MemStorage - реализует интерфейс repositories.ServerRepo, для возможности использования структуры в качестве хранилища метрик.
type MemStorage struct {
	sync.Mutex
	gauges   map[string]float64
	counters map[string]int64
}

// NewDefaultMemStorage - фабричная функция для создания структуры MemStorage с параметрами по умолчанию.
func NewDefaultMemStorage() *MemStorage {
	return &MemStorage{
		gauges:   make(map[string]float64),
		counters: make(map[string]int64),
	}
}

// NewDefaultMemStorage - фабричная функция для создания структуры MemStorage с принятыми параметрами.
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

// AddGauge - реализует метод AddGauge интерфейса repositories.ServerRepo.
func (storage *MemStorage) AddGauge(_ context.Context, name string, guage float64) error {
	storage.Mutex.Lock()
	defer storage.Mutex.Unlock()
	storage.gauges[name] = guage
	return nil
}

// AddCounter - реализует метод AddCounter интерфейса repositories.ServerRepo.
func (storage *MemStorage) AddCounter(_ context.Context, name string, counter int64) error {
	storage.Mutex.Lock()
	defer storage.Mutex.Unlock()
	storage.counters[name] += counter
	return nil
}

// GetMetric - реализует метод GetMetric интерфейса repositories.ServerRepo.
func (storage *MemStorage) GetMetric(_ context.Context, metricType, name string) (string, error) {
	storage.Mutex.Lock()
	defer storage.Mutex.Unlock()

	switch metricType {
	case "gauge":
		val, ok := storage.gauges[name]
		if !ok {
			return "", fmt.Errorf("metric %s of type gauge not found", name)
		}
		return fmt.Sprintf("%g", val), nil
	case "counter":
		val, ok := storage.counters[name]
		if !ok {
			return "", fmt.Errorf("metric %s of type counter not found", name)
		}
		return fmt.Sprintf("%d", val), nil
	default:
		return "", fmt.Errorf("whrong type of metric")
	}
}

// GetAllMetrics - реализует метод GetAllMetrics интерфейса repositories.ServerRepo.
func (storage *MemStorage) GetAllMetrics(_ context.Context) (string, error) {
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

// GetAllMetricsSlice - реализует метод GetAllMetricsSlice интерфейса repositories.ServerRepo.
func (storage *MemStorage) GetAllMetricsSlice(_ context.Context) ([]repositories.Metric, error) {
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

// AddMetricsFromSlice - реализует метод AddMetricsFromSlice интерфейса repositories.ServerRepo.
func (storage *MemStorage) AddMetricsFromSlice(ctx context.Context, metrics []repositories.Metric) error {
	if metrics == nil {
		return nil
	}

	for _, metric := range metrics {
		switch metric.MType {
		case "gauge":
			if metric.Value == nil {
				return fmt.Errorf("invalid metric, value of gauge metric is nil")
			}
			err := storage.AddGauge(ctx, metric.ID, *metric.Value)
			if err != nil {
				return fmt.Errorf("add gauge error: %f", err)
			}
		case "counter":
			if metric.Delta == nil {
				return fmt.Errorf("invalid metric, delta of counter metric is nil")
			}
			err := storage.AddCounter(ctx, metric.ID, *metric.Delta)
			if err != nil {
				return fmt.Errorf("add counter error: %f", err)
			}
		default:
			return fmt.Errorf("invalid metric, undefined type of metric: %s", metric.MType)
		}
	}
	return nil
}

// MemStorage_Bootstrap - реализует метод Bootstrap интерфейса repositories.ServerRepo.
func (storage *MemStorage) Bootstrap(_ context.Context) error {
	return nil
}

// Clean - очищает хранилище от данных.
func (storage *MemStorage) Clean(_ context.Context) {
	storage.counters = map[string]int64{}
	storage.gauges = map[string]float64{}
}

// Хранилище метрик -----------------------------------------------------------------------------------------
