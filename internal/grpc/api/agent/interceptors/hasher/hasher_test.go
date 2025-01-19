package hasher

import (
	"context"
	"fmt"
	"log"
	"net"
	"testing"

	"go.uber.org/zap"

	httpAgentHasher "github.com/AntonBezemskiy/go-musthave-metrics/internal/agent/hasher"
	grpcServerHasher "github.com/AntonBezemskiy/go-musthave-metrics/internal/grpc/api/server/interceptors/hasher"
	pb "github.com/AntonBezemskiy/go-musthave-metrics/internal/grpc/protoc"
	pbModel "github.com/AntonBezemskiy/go-musthave-metrics/internal/grpc/protoc/model"
	httpHasher "github.com/AntonBezemskiy/go-musthave-metrics/internal/server/hasher"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/server/logger"

	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/reflection"
	"google.golang.org/grpc/status"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/grpc/api/server/impl"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/server/storage"
)

func serverInterceptor(secretKey, serverHash string) grpc.UnaryServerInterceptor {
	interceptor := func(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {

		// Если не сервере не задан секретный ключ для подписи данных, то эта операция не производится
		if k := httpHasher.GetKey(); k == "" {
			// вызываю основной обработчик без изменений и преобразований
			return handler(ctx, req)
		}

		// О необходимости такого поведения понял из тестов -------------------------------------------------------------------
		// Метаданные получаю из контекста
		md, ok := metadata.FromIncomingContext(ctx)
		if !ok {
			return nil, status.Error(codes.InvalidArgument, "hasher error: metadata not set to context")
		}

		if noneHash := md.Get("Hash"); len(noneHash) > 0 && noneHash[0] == "none" {
			// вызываю основной обработчик без изменений и преобразований
			return handler(ctx, req)
		}

		reqHashs := md.Get("HashSHA256")
		if len(reqHashs) == 0 {
			// вызываю основной обработчик без изменений и преобразований
			return handler(ctx, req)
		}

		// проверка подписи в случае непустого тела запроса ------------------------------------------------------------------
		if req != nil {
			// извлечение сообщения от сервера и проверка подписи
			switch r := req.(type) {
			case *pbModel.AddMetricRequest:
				ok, err := grpcServerHasher.CheckHash(r, reqHashs[0], secretKey)
				if err != nil {
					logger.ServerLog.Error("checking hash error", zap.String("error: ", error.Error(err)))
					return nil, status.Errorf(codes.Internal, "checking hash error: %v", err)
				}
				if !ok {
					logger.ServerLog.Error("hashs is not equal")
					return nil, status.Error(codes.InvalidArgument, "hashs is not equal")
				}
			default:
				return nil, status.Error(codes.InvalidArgument, "failed to serialize request, unknown request type")
			}
		}

		// Подписываю ответ сервера в случае, если задан ключ---------------------------------------------
		// вызываю основной обработчик
		resp, err = handler(ctx, req)
		if err != nil {
			return nil, err
		}

		// подписываю ответ сервера хэшем, котороый был передан в тестовый интерсептор
		// Добавляю хэш в метаданные
		if err := grpc.SetHeader(ctx, metadata.Pairs("HashSHA256", serverHash)); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to set response hash: %v", err)
		}
		return
	}
	return interceptor
}

func TestUnaryClientInterceptor(t *testing.T) {
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
		conn, err := grpc.NewClient(netAddr, grpc.WithTransportCredentials(insecure.NewCredentials()),
			grpc.WithUnaryInterceptor(UnaryClientInterceptor))
		if err != nil {
			return nil, nil, fmt.Errorf("failed to create a new client: %w", err)
		}
		client := pb.NewServiceClient(conn)
		return client, conn, err
	}

	// Создаю тестовый запрос
	goodCaunterMetric := pbModel.Metric{
		Id:    "good caunter metric",
		Mtype: "counter",
		Delta: delta(334643),
	}
	req := &pbModel.AddMetricRequest{
		Metric: &goodCaunterMetric,
	}

	{
		{
			// Устанавливаю секретный ключ
			key1 := "secret key for test1"
			httpHasher.SetKey(key1)
			httpAgentHasher.SetKey(key1)

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
				grpc.UnaryInterceptor(grpcServerHasher.UnaryServerInterceptor),
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

			// Запускаю клиента -------------------------------------------------------------------------
			client, conn, err := initClient(netAddr)
			defer conn.Close()
			require.NoError(t, err)

			// отправляю метрику
			_, err = client.AddMetric(context.Background(), req)
			require.NoError(t, err)
		}
		// Тест с установкой тестового перехватчика на сервер. Проверка случаев, когда секретный ключ отличается на сервере и клинете,
		// и когда сервер расчитывает неверный хэш ответа
		{
			// Устанавливаю секретный ключ
			key1 := "secret key for tests"
			httpHasher.SetKey(key1)
			httpAgentHasher.SetKey(key1)

			type request struct {
				req               *pbModel.AddMetricRequest
				serverInterceptor grpc.UnaryServerInterceptor
			}
			tests := []struct {
				name      string
				request   request
				wantError bool
			}{
				{
					name: "wrong server secret key",
					request: request{
						req:               req,
						serverInterceptor: serverInterceptor("wrong server secret key", ""),
					},
					wantError: true,
				},
				{
					name: "wrong server hash",
					request: request{
						req:               req,
						serverInterceptor: serverInterceptor(key1, "wrong hash calculated by server"),
					},
					wantError: true,
				},
			}

			for _, tt := range tests {
				t.Run(tt.name, func(t *testing.T) {

					// Адрес запуска сервера -----------------------------
					serverPort, err := getFreePort()
					require.NoError(t, err)
					netAddr := fmt.Sprintf("localhost:%d", serverPort)

					// инициализирую хранилище метрик
					stor := storage.NewDefaultMemStorage()

					// Запускаю сервер----------------------------------------------------------------------------
					lis, err := net.Listen("tcp", netAddr)
					require.NoError(t, err)

					// Устанавливаю тестовый серверный перехватчик
					grpcServer := grpc.NewServer(
						grpc.UnaryInterceptor(tt.request.serverInterceptor),
					)
					// Останавливаю сервер после окончания теста
					defer grpcServer.Stop()
					pb.RegisterServiceServer(grpcServer, impl.NewServer(stor))

					go func(lis net.Listener) {
						err := grpcServer.Serve(lis)
						if err != nil {
							log.Printf("server stoped with error %v", err)
						}
					}(lis)

					// Запускаю клиента -------------------------------------------------------------------------
					client, conn, err := initClient(netAddr)
					defer conn.Close()
					require.NoError(t, err)

					// отправляю метрику
					_, err = client.AddMetric(context.Background(), req)
					if tt.wantError {
						require.Error(t, err)
					} else {
						require.NoError(t, err)
					}

				})
			}
		}
	}
}
