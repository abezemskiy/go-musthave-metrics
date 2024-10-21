package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/agent/handlers"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/agent/logger"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/agent/storage"
)

func main() {
	parseFlags()

	metrics := storage.NewMetricsStats()
	err := run(metrics)
	if err != nil {
		log.Printf("Error initialize agent logger: %v\n", err)
	}
}

// run - будет полезна при инициализации зависимостей агента перед запуском
func run(metrics *storage.MetricsStats) error {
	if err := logger.Initialize(flagLogLevel); err != nil {
		return err
	}
	// Добавляю многопоточность
	var wg sync.WaitGroup

	logger.AgentLog.Info("Running agent", zap.String("address", flagNetAddr), zap.String("rateLimit", fmt.Sprintf("%d", *rateLimit)))
	go handlers.CollectMetricsTimer(metrics, &wg)
	time.Sleep(50 * time.Millisecond)

	// Размер буферизованного канала равен количеству количеству одновременно исходящих запросов
	var pushTasks = make(chan handlers.Task, *rateLimit)
	go GeneratePushTasks(pushTasks, "http://"+flagNetAddr, "updates/", metrics, &wg)

	// создаю и запускаю воркеры, это и есть пул
	for w := 0; w < *rateLimit; w++ {
		go handlers.PushWorker(pushTasks, &wg)
		logger.AgentLog.Debug("start pushing worker", zap.String("worker", fmt.Sprintf("%d", w)))
	}

	wg.Wait()
	return nil
}

// GeneratePushTasks - генерирует задачи для их выполнения пулом работников.
func GeneratePushTasks(tasks chan<- handlers.Task, address, action string, metrics *storage.MetricsStats, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()
	defer close(tasks)

	sleepInterval := handlers.GetReportInterval() * time.Second
	for {
		tasks <- *handlers.NewTask(address, action, metrics, handlers.PushMetricsBatch)
		time.Sleep(sleepInterval)
	}
}
