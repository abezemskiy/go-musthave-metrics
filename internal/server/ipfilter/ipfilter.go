package ipfilter

import (
	"net/http"

	"go.uber.org/zap"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/server/logger"
	"github.com/AntonBezemskiy/go-musthave-metrics/internal/tools/ipchecker"
)

// trustedSubnet - переменная, которая хранит строковое представление бесклассовой адресации (CIDR).
var trustedSubnet string

// SetTrustedSubnet - функция для установки строкового представления бесклассовой адресации (CIDR).
func SetTrustedSubnet(t string) {
	trustedSubnet = t
}

// getTrustedSubnet - функция для получения строкового представления бесклассовой адресации (CIDR).
func GetTrustedSubnet() string {
	return trustedSubnet
}

// Middleware - мидлварь для проверки вхождения ip адреса в доверенную сеть. Ip адрес извлекается из заголовка X-Real-IP.
// Проверка осуществляется только в случае, если установлена переменная trustedSubnet.
func Middleware(h http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		subnet := GetTrustedSubnet()

		// проверяю вхождение Ip адреса в доверенную сеть только в том случае, если установлена переменная trustedSubnet
		if subnet != "" {
			realIP := r.Header.Get("X-Real-IP")

			logger.ServerLog.Debug("real ip of agent host is", zap.String("nrealip", realIP))

			if realIP == "" {
				w.WriteHeader(http.StatusForbidden)
				return
			}
			// проверка вхождения ip в доверенную сеть
			intrusted, err := ipchecker.InTrustedSubNet(subnet, realIP)
			if err != nil {
				logger.ServerLog.Error("in trusted subNet check error", zap.String("error", error.Error(err)))
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			if !intrusted {
				logger.ServerLog.Info("ip of agent not in trusted sub net")
				w.WriteHeader(http.StatusForbidden)
				return
			}
		}

		// передаём управление хендлеру
		h.ServeHTTP(w, r)
	}
}
