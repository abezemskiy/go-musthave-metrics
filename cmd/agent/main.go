package main

import (
	"time"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/agenthandlers"
)

func main() {
	parseFlags()

	var metrics agenthandlers.MetricsStats
	go agenthandlers.CollectMetricsTimer(&metrics)
	time.Sleep(50 * time.Millisecond)
	agenthandlers.PushMetricsTimer("http://"+flagNetAddr, "update", &metrics)
}
