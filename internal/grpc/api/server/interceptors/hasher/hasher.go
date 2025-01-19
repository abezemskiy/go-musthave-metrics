package hasher

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/proto"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/grpc/protoc/model"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/repositories"
	httpHasher "github.com/AntonBezemskiy/go-musthave-metrics/internal/server/hasher"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/server/logger"
)

// UnaryServerInterceptor - перехватчик для проверки подписи и подписи данных, если установлен ключ.
func UnaryServerInterceptor(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
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
		case *model.AddMetricRequest:
			ok, err := CheckHash(r, reqHashs[0], httpHasher.GetKey())
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

	// Подписываю ответ сервера
	switch r := resp.(type) {
	case *model.AddMetricResponce:
		// подписываю ответ сервера
		hash, err := CalkHash(r, httpHasher.GetKey())
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to hash response %v", err)
		}
		// Добавляю хэш в метаданные
		if err := grpc.SetHeader(ctx, metadata.Pairs("HashSHA256", hash)); err != nil {
			return nil, status.Errorf(codes.Internal, "failed to set response hash: %v", err)
		}
	default:
		return nil, status.Error(codes.Internal, "failed to hash response, unknown response type")
	}
	return
}

// CalkHash - вспомогательная функция для вычисления хэша из proto сообщения с помощью секретного ключа.
func CalkHash(resp proto.Message, key string) (string, error) {
	// Сериализация protoc сообщения в байты
	body, err := proto.Marshal(resp)
	if err != nil {
		return "", fmt.Errorf("failed to serialize response: %v", err)
	}
	// вычисление хэша
	res, err := repositories.CalkHash(body, key)
	if err != nil {
		return "", fmt.Errorf("failed to hash response, error: %v", err)
	}
	return res, nil
}

// CheckHash - вспомогательная функция для проверки переданного хэша и расчитанного из proto сообщения
// используя секретный ключ.
func CheckHash(resp proto.Message, wantHash, key string) (bool, error) {
	// Сериализация protoc сообщения в байты
	body, err := proto.Marshal(resp)
	if err != nil {
		return false, fmt.Errorf("failed to serialize request: %v", err)
	}

	ok, err := repositories.CheckHash(body, wantHash, key)
	if err != nil {
		return false, fmt.Errorf("checking hash error: %v", err)
	}
	return ok, nil
}
