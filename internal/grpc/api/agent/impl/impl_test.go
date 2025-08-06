package impl

import (
	"context"
	"fmt"
	"log"
	"net"
	"strconv"
	"testing"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/agent/metrics/checker"
	server "github.com/AntonBezemskiy/go-musthave-metrics/internal/grpc/api/server/impl"
	pb "github.com/AntonBezemskiy/go-musthave-metrics/internal/grpc/protoc"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/repositories"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/server/storage"
)

func TestAddMetric(t *testing.T) {
	// функция для получения свободного порта для запуска приложений
	getFreePort := func() (int, error) {
		// Слушаем на порту 0, чтобы операционная система выбрала свободный порт
		listener, err := net.Listen("tcp", ":0")
		if err != nil {
			return 0, err
		}
		defer listener.Close()

		// Получаем назначенный системой порт
		port := listener.Addr().(*net.TCPAddr).Port
		return port, nil
	}

	// Вспомогательные функции --------------------------
	delta := func(d int64) *int64 {
		return &d
	}
	value := func(v float64) *float64 {
		return &v
	}

	// Отправка на сервер заполненный слайс метрик --------------------------
	wantSlice := []repositories.Metric{
		{
			ID:    "counter1",
			MType: "counter",
			Delta: delta(3234),
		},
		{
			ID:    "counter2",
			MType: "counter",
			Delta: delta(-23487),
		},
		{
			ID:    "gauge2",
			MType: "gauge",
			Value: value(2346.47345),
		},
		{
			ID:    "gauge3",
			MType: "gauge",
			Value: value(-2346.47345),
		},
	}
	{
		// Адрес запуска сервера -----------------------------
		serverPort, err := getFreePort()
		require.NoError(t, err)
		netAddr := fmt.Sprintf("localhost:%d", serverPort)

		// Создаю хранилище метрик
		stor := storage.NewDefaultMemStorage()

		// Запускаю сервер----------------------------------------------------------------------------
		lis, err := net.Listen("tcp", netAddr)
		require.NoError(t, err)

		grpcServer := grpc.NewServer()
		pb.RegisterServiceServer(grpcServer, server.NewServer(stor))

		reflection.Register(grpcServer)
		defer grpcServer.GracefulStop()

		go func(lis net.Listener) {
			err := grpcServer.Serve(lis)
			if err != nil {
				log.Printf("server stoped with error %v", err)
			}
		}(lis)

		// Запуск клиента
		cl, err := InitClient(netAddr)
		require.NoError(t, err)

		ctx := context.Background()

		// Тест с попыткой отправить на сервер пустой слайс метрик --------------
		err = AddMetric(ctx, cl, nil)
		require.Error(t, err)

		getSlice, err := stor.GetAllMetricsSlice(ctx)
		require.NoError(t, err)
		assert.Equal(t, []repositories.Metric{}, getSlice)

		err = AddMetric(ctx, cl, wantSlice)
		require.NoError(t, err)

		for _, metric := range wantSlice {
			getMetric, err := stor.GetMetric(ctx, metric.MType, metric.ID)
			require.NoError(t, err)
			switch metric.MType {
			case "gauge":
				value, err := strconv.ParseFloat(getMetric, 64)
				require.NoError(t, err)
				assert.Equal(t, true, checker.EqualFloat(*metric.Value, value))
			case "counter":
				delta, err := strconv.ParseInt(getMetric, 10, 64)
				require.NoError(t, err)
				assert.Equal(t, *metric.Delta, delta)
			}
		}
	}
	// Тест с попыткой отправить метрики когда сервер недоступен
	{
		// Запуск клиента
		cl, err := InitClient("localhost:8087")
		require.NoError(t, err)

		ctx := context.Background()

		// Тест с попыткой отправить на сервер пустой слайс метрик --------------
		err = AddMetric(ctx, cl, nil)
		require.Error(t, err)

		// Тест с попыткой отправить на сервер заполненный слайс метрик --------------
		err = AddMetric(ctx, cl, wantSlice)
		require.Error(t, err)
	}
}
