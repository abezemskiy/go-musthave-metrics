package ipchecker

import (
	"fmt"
	"net"
	"strings"
)

// InTrustedSubNet - функия проверяет, находится ли переданный Ip в доверенной подсети используя строковое представление CIDR.
func InTrustedSubNet(trustedSubnet, realIP string) (bool, error) {
	_, ipNet, err := net.ParseCIDR(trustedSubnet)
	if err != nil {
		return false, fmt.Errorf("parse CIDR error: %v", err)
	}

	realIPWithoutPort, err := getClientIP(realIP)
	if err != nil {
		return false, fmt.Errorf("parse real client ip error %v", err)
	}

	ip := net.ParseIP(realIPWithoutPort)
	if ip == nil {
		return false, fmt.Errorf("parse real client ip error %v", err)
	}
	return ipNet.Contains(ip), nil
}

// getClientIP - Функция для извлечения IP без порта.
func getClientIP(address string) (string, error) {
	// пытаюсь разделить адрес на хост и порт
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		// Если адрес без порта, возвращаем его как есть
		if strings.Contains(err.Error(), "missing port") {
			return address, nil
		}
		return "", fmt.Errorf("failed to parse client address: %w", err)
	}

	// возвращаю только хост (IP-адрес без порта)
	return host, nil
}
