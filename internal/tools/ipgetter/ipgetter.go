package ipgetter

import (
	"fmt"
	"net"

	"go.uber.org/zap"

	"github.com/AntonBezemskiy/go-musthave-metrics/internal/server/logger"
)

// Get - функция, которая находит первый попавшийся Ipv4 адрес и возвращает его строковое представление.
func Get() (string, error) {
	// Получаею список всех сетевых интерфейсов
	interfaces, err := net.Interfaces()
	if err != nil {
		return "", fmt.Errorf("interfaces getting error: %w", err)
	}

	for _, iface := range interfaces {
		// Получаю адрес каждого интерфейса
		addrs, err := iface.Addrs()
		if err != nil {
			logger.ServerLog.Error("getting address error", zap.String("error", error.Error(err)))
			continue
		}
		// Перебираю адреса и вывожу их
		for _, addr := range addrs {
			// проверяю, является ли это IP-адресом IPv4
			if ipnet, ok := addr.(*net.IPNet); ok && ipnet.IP.To4() != nil {
				return ipnet.IP.String(), nil
			}
		}
	}

	return "", fmt.Errorf("ip address of host is empty")
}
