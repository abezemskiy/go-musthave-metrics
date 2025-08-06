package agent

import (
	"context"
	"fmt"
	"log"
	"sync"

	"go.uber.org/zap"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/agent/logger"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/agent/metrics/builder"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/agent/storage"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/grpc/api/agent/impl"
)

// Worker - структура для реализации патерна worker pool.
type Worker struct {
	cl                 *impl.Client
	metrics            *storage.MetricsStats
	transmittionMethod string
}

// Do - метод для выполнения задачи.
func (w *Worker) Do(ctx context.Context) error {
	metricsSlice := builder.BuildSlice(w.metrics)

	switch w.transmittionMethod {
	case "AddMetric":
		err := impl.AddMetric(ctx, w.cl, metricsSlice)
		if err != nil {
			return fmt.Errorf("failed to send a message to the server with AddMetric method: %v", err)
		}
	default:
		return fmt.Errorf("unknown transmittion method")
	}
	return nil
}

// NewTask - фабричная функция структуры Worker.
func NewWorker(netAddr, transmittionMethod string, metrics *storage.MetricsStats) *Worker {
	cl, err := impl.InitClient(netAddr)
	// Если инициализация клиента завершилась ошибкой считаю это критической ошибкой, так как это мешает корректно запустить работу агента.
	if err != nil {
		log.Fatalf("failed to start grpc client %v", err)
	}
	return &Worker{
		cl:                 cl,
		metrics:            metrics,
		transmittionMethod: transmittionMethod,
	}
}

// InitWorkerAndDo - создает воркера, принимает задачу из канала и выполняет её.
func InitWorkerAndDo(ctx context.Context, netAddr, transmittionMethod string, metrics *storage.MetricsStats, pushTasks <-chan struct{}, wg *sync.WaitGroup) {
	defer wg.Done()

	worker := NewWorker(netAddr, transmittionMethod, metrics)

	for range pushTasks {
		err := worker.Do(ctx)

		if err != nil {
			logger.AgentLog.Error("do work error", zap.String("error", err.Error()))
		}
	}
}
