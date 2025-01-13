package main

import (
	"context"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/agent/logger"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/agent/storage"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/agent/worker"
)

func TestRun(t *testing.T) {
	{
		metrics := storage.NewMetricsStats()

		err := logger.Initialize("debug")
		require.NoError(t, err)

		// Запускаем run в отдельной горутине
		go func() {
			time.Sleep(100 * time.Millisecond) // даю run время для старта
			// имитирую сигнал завершения
			p, _ := os.FindProcess(os.Getpid())
			_ = p.Signal(os.Interrupt)
		}()

		err = run(metrics)
		require.NoError(t, err)

		// проверяю, что контекст завершился
		require.NotNil(t, logger.AgentLog, "AgentLog should be initialized")
	}
	{
		err := run(nil)
		require.Error(t, err)
	}
}

func TestGeneratePushTasks(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	tasks := make(chan worker.Task, 10)

	mockMetrics := storage.NewMetricsStats()
	var wg sync.WaitGroup

	go GeneratePushTasks(ctx, tasks, "http://localhost", "updates/", mockMetrics, &wg)

	// Проверяем, что задачи генерируются в канал
	select {
	case <-ctx.Done():
		t.Fatal("context finished before task generation")
	case task := <-tasks:
		require.NotNil(t, task, "Generated task should not be nil")
	}
}
