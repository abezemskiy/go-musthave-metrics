package storage

import (
	"runtime"
	"sync"
)

// MetricsStats структура для хранения метрик
type MetricsStats struct {
	sync.Mutex
	runtime.MemStats
	PollCount   int64
	RandomValue float64
}

// CollectMetrics собирает метрики
func (metrics *MetricsStats) CollectMetrics() {
	metrics.PollCount++
	runtime.ReadMemStats(&metrics.MemStats)
}
