package server

import (
	"crypto/rsa"
	"fmt"
	"net"

	"github.com/xhrobj/go-metrics-and-alerts/internal/encryption"
)

func loadPrivateKey(path string) (*rsa.PrivateKey, error) {
	if path == "" {
		return nil, nil
	}

	privateKey, err := encryption.LoadPrivateKey(path)
	if err != nil {
		return nil, fmt.Errorf("load private key: %w", err)
	}

	return privateKey, nil
}

func parseTrustedSubnet(value string) (*net.IPNet, error) {
	if value == "" {
		return nil, nil
	}

	_, subnet, err := net.ParseCIDR(value)
	if err != nil {
		return nil, fmt.Errorf("parse trusted subnet %q: %w", value, err)
	}

	return subnet, nil
}
