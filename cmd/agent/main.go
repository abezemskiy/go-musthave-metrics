package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"go.uber.org/zap"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/agent/logger"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/agent/metrics/collecter"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/agent/metrics/config"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/agent/metrics/pusher"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/agent/storage"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/agent/worker"
)

const shutdownWaitPeriod = 20 * time.Second // таймаут для graceful shutdown

func main() {
	// вывод глобальной информации о сборке
	printGlobalInfo(os.Stdout)

	parseFlags()

	metrics := storage.NewMetricsStats()
	err := run(metrics)
	if err != nil {
		log.Printf("Error initialize agent logger: %v\n", err)
	}
	log.Println("Shutdown the agent gracefully")
}

// run - будет полезна при инициализации зависимостей агента перед запуском
func run(metrics *storage.MetricsStats) error {
	if err := logger.Initialize(flagLogLevel); err != nil {
		return err
	}
	// Добавляю многопоточность
	var wg sync.WaitGroup

	// Create a context with timeout for graceful shutdown
	ctx, cancelCtx := context.WithTimeout(context.Background(), shutdownWaitPeriod)

	logger.AgentLog.Info("Running agent", zap.String("address", flagNetAddr), zap.String("rateLimit", fmt.Sprintf("%d", *rateLimit)))
	go collecter.CollectWithTimer(ctx, metrics, &wg)
	time.Sleep(50 * time.Millisecond)

	// Размер буферизованного канала равен количеству количеству одновременно исходящих запросов
	var pushTasks = make(chan worker.Task, *rateLimit)
	go GeneratePushTasks(ctx, pushTasks, "http://"+flagNetAddr, "updates/", metrics, &wg)

	log.Printf("rateLimit is: %d\n", *rateLimit)
	// создаю и запускаю воркеры, это и есть пул
	for w := 0; w < *rateLimit; w++ {
		go worker.DoWork(pushTasks, &wg)
		logger.AgentLog.Debug("start pushing worker", zap.String("worker", fmt.Sprintf("%d", w)))
	}

	// Канал для получения сигнала прерывания
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	// Блокирование до тех пор, пока не поступит сигнал о прерывании
	<-quit
	log.Println("Shutting down agent...")

	// Закрываю контекст, для остановки функции записи данных в канал для отправки на сервер
	cancelCtx()

	wg.Wait()
	return nil
}

// GeneratePushTasks - генерирует задачи для их выполнения пулом работников.
func GeneratePushTasks(ctx context.Context, tasks chan<- worker.Task, address, action string, metrics *storage.MetricsStats, wg *sync.WaitGroup) {
	wg.Add(1)
	defer wg.Done()
	defer close(tasks)

	sleepInterval := config.GetReportInterval() * time.Second
	for {
		select {
		case <-ctx.Done():
			return
		default:
			tasks <- *worker.NewTask(address, action, metrics, pusher.PrepareAndPushBatch)
			time.Sleep(sleepInterval)
		}
	}
}
