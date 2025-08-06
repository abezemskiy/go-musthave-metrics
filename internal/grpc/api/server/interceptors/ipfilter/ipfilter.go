package ipfilter

import (
	"context"

	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/status"

	httpIpfilter "github.com/AntonBezemskiy/go-musthave-metrics/internal/server/ipfilter"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/server/logger"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/tools/ipchecker"
)

// UnaryServerInterceptor - перехватчик проверки вхождения ip адреса клиента в доверенную сеть сервера. Ip адрес извлекается из заголовка контекста.
// Проверка осуществляется только в случае, если установлена переменная trustedSubnet.
func UnaryServerInterceptor(ctx context.Context, req any, _ *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (resp any, err error) {
	subnet := httpIpfilter.GetTrustedSubnet()

	// проверяю вхождение Ip адреса в доверенную сеть только в том случае, если установлена переменная trustedSubnet
	if subnet != "" {
		// Извлекаем информацию о клиенте
		p, ok := peer.FromContext(ctx)
		if !ok {
			return nil, status.Error(codes.PermissionDenied, "unable to get client IP")
		}

		// Получаем адрес клиента (включая порт, например "192.168.1.10:12345")
		realIP := p.Addr.String()
		logger.ServerLog.Debug("real ip of agent host is", zap.String("realip", realIP))

		// проверка вхождения ip в доверенную сеть
		intrusted, err := ipchecker.InTrustedSubNet(subnet, realIP)
		if err != nil {
			logger.ServerLog.Error("in trusted subNet check error", zap.String("error", error.Error(err)))
			return nil, status.Error(codes.Internal, "in trusted subNet check error")
		}
		if !intrusted {
			logger.ServerLog.Info("ip of agent is not in trusted sub net")
			return nil, status.Error(codes.PermissionDenied, "ip of agent is not in trusted sub net")
		}
	}

	// вызываю основной обработчик
	return handler(ctx, req)
}
