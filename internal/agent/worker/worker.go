package worker

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/go-resty/resty/v2"
	"go.uber.org/zap"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/agent/errors/checker"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/agent/hasher"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/agent/logger"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/agent/storage"
)

// PushFunction - тип функции выполняющей отправку метрики.
type PushFunction = func(string, string, *storage.MetricsStats, *resty.Client) error

// RetryExecPushFunction - для повторной отправки запроса в случае, если сервер не отвечает. Установлено три дополнительных попыток.
func RetryExecPushFunction(address, action string, metrics *storage.MetricsStats, client *resty.Client, pushFunction PushFunction) {
	sleepIntervals := []time.Duration{0, 1, 3, 5}

	for i := 0; i < 4; i++ {
		logger.AgentLog.Debug(fmt.Sprintf("Push metrics to server, attemption %d", i+1))

		time.Sleep(sleepIntervals[i] * time.Second)

		err := pushFunction(address, action, metrics, client)
		if err != nil && (errors.Is(err, context.DeadlineExceeded) ||
			checker.IsConnectionRefused(err) ||
			checker.IsDBTransportError(err)) ||
			checker.IsFileLockedError(err) {
			continue
		}
		return
	}
}

// Task - структура для хранения всех необходимых параметров для отправки метрик.
// Реализация патерна worker pool.
type Task struct {
	address      string                // адрес отправки
	action       string                // http метод, например: POST
	metrics      *storage.MetricsStats // структура с собранными метриками
	pushFunction PushFunction          // функция, непосредственно выполняющая отправку
	restyClient  *resty.Client         // клиент resty
}

// NewTask - фабричная функция структуры Task.
func NewTask(address, action string, metrics *storage.MetricsStats, pushFunction PushFunction) *Task {
	return &Task{
		address:      address,
		action:       action,
		metrics:      metrics,
		pushFunction: pushFunction,
		restyClient:  resty.New(),
	}
}

// Do - метод для выполнения задачи.
func (t Task) Do() {
	// Добавляем middleware для обработки ответа
	t.restyClient.OnAfterResponse(hasher.VerifyHashMiddleware)

	RetryExecPushFunction(t.address, t.action, t.metrics, t.restyClient, t.pushFunction)
	logger.AgentLog.Debug("Running agent", zap.String("action", "push metrics"))
}

// DoWork - принимает задачу из канала и выполняет её.
func DoWork(pushTasks <-chan Task, wg *sync.WaitGroup) {
	defer wg.Done()

	for pushTask := range pushTasks {
		pushTask.Do()
	}
}
