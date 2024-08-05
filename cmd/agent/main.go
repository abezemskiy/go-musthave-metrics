package main

import (
	"log"
	"time"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/agent/handlers"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/agent/logger"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/agent/storage"
	"go.uber.org/zap"
)

func main() {
	parseFlags()

	metrics := storage.NewMetricsStats()
	err := run(metrics)
	if err != nil {
		log.Printf("Error initialize agent logger: %v\n", err)
	}
}

// функция run будет полезна при инициализации зависимостей агента перед запуском
func run(metrics *storage.MetricsStats) error {
	if err := logger.Initialize(flagLogLevel); err != nil {
		return err
	}

	logger.AgentLog.Info("Running agent", zap.String("address", flagNetAddr))
	go handlers.CollectMetricsTimer(metrics)
	time.Sleep(50 * time.Millisecond)
	handlers.PushMetricsTimer("http://"+flagNetAddr, "update", metrics)
	return nil
}
