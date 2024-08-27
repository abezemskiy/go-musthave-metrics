package main

import (
	"fmt"
	"log"
	"sync"
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
	//handlers.PushMetricsTimer("http://"+flagNetAddr, "updates/", metrics)

	// Добавляю многопоточность
	// Размер буферизованного канала равен количеству количеству одновременно исходящих запросов
	var wg sync.WaitGroup

	var pushTasks = make(chan handlers.Task, *rateLimit)
	go GeneratePushTasks(pushTasks, "http://"+flagNetAddr, "updates/", metrics)

	// создаю и запускаю воркеры, это и есть пул
	for w := 0; w < *rateLimit; w++ {
		go handlers.PushWorker(pushTasks, &wg)
		logger.AgentLog.Debug("start pushing worker", zap.String("worker", fmt.Sprintf("%d", w)))
	}

	//go func() {
	wg.Wait()
	//}()
	return nil
}

func GeneratePushTasks(tasks chan<- handlers.Task, address, action string, metrics *storage.MetricsStats) {
	defer close(tasks)

	sleepInterval := handlers.GetReportInterval() * time.Second
	for {
		tasks <- *handlers.NewTask(address, action, metrics, handlers.PushMetricsBatch)
		time.Sleep(sleepInterval)
	}
}
