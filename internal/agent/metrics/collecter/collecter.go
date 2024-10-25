package collecter

import (
	"sync"
	"time"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/agent/metrics/config"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/agent/storage"
)

// CollectWithTimer запускает сбор метрик через заданный интервал времени.
func CollectWithTimer(metrics *storage.MetricsStats, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()

	sleepInterval := config.GetPollInterval() * time.Second
	for {
		config.SyncCollectMetrics(metrics)
		time.Sleep(sleepInterval)
	}
}
