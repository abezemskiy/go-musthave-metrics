package hasher

import (
	"context"
	"fmt"
	"log"
	"net"
	"testing"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/grpc/api/server/impl"
	pb "github.com/AntonBezemskiy/go-musthave-metrics/internal/grpc/protoc"
	pbModel "github.com/AntonBezemskiy/go-musthave-metrics/internal/grpc/protoc/model"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/repositories"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/server/storage"

	httpHasher "github.com/AntonBezemskiy/go-musthave-metrics/internal/server/hasher"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"
)

func TestUnaryServerInterceptor(t *testing.T) {
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

	// Функция для инициализации клиента-------------------------------------------------------------
	initClient := func(netAddr string) (pb.ServiceClient, *grpc.ClientConn, error) {
		conn, err := grpc.NewClient(netAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create a new client: %w", err)
		}
		client := pb.NewServiceClient(conn)
		return client, conn, err
	}

	{
		// Устанавливаю секретный ключ
		key1 := "secret key for test1"
		httpHasher.SetKey(key1)

		// Адрес запуска сервера -----------------------------
		serverPort, err := getFreePort()
		require.NoError(t, err)
		netAddr := fmt.Sprintf("localhost:%d", serverPort)

		// инициализирую хранилище метрик
		stor := storage.NewDefaultMemStorage()

		// Запускаю сервер----------------------------------------------------------------------------
		lis, err := net.Listen("tcp", netAddr)
		require.NoError(t, err)

		// Устанавливаю тестируемый перехватчик
		grpcServer := grpc.NewServer(
			grpc.UnaryInterceptor(UnaryServerInterceptor),
		)
		// Останавливаю сервер после окончания теста
		defer grpcServer.Stop()
		pb.RegisterServiceServer(grpcServer, impl.NewServer(stor))

		reflection.Register(grpcServer)

		go func(lis net.Listener) {
			err := grpcServer.Serve(lis)
			if err != nil {
				log.Printf("server stoped with error %v", err)
			}
		}(lis)

		// Создаю тестовый запрос
		goodCaunterMetric := pbModel.Metric{
			Id:    "good caunter metric",
			Mtype: "counter",
			Delta: delta(334643),
		}
		req := &pbModel.AddMetricRequest{
			Metric: &goodCaunterMetric,
		}

		// вычисляю хэш тестового запроса
		hash, err := CalkHash(req, key1)
		require.NoError(t, err)

		{
			type want struct {
				existHash  bool
				hash       string
				err        bool
				statusCode codes.Code
			}
			tests := []struct {
				name   string
				req    *pbModel.AddMetricRequest
				key    string
				want   want
				header []string
			}{
				{
					name: "successful hashing",
					req:  req,
					key:  key1,
					want: want{
						existHash: true,
						hash:      hash,
						err:       false,
					},
					header: []string{"Hash", "exist", "HashSHA256", hash},
				},
				{
					name: "header hash is none",
					req:  req,
					key:  key1,
					want: want{
						existHash: false,
						err:       false,
					},
					header: []string{"Hash", "none", "HashSHA256", hash},
				},
				{
					name: "header HashSHA256 not set",
					req:  req,
					key:  key1,
					want: want{
						existHash: false,
						err:       false,
					},
					header: []string{"Hash", "exist"},
				},
				{
					name: "hash is invalid",
					req:  req,
					key:  key1,
					want: want{
						existHash:  false,
						hash:       hash,
						err:        true,
						statusCode: codes.Internal,
					},
					header: []string{"Hash", "exist", "HashSHA256", "wrong hash"},
				},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {
					// Запускаю клиента -------------------------------------------------------------------------
					client, conn, err := initClient(netAddr)
					defer conn.Close()
					require.NoError(t, err)

					// Создаю контекст с метаданными
					ctx := metadata.AppendToOutgoingContext(context.Background(),
						tt.header...,
					)

					// Контейнер для метаданных отправленных сервером
					header := metadata.MD{}

					// отправляю метрику
					responce, err := client.AddMetric(ctx, req, grpc.Header(&header))
					if tt.want.err {
						require.Error(t, err)
						e, ok := status.FromError(err)
						assert.Equal(t, true, ok)
						assert.Equal(t, tt.want.statusCode, e.Code())
					} else {
						require.NoError(t, err)
					}

					// Чтение метаданных
					responseHash := header.Get("HashSHA256")

					if tt.want.existHash {
						// Сервер должен был проверить подписать и подписать отправляемые в ответ данные
						// Проверяем метаданные ответа
						require.NotEmpty(t, responseHash)
						require.Equal(t, tt.want.hash, responseHash[0])
					}

					if !tt.want.err {
						// проверяю ответ сервера, сервер должен вернуть теже метрику, что отправил клиент
						require.NotNil(t, responce)
						require.NotNil(t, responce.Metric)

						assert.Equal(t, goodCaunterMetric.Id, responce.Metric.Id)
						assert.Equal(t, goodCaunterMetric.Mtype, responce.Metric.Mtype)
						assert.Equal(t, goodCaunterMetric.Delta, responce.Metric.Delta)
					}
				})
			}
		}
	}
	// Не устанавливаю секретный ключ и запрос выполняется без вмешательства перехватчика
	{
		// Не устанавливаю секретный ключ
		key1 := ""
		httpHasher.SetKey(key1)

		// Адрес запуска сервера -----------------------------
		serverPort, err := getFreePort()
		require.NoError(t, err)
		netAddr := fmt.Sprintf("localhost:%d", serverPort)

		// инициализирую хранилище метрик
		stor := storage.NewDefaultMemStorage()

		// Запускаю сервер----------------------------------------------------------------------------
		lis, err := net.Listen("tcp", netAddr)
		require.NoError(t, err)

		// Устанавливаю тестируемый перехватчик
		grpcServer := grpc.NewServer(
			grpc.UnaryInterceptor(UnaryServerInterceptor),
		)
		// Останавливаю сервер после окончания теста
		defer grpcServer.Stop()
		pb.RegisterServiceServer(grpcServer, impl.NewServer(stor))

		reflection.Register(grpcServer)

		go func(lis net.Listener) {
			err := grpcServer.Serve(lis)
			if err != nil {
				log.Printf("server stoped with error %v", err)
			}
		}(lis)

		// Создаю тестовый запрос
		goodCaunterMetric := pbModel.Metric{
			Id:    "good caunter metric",
			Mtype: "counter",
			Delta: delta(334643),
		}
		req := &pbModel.AddMetricRequest{
			Metric: &goodCaunterMetric,
		}

		// Запускаю клиента -------------------------------------------------------------------------
		client, conn, err := initClient(netAddr)
		defer conn.Close()
		require.NoError(t, err)

		// отправляю метрику
		responce, err := client.AddMetric(context.Background(), req)
		require.NoError(t, err)

		// проверяю ответ сервера, сервер должен вернуть теже метрику, что отправил клиент
		require.NotNil(t, responce)
		require.NotNil(t, responce.Metric)

		assert.Equal(t, goodCaunterMetric.Id, responce.Metric.Id)
		assert.Equal(t, goodCaunterMetric.Mtype, responce.Metric.Mtype)
		assert.Equal(t, goodCaunterMetric.Delta, responce.Metric.Delta)
	}
}

