package subnet

import (
	"net"
	"net/http"

	"go.uber.org/zap"
)

// HTTPMiddleware - посредник для проверки нахождения пользователя в доверенной подсети
func HTTPMiddleware(sugar *zap.SugaredLogger, cidr string) func(http.Handler) http.Handler {
	return func(handler http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ipStr := r.Header.Get("X-Real-IP")

			// Если через конфиг не передали cidr или у пользователя отсутствует X-Real-IP - выводим 403
			if cidr == "" || ipStr == "" {
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}

			// Парсим IP и сеть
			ip := net.ParseIP(ipStr)

			_, ipNet, err := net.ParseCIDR(cidr)
			if err != nil {
				sugar.Errorf("ошибка парсинга подсети: %v", err)
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
				return
			}

			// Если IP не входит в подсеть - выводим ошибку
			if !ipNet.Contains(ip) {
				http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
			}

			handler.ServeHTTP(w, r)
		})
	}

}
