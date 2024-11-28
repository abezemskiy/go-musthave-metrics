package ipchecker

import (
	"fmt"
	"net"
)

// InTrustedSubNet - функия проверяет, находится ли переданный Ip в доверенной подсети используя строковое представление CIDR.
func InTrustedSubNet(trustedSubnet, realIP string) (bool, error) {
	_, ipNet, err := net.ParseCIDR(trustedSubnet)
	if err != nil {
		return false, fmt.Errorf("parse CIDR error: %v", err)
	}

	ip := net.ParseIP(realIP)
	if ip == nil {
		return false, fmt.Errorf("wrong real ip from agent")
	}
	return ipNet.Contains(ip), nil
}
