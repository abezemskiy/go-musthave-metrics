package main

import (
	"time"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/agent/handlers"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/agent/storage"
)

func main() {
	parseFlags()

	var metrics storage.MetricsStats
	go handlers.CollectMetricsTimer(&metrics)
	time.Sleep(50 * time.Millisecond)
	handlers.PushMetricsTimer("http://"+flagNetAddr, "update", &metrics)
}
