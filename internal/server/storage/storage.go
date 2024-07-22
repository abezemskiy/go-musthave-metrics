package storage

import "fmt"

// Хранилище метрик ------------------------------------------------------------------------------------

type MemStorage struct {
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
		result += name + " " + fmt.Sprint(val) + "\n"
	}

	for name, val := range storage.counters {
		result += name + " " + fmt.Sprint(val) + "\n"
	}
	return result
}

// Хранилище метрик -----------------------------------------------------------------------------------------
