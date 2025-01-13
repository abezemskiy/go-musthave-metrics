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
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/grpc/api/agent"
)

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
	// Проверка хранилища на nil
	if metrics == nil {
		return fmt.Errorf("storage is nil")
	}

	// инициализация логера
	if err := logger.Initialize(flagLogLevel); err != nil {
		return err
	}
	// Добавляю многопоточность
	var wg sync.WaitGroup

	// Create a context with cancel function for graceful shutdown
	ctx, cancelCtx := context.WithCancel(context.Background())

	// запуск сбора метрик через определенный промежуток времени
	logger.AgentLog.Info("Running agent", zap.String("address", flagNetAddr), zap.String("rateLimit", fmt.Sprintf("%d", *rateLimit)))
	wg.Add(1)
	go collecter.CollectWithTimer(ctx, metrics, &wg)
	time.Sleep(50 * time.Millisecond)

	// Запуск отправки метрик агентом через http или grpc
	switch flagProtocol {
	case "http":
		startHTTPAgent(ctx, metrics, &wg)
	case "grpc":
		startGRPCAgent(ctx, metrics, "AddMetric", &wg)
	default:
		log.Fatalf("wrong protocol type: %s", flagProtocol)
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

// startHTTPAgent - Запуск HTTP агента.
func startHTTPAgent(ctx context.Context, metrics *storage.MetricsStats, wg *sync.WaitGroup) {
	// Размер буферизованного канала равен количеству количеству одновременно исходящих запросов
	var pushTasks = make(chan worker.Task, *rateLimit)
	wg.Add(1)
	go GeneratePushTasks(ctx, pushTasks, "http://"+flagNetAddr, "updates/", metrics, wg)

	// создаю и запускаю воркеры, это и есть пул
	for w := 0; w < *rateLimit; w++ {
		wg.Add(1)
		go worker.DoWork(pushTasks, wg)
		logger.AgentLog.Debug("start pushing worker", zap.String("worker", fmt.Sprintf("%d", w)))
	}
}

// GeneratePushTasks - генерирует задачи для их выполнения пулом работников.
func GeneratePushTasks(ctx context.Context, tasks chan<- worker.Task, address, action string, metrics *storage.MetricsStats, wg *sync.WaitGroup) {
	defer wg.Done()
	defer close(tasks)

	sleepInterval := config.GetReportInterval() * time.Second
	for {
		select {
		case <-ctx.Done():
			return
		case tasks <- *worker.NewTask(address, action, metrics, pusher.PrepareAndPushBatch):
			time.Sleep(sleepInterval)
		}
	}
}

// startGRPCAgent - Запуск gRPC агента.
func startGRPCAgent(ctx context.Context, metrics *storage.MetricsStats, transmMethod string, wg *sync.WaitGroup) {
	var pushTasks = make(chan struct{}, *rateLimit)

	wg.Add(1)
	go GenerateGRPCPushTasks(ctx, pushTasks, wg)

	// создаю и запускаю воркеры, это и есть пул
	for w := 0; w < *rateLimit; w++ {
		wg.Add(1)
		go agent.InitWorkerAndDo(ctx, flagNetAddr, transmMethod, metrics, pushTasks, wg)
		logger.AgentLog.Info("start pushing worker", zap.String("worker", fmt.Sprintf("%d", w)))
	}
}

// GenerateGRPCPushTasks - генерирует задачи для их выполнения пулом работников.
func GenerateGRPCPushTasks(ctx context.Context, tasks chan<- struct{}, wg *sync.WaitGroup) {
	defer wg.Done()
	defer close(tasks)

	sleepInterval := config.GetReportInterval() * time.Second
	for {
		select {
		case <-ctx.Done():
			return
		case tasks <- struct{}{}:
			time.Sleep(sleepInterval)
		}
	}
}