func TestCalkHash(t *testing.T) {
	// Вспомогательные функции --------------------------
	delta := func(d int64) *int64 {
		return &d
	}
	{
		key := "secret key"
		// Создаю тестовый запрос
		goodCaunterMetric := pbModel.Metric{
			Id:    "good caunter metric",
			Mtype: "counter",
			Delta: delta(334643),
		}
		req := &pbModel.AddMetricRequest{
			Metric: &goodCaunterMetric,
		}

		// Сериализация ответа сервера в байты
		body, err := proto.Marshal(req)
		require.NoError(t, err)
		// вычисление хэша
		wantHash, err := repositories.CalkHash(body, key)
		require.NoError(t, err)

		getHash, err := CalkHash(req, key)
		require.NoError(t, err)

		// Проверка
		assert.Equal(t, wantHash, getHash)
	}
}

func TestCheckHash(t *testing.T) {
	// Вспомогательные функции --------------------------
	delta := func(d int64) *int64 {
		return &d
	}
	{
		key := "secret key"
		// Создаю тестовый запрос
		goodCaunterMetric := pbModel.Metric{
			Id:    "good caunter metric",
			Mtype: "counter",
			Delta: delta(334643),
		}
		req := &pbModel.AddMetricRequest{
			Metric: &goodCaunterMetric,
		}

		// Сериализация ответа сервера в байты
		body, err := proto.Marshal(req)
		require.NoError(t, err)
		// вычисление хэша
		wantHash, err := repositories.CalkHash(body, key)
		require.NoError(t, err)

		// Успешная проверка
		ok, err := CheckHash(req, wantHash, key)
		require.NoError(t, err)
		assert.Equal(t, true, ok)

		// Невалидный хэш
		_, err = CheckHash(req, "wrong hash", key)
		require.Error(t, err)
	}
}
