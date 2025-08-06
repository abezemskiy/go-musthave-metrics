package impl

import (
	"context"
	"database/sql"
	"fmt"
	"net"
	"testing"

	"log"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pb "github.com/AntonBezemskiy/go-musthave-metrics/internal/grpc/protoc"
	pbModel "github.com/AntonBezemskiy/go-musthave-metrics/internal/grpc/protoc/model"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/server/pg"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
)

func TestServer_AddMetric(t *testing.T) {
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

	{
		// Вспомогательные функции --------------------------
		delta := func(d int64) *int64 {
			return &d
		}
		value := func(v float64) *float64 {
			return &v
		}

		// Адрес запуска сервера -----------------------------
		serverPort, err := getFreePort()
		require.NoError(t, err)
		netAddr := fmt.Sprintf("localhost:%d", serverPort)

		// Функция для инициализации клиента
		initClient := func() (pb.ServiceClient, error) {
			conn, err := grpc.NewClient(netAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				return nil, fmt.Errorf("failed to create a new client: %w", err)
			}
			client := pb.NewServiceClient(conn)
			return client, err
		}

		// Функция для очистки данных в базе
		cleanBD := func(dsn string) {
			// очищаю данные в тестовой бд------------------------------------------------------
			// создаём соединение с СУБД PostgreSQL
			conn, err := sql.Open("pgx", dsn)
			require.NoError(t, err)
			defer conn.Close()

			// Проверка соединения с БД
			ctx := context.Background()
			err = conn.PingContext(ctx)
			require.NoError(t, err)

			// создаем экземпляр хранилища pg
			stor := pg.NewStore(conn)
			err = stor.Bootstrap(ctx)
			require.NoError(t, err)
			err = stor.Disable(ctx)
			require.NoError(t, err)
		}
		databaseDsn := "host=localhost user=benchmarkmetrics password=password dbname=benchmarkmetrics sslmode=disable"

		// Очищаю данные в БД от предыдущих запусков
		cleanBD(databaseDsn)
		// Очистка данных в БД после работы теста
		defer cleanBD(databaseDsn)

		// создаём соединение с СУБД PostgreSQL
		conn, err := sql.Open("pgx", databaseDsn)
		require.NoError(t, err)
		defer conn.Close()

		// Проверка соединения с БД
		ctx := context.Background()
		err = conn.PingContext(ctx)
		require.NoError(t, err)

		// создаем экземпляр хранилища pg
		stor := pg.NewStore(conn)
		err = stor.Bootstrap(ctx)
		require.NoError(t, err)

		// Запускаю сервер----------------------------------------------------------------------------
		lis, err := net.Listen("tcp", netAddr)
		require.NoError(t, err)

		grpcServer := grpc.NewServer()
		pb.RegisterServiceServer(grpcServer, NewServer(stor))

		reflection.Register(grpcServer)

		go func(lis net.Listener) {
			err := grpcServer.Serve(lis)
			if err != nil {
				log.Printf("server stoped with error %v", err)
			}
		}(lis)

		// Запускаю клиента-------------------------------------------------------------------------
		client, err := initClient()
		require.NoError(t, err)

		// Отправляю counter метрику на сервер. Тест успешной передачи--------------------------------------
		goodCaunterMetric := pbModel.Metric{
			Id:    "good caunter metric",
			Mtype: "counter",
			Delta: delta(334643),
		}
		req := pbModel.AddMetricRequest{
			Metric: &goodCaunterMetric,
		}
		responce, err := client.AddMetric(ctx, &req)
		// проверяю ответ сервера, сервер должен вернуть теже метрику, что отправил клиент
		require.NoError(t, err)
		require.Nil(t, responce.Error)

		require.NotNil(t, responce)
		require.NotNil(t, responce.Metric)

		assert.Equal(t, goodCaunterMetric.Id, responce.Metric.Id)
		assert.Equal(t, goodCaunterMetric.Mtype, responce.Metric.Mtype)
		assert.Equal(t, goodCaunterMetric.Delta, responce.Metric.Delta)

		// Отправляю gauge метрику на сервер. Тест успешной передачи--------------------------------------
		goodGaugeMetric := pbModel.Metric{
			Id:    "good gauge metric",
			Mtype: "gauge",
			Value: value(13599.2352),
		}
		req = pbModel.AddMetricRequest{
			Metric: &goodGaugeMetric,
		}
		responce, err = client.AddMetric(ctx, &req)
		// проверяю ответ сервера, сервер должен вернуть теже метрику, что отправил клиент
		require.NoError(t, err)
		require.Nil(t, responce.Error)

		require.NotNil(t, responce)
		require.NotNil(t, responce.Metric)

		assert.Equal(t, goodGaugeMetric.Id, responce.Metric.Id)
		assert.Equal(t, goodGaugeMetric.Mtype, responce.Metric.Mtype)
		assert.Equal(t, goodGaugeMetric.Value, responce.Metric.Value)

		// Ошибка. Поле Delta не определено в метрике типа counter---------------------------------------
		badCounterDelta := pbModel.Metric{
			Id:    "bad counter delta",
			Mtype: "counter",
			Delta: nil,
		}
		req = pbModel.AddMetricRequest{
			Metric: &badCounterDelta,
		}
		_, err = client.AddMetric(ctx, &req)
		require.Error(t, err)
		e, ok := status.FromError(err)
		assert.Equal(t, true, ok)
		assert.Equal(t, codes.InvalidArgument, e.Code())

		// Ошибка. Поле Value не определено в метрике типа gauge---------------------------------------
		badGaugeValue := pbModel.Metric{
			Id:    "bad gauge value",
			Mtype: "gauge",
			Value: nil,
		}
		req = pbModel.AddMetricRequest{
			Metric: &badGaugeValue,
		}
		_, err = client.AddMetric(ctx, &req)
		require.Error(t, err)
		e, ok = status.FromError(err)
		assert.Equal(t, true, ok)
		assert.Equal(t, codes.InvalidArgument, e.Code())

		// Ошибка. Неправильный тип метрики---------------------------------------
		badMetricType := pbModel.Metric{
			Id:    "bad metric type",
			Mtype: "bad type",
		}
		req = pbModel.AddMetricRequest{
			Metric: &badMetricType,
		}
		_, err = client.AddMetric(ctx, &req)
		require.Error(t, err)
		e, ok = status.FromError(err)
		assert.Equal(t, true, ok)
		assert.Equal(t, codes.InvalidArgument, e.Code())

		// Ошибка. В запросе клиента не установлена метрика ---------------------------------------
		req = pbModel.AddMetricRequest{
			Metric: nil,
		}
		_, err = client.AddMetric(ctx, &req)
		require.Error(t, err)
		e, ok = status.FromError(err)
		assert.Equal(t, true, ok)
		assert.Equal(t, codes.InvalidArgument, e.Code())
	}
	{
		// Тест с непроинициализированным хранилищем метрик сервера

		// Функция для инициализации клиента
		initClient := func(netAddr string) (pb.ServiceClient, error) {
			conn, err := grpc.NewClient(netAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
			if err != nil {
				return nil, fmt.Errorf("failed to create a new client: %w", err)
			}
			client := pb.NewServiceClient(conn)
			return client, err
		}

		// Запускаю сервер----------------------------------------------------------------------------
		// Адрес запуска сервера
		serverPort, err := getFreePort()
		require.NoError(t, err)
		netAddr := fmt.Sprintf("localhost:%d", serverPort)

		lis, err := net.Listen("tcp", netAddr)
		require.NoError(t, err)

		grpcServer := grpc.NewServer()
		pb.RegisterServiceServer(grpcServer, NewServer(nil))

		reflection.Register(grpcServer)

		go func(lis net.Listener) {
			err := grpcServer.Serve(lis)
			if err != nil {
				log.Printf("server stoped with error %v", err)
			}
		}(lis)

		// Запускаю клиента-------------------------------------------------------------------------
		client, err := initClient(netAddr)
		require.NoError(t, err)

		// Ошибка. Хранилище сервера непроинициализировано ---------------------------------------
		ctx := context.Background()
		req := pbModel.AddMetricRequest{
			Metric: nil,
		}
		_, err = client.AddMetric(ctx, &req)
		require.Error(t, err)
		e, ok := status.FromError(err)
		assert.Equal(t, true, ok)
		assert.Equal(t, codes.Internal, e.Code())
	}
}
