package ipfilter

import (
	"context"
	"fmt"
	"log"
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/grpc/api/server/impl"
	pb "github.com/AntonBezemskiy/go-musthave-metrics/internal/grpc/protoc"
	pbModel "github.com/AntonBezemskiy/go-musthave-metrics/internal/grpc/protoc/model"
	httpIpfilter "github.com/AntonBezemskiy/go-musthave-metrics/internal/server/ipfilter"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/server/storage"
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

	// Создаю тестовый запрос
	goodCaunterMetric := pbModel.Metric{
		Id:    "good caunter metric",
		Mtype: "counter",
		Delta: delta(334643),
	}
	req := &pbModel.AddMetricRequest{
		Metric: &goodCaunterMetric,
	}

	tests := []struct {
		name    string
		subNet  string
		realIP  string
		wantErr bool
	}{
		{
			name:    "in trusted",
			subNet:  "127.0.0.0/24",
			wantErr: false,
		},
		{
			name:    "not in trusted",
			subNet:  "192.168.1.0/24",
			wantErr: true,
		},
		{
			name:    "wrong subNet",
			subNet:  "wrong.sub.net",
			wantErr: true,
		},
		{
			name:    "empty subNet",
			subNet:  "",
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			httpIpfilter.SetTrustedSubnet(tt.subNet)

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
			responce, err := client.AddMetric(context.Background(), req)
			if tt.wantErr {
				require.Error(t, err)
			} else {
				require.NoError(t, err)
				assert.Equal(t, goodCaunterMetric.Id, responce.Metric.Id)
				assert.Equal(t, goodCaunterMetric.Mtype, responce.Metric.Mtype)
				assert.Equal(t, goodCaunterMetric.Delta, responce.Metric.Delta)
			}
		})
	}
}
