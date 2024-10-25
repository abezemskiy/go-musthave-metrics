package main

import (
	"fmt"
	"log"
	"sync"
	"time"

	"go.uber.org/zap"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/agent/logger"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/agent/metrics/collecter"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/agent/metrics/config"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/agent/metrics/pusher"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/agent/storage"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/agent/worker"
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
	go collecter.CollectWithTimer(metrics, &wg)
	time.Sleep(50 * time.Millisecond)

	// Размер буферизованного канала равен количеству количеству одновременно исходящих запросов
	var pushTasks = make(chan worker.Task, *rateLimit)
	go GeneratePushTasks(pushTasks, "http://"+flagNetAddr, "updates/", metrics, &wg)

	// создаю и запускаю воркеры, это и есть пул
	for w := 0; w < *rateLimit; w++ {
		go worker.DoWork(pushTasks, &wg)
		logger.AgentLog.Debug("start pushing worker", zap.String("worker", fmt.Sprintf("%d", w)))
	}

	wg.Wait()
	return nil
}

// GeneratePushTasks - генерирует задачи для их выполнения пулом работников.
func GeneratePushTasks(tasks chan<- worker.Task, address, action string, metrics *storage.MetricsStats, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()
	defer close(tasks)

	sleepInterval := config.GetReportInterval() * time.Second
	for {
		tasks <- *worker.NewTask(address, action, metrics, pusher.PrepareAndPushBatch)
		time.Sleep(sleepInterval)
	}
}
