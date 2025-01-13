package hasher

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"

	httpHasher "github.com/AntonBezemskiy/go-musthave-metrics/internal/agent/hasher"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/agent/logger"
	serverHasher "github.com/AntonBezemskiy/go-musthave-metrics/internal/grpc/api/server/interceptors/hasher"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/grpc/protoc/model"

	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
)

// UnaryClientInterceptor - перехватчик клиента для подписи данных и проверки подписи ответа сервера, если установлен ключ.
func UnaryClientInterceptor(ctx context.Context, method string, req, reply interface{},
	cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {

	// секретный ключ для подписи данных
	secretKey := httpHasher.GetKey()
	// если ключ не установлен, подписыать данные и проверять подпись ответа сервера не нужно
	if secretKey == "" {
		return invoker(ctx, method, req, reply, cc, opts...)
	}

	// Подпись запроса клиента------------------------------------------------------
	switch r := req.(type) {
	case *model.AddMetricRequest:
		// подписываю запрос
		outgoingCtx, err := SetHash(ctx, r, secretKey)
		if err != nil {
			logger.AgentLog.Error("failed to hash request", zap.String("method: ", method), zap.String("error: ", error.Error(err)))
			return fmt.Errorf("failed to hash request %v", err)
		}
		ctx = outgoingCtx
	default:
		return fmt.Errorf("failed to hash request, unknown request type")
	}

	// вызываем RPC-метод--------------------------------
	// для получения метаданных от сервера
	var header metadata.MD
	err := invoker(ctx, method, req, reply, cc, append(opts, grpc.Header(&header))...)
	if err != nil {
		logger.AgentLog.Error("invoke grpc method error", zap.String("method: ", method), zap.String("error: ", error.Error(err)))
		return err
	}

	// Проверка подписи ответа сервера----------------------------------------------------
	reqHashes := header.Get("HashSHA256")
	if len(reqHashes) == 0 {
		// хэш не установлен
		return fmt.Errorf("hash is not set in server response")
	}

	// извлечение ответа сервера и проверка подписи
	switch r := reply.(type) {
	case *model.AddMetricResponce:
		ok, err := serverHasher.CheckHash(r, reqHashes[0], httpHasher.GetKey())
		if err != nil {
			logger.AgentLog.Error("checking hash error", zap.String("method: ", method), zap.String("error: ", error.Error(err)))
			return fmt.Errorf("checking hash error: %v", err)
		}
		if !ok {
			logger.AgentLog.Error("hashs is not equal", zap.String("method: ", method))
			return fmt.Errorf("hashs is not equal")
		}
	default:
		return fmt.Errorf("failed to serialize response, unknown request type")
	}
	return nil
}

// SetHash - вспомогательная функция подписи запроса у установки хэша в контекст.
func SetHash(ctx context.Context, req proto.Message, secretKey string) (context.Context, error) {
	// подписываю запрос
	hash, err := serverHasher.CalkHash(req, secretKey)
	if err != nil {
		return nil, fmt.Errorf("failed to hash request %v", err)
	}
	// Добавляю хэш в метаданные
	md := metadata.New(map[string]string{"HashSHA256": hash})
	outgoingCtx := metadata.NewOutgoingContext(ctx, md)
	return outgoingCtx, nil
}
