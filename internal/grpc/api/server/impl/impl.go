package impl

import (
	"context"

	"go.uber.org/zap"

	pb "github.com/AntonBezemskiy/go-musthave-metrics/internal/grpc/protoc"
	pbModel "github.com/AntonBezemskiy/go-musthave-metrics/internal/grpc/protoc/model"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/repositories"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/server/logger"
)

type Server struct {
	pb.UnimplementedServiceServer
	storage repositories.IStorage
}

func NewServer(stor repositories.IStorage) *Server {
	return &Server{
		storage: stor,
	}
}

// AddMetric - gRPC метод для добавления метрики на сервер.
func (s *Server) AddMetric(ctx context.Context, req *pbModel.AddMetricRequest) (*pbModel.AddMetricResponce, error) {
	responce := &pbModel.AddMetricResponce{
		Error: nil,
	}

	// Проверка на nil для storage
	if s.storage == nil || req.Metric == nil {
		errStr := "storage not initialized or metric is nil"
		responce.Error = &errStr
		return responce, nil
	}
	metric := req.Metric

	switch metric.Mtype {
	case "gauge":
		if metric.Value == nil {
			logger.ServerLog.Error("Decode message error, value in gauge metric is nil")

			errStr := "decode message error, value in gauge metric is nil"
			responce.Error = &errStr
			return responce, nil
		}
		err := s.storage.AddGauge(ctx, metric.Id, *metric.Value)
		if err != nil {
			logger.ServerLog.Error("add gauge error", zap.String("error", error.Error(err)))
			errStr := "add gauge error"
			responce.Error = &errStr
			return responce, nil
		}
	case "counter":
		if metric.Delta == nil {
			logger.ServerLog.Error("Decode message error, delta in counter metric is nil")

			errStr := "decode message error, delta in counter metric is nil"
			responce.Error = &errStr
			return responce, nil
		}
		err := s.storage.AddCounter(ctx, metric.Id, *metric.Delta)
		if err != nil {
			logger.ServerLog.Error("add counter error", zap.String("error", error.Error(err)))

			errStr := "add counter error"
			responce.Error = &errStr
			return responce, nil
		}
	default:
		logger.ServerLog.Error("Invalid type of metric", zap.String("type", metric.Mtype))

		errStr := "invalid type of metric: " + metric.Mtype
		responce.Error = &errStr
		return responce, nil
	}
	logger.ServerLog.Debug("Successful decode metrcic from json")

	// возвращаю клиенту туже метрику, которую он отправил на сервер
	// в случае успешного добавления метрики на сервере
	responce.Metric = req.Metric
	return responce, nil
}
