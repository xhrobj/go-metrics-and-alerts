// Package transport содержит общие вспомогательные функции транспортов Агента.
package transport

import (
	"errors"
	"fmt"
	"net"
)

// LocalIP возвращает первый нелокальный IPv4-адрес Агента.
func LocalIP() (string, error) {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "", fmt.Errorf("get interface addresses: %w", err)
	}

	for _, addr := range addrs {
		ipNet, ok := addr.(*net.IPNet)
		if !ok || ipNet.IP.IsLoopback() {
			continue
		}

		ip := ipNet.IP.To4()
		if ip != nil {
			return ip.String(), nil
		}
	}

	return "", errors.New("local IPv4 address not found")
}
