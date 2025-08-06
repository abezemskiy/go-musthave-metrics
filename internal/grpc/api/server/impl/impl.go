package impl

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	pb "github.com/AntonBezemskiy/go-musthave-metrics/internal/grpc/protoc"
	pbModel "github.com/AntonBezemskiy/go-musthave-metrics/internal/grpc/protoc/model"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/repositories"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/server/logger"
)

// Server - структура для реализации proto интерфейса сервера.
type Server struct {
	pb.UnimplementedServiceServer
	storage repositories.IStorage
}

// NewServer - фабричная функция структуры Server.
func NewServer(stor repositories.IStorage) *Server {
	return &Server{
		storage: stor,
	}
}

// AddMetric - gRPC метод для добавления метрики на сервер.
func (s *Server) AddMetric(ctx context.Context, req *pbModel.AddMetricRequest) (*pbModel.AddMetricResponce, error) {
	responce := &pbModel.AddMetricResponce{}

	// Проверка на nil для хранилище сервера
	if s.storage == nil {
		return nil, status.Error(codes.Internal, "storage not initialized")
	}
	if req.Metric == nil {
		return nil, status.Error(codes.InvalidArgument, "metric in request is nil")
	}
	metric := req.Metric

	// загрузка метрики в хранилище сервера
	switch metric.Mtype {
	case "gauge":
		if metric.Value == nil {
			logger.ServerGRPCLog.Error("Decode message error, value in gauge metric is nil")
			return nil, status.Error(codes.InvalidArgument, "decode message error, value in gauge metric is nil")
		}
		err := s.storage.AddGauge(ctx, metric.Id, *metric.Value)
		if err != nil {
			logger.ServerGRPCLog.Error("add gauge error", zap.String("error", error.Error(err)))
			return nil, status.Error(codes.Internal, "add gauge error")
		}
	case "counter":
		if metric.Delta == nil {
			logger.ServerGRPCLog.Error("Decode message error, delta in counter metric is nil")
			return nil, status.Error(codes.InvalidArgument, "decode message error, delta in counter metric is nil")
		}
		err := s.storage.AddCounter(ctx, metric.Id, *metric.Delta)
		if err != nil {
			logger.ServerGRPCLog.Error("add counter error", zap.String("error", error.Error(err)))
			return nil, status.Error(codes.Internal, "add counter error")
		}
	default:
		logger.ServerGRPCLog.Error("Invalid type of metric", zap.String("type", metric.Mtype))
		return nil, status.Errorf(codes.InvalidArgument, "invalid type of metric, type %s", metric.Mtype)
	}
	logger.ServerGRPCLog.Debug("Successful decode metrcic from json")

	// возвращаю клиенту туже метрику, которую он отправил на сервер
	// в случае успешного добавления метрики на сервере
	responce.Metric = req.Metric
	return responce, nil
}
