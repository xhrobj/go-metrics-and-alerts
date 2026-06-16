package middleware

import (
	"net"
	"net/http"
)

const realIPHeader = "X-Real-IP"

// WithTrustedSubnet разрешает запросы только от IP-адресов,
// входящих в доверенную подсеть.
// Если подсеть не задана, проверка отключена.
func WithTrustedSubnet(subnet *net.IPNet) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if subnet == nil {
				next.ServeHTTP(w, r)
				return
			}

			ip := net.ParseIP(r.Header.Get(realIPHeader))
			if ip == nil || !subnet.Contains(ip) {
				w.WriteHeader(http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
