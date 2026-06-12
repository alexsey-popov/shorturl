package logger

import (
	"net/http"
	"time"

	"go.uber.org/zap"
)

// Sugar - логгер
var Sugar *zap.SugaredLogger

func SetLogger(logger *zap.SugaredLogger) {
	Sugar = logger
}

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

// HTTPMiddleware - Посредник для логирования запросов
func HTTPMiddleware(h http.Handler) http.Handler {
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

		Sugar.Infoln(
			"url", url,
			"method", method,
			"duration", duration,
			"status", status,
			"size", size)
	}

	return http.HandlerFunc(logFn)
}
