package impl

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/encoding/gzip"
	"google.golang.org/grpc/status"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/agent/logger"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/agent/metrics/checker"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/grpc/api/agent/interceptors/hasher"
	pb "github.com/AntonBezemskiy/go-musthave-metrics/internal/grpc/protoc"
	pbModel "github.com/AntonBezemskiy/go-musthave-metrics/internal/grpc/protoc/model"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/repositories"
)

// Client - структура для реализации proto интерфейса клиента.
type Client struct {
	pb.ServiceClient
	conn *grpc.ClientConn
}

// InitClient - функция для инициализации gRPC клиента.
func InitClient(netAddr string) (*Client, error) {
	logger.AgentLog.Info("initialize new grpc client", zap.String("netAddr", netAddr))

	opts := []grpc.DialOption{
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithDefaultCallOptions(grpc.UseCompressor(gzip.Name)),
		grpc.WithUnaryInterceptor(hasher.UnaryClientInterceptor),
	}

	conn, err := grpc.NewClient(netAddr, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to create a new client: %w", err)
	}
	client := pb.NewServiceClient(conn)

	return &Client{
		conn:          conn,
		ServiceClient: client,
	}, err
}

// AddMetric - функция для последовательной отправки метрик на сервер из слайса метрик.
func AddMetric(ctx context.Context, cl *Client, metricsSlice []repositories.Metric) error {
	logger.AgentLog.Info("send metrics from slice to server")

	if metricsSlice == nil {
		return fmt.Errorf("metric slice is nil")
	}

	for _, metric := range metricsSlice {
		m := pbModel.Metric{
			Id:    metric.ID,
			Mtype: metric.MType,
			Delta: metric.Delta,
			Value: metric.Value,
		}
		req := pbModel.AddMetricRequest{
			Metric: &m,
		}
		// Вызов grpc метода. В этом месте можно установить необходимые перехватчики.
		resp, err := cl.AddMetric(ctx, &req)
		if err != nil {
			if e, ok := status.FromError(err); ok {
				return fmt.Errorf("error of add metric to server: %v", e.Message())
			}
			return fmt.Errorf("error of add metric to server, can't parse error: %v", err)
		}
		if resp.Error != nil && *resp.Error != "" {
			return fmt.Errorf("error of add metric to server, error from server response: %s", *resp.Error)
		}

		// Проверяю ответ сервера. Сравниваю метрику отправленную на сервер с метрикой, которую сервер вернул.
		if resp.Metric == nil {
			return fmt.Errorf("metric from server is nil in grpc.AddMetric")
		}
		if !checker.Equal(metric, repositories.Metric{
			ID:    resp.Metric.Id,
			MType: resp.Metric.Mtype,
			Delta: resp.Metric.Delta,
			Value: resp.Metric.Value,
		}) {
			return fmt.Errorf("metric from server is not equal request metric")
		}
	}
	return nil
}
