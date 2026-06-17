package logger

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

type (
	// ResponseDetails - Детализация ответа
	ResponseDetails struct {
		size   int
		status int
	}

	// DetailsResponseWriter - Дополненный ResponseWriter
	DetailsResponseWriter struct {
		http.ResponseWriter
		details ResponseDetails
	}
)

// Write - Метод оборачивающий ResponseWriter.Write
func (drw *DetailsResponseWriter) Write(data []byte) (int, error) {
	size, err := drw.ResponseWriter.Write(data)
	if err != nil {
		return 0, err
	}

	drw.details.size += size

	return size, err
}

// WriteHeader - Метод оборачивающий для ResponseWriter.Write
func (drw *DetailsResponseWriter) WriteHeader(statusCode int) {
	drw.ResponseWriter.WriteHeader(statusCode)

	drw.details.status = statusCode
}

// NewHTTPMiddleware Принимаем логгер, возвращаем функцию-замыкание, которая будет логировать запросы
func NewHTTPMiddleware(l *zap.SugaredLogger) func(http.Handler) http.Handler {

	return func(h http.Handler) http.Handler {
		logFn := func(rw http.ResponseWriter, r *http.Request) {
			url, method, timeStart := r.URL.String(), r.Method, time.Now()

			dwr := DetailsResponseWriter{
				ResponseWriter: rw,
				details: ResponseDetails{
					status: 0,
					size:   0,
				},
			}

			h.ServeHTTP(&dwr, r)

			status, size, duration := dwr.details.status, dwr.details.size, time.Since(timeStart)

			l.Infow(
				"request",
				"url", url,
				"method", method,
				"duration", duration,
				"status", status,
				"size", size,
			)
		}

		return http.HandlerFunc(logFn)
	}
}
