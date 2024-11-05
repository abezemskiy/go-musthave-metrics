package config

import (
	"time"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/agent/storage"
)

var (
	pollInterval   time.Duration = 2
	reportInterval time.Duration = 10
	contextTimeout time.Duration = 500 * time.Millisecond
)

// SetPollInterval устанавливает интервал между сбором.
func SetPollInterval(interval time.Duration) {
	pollInterval = interval
}

// GetPollInterval - функция для получения интервала сбора метрик.
func GetPollInterval() time.Duration {
	return pollInterval
}

// SetReportInterval устанавливает интервал между отправками метрик на сервер.
func SetReportInterval(interval time.Duration) {
	reportInterval = interval
}

// GetReportInterval - функция для получения интервала отправки метрик на сервер.
func GetReportInterval() time.Duration {
	return reportInterval
}

// SetContextTimeout - установка таймаута.
func SetContextTimeout(timeout time.Duration) {
	contextTimeout = timeout
}

// GetContextTimeout - получение таймаута.
func GetContextTimeout() time.Duration {
	return contextTimeout
}

// SyncCollectMetrics - собирает метрики.
func SyncCollectMetrics(metrics *storage.MetricsStats) {
	metrics.CollectMetrics()
}
