package logger

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"go.uber.org/zap/zaptest/observer"
)

func TestDetailsResponseWriter(t *testing.T) {
	rec := httptest.NewRecorder()
	drw := &DetailsResponseWriter{
		ResponseWriter: rec,
		details: ResponseDetails{
			size:   0,
			status: 0,
		},
	}

	drw.WriteHeader(http.StatusCreated)
	assert.Equal(t, http.StatusCreated, drw.details.status)

	n, err := drw.Write([]byte("test data"))
	assert.NoError(t, err)
	assert.Equal(t, 9, n)
	assert.Equal(t, 9, drw.details.size)
}

func TestNewHTTPMiddleware(t *testing.T) {
	core, observedLogs := observer.New(zap.InfoLevel)
	logger := zap.New(core).Sugar()

	middleware := NewHTTPMiddleware(logger)

	handler := middleware(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("hello"))
	}))

	req := httptest.NewRequest(http.MethodGet, "/test-url", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Equal(t, "hello", rec.Body.String())

	logs := observedLogs.All()
	assert.Len(t, logs, 1)

	entry := logs[0]
	assert.Equal(t, zap.InfoLevel, entry.Level)
	assert.Equal(t, "request", entry.Message)

	fieldMap := make(map[string]interface{})
	for _, f := range entry.Context {
		switch f.Key {
		case "url", "method":
			fieldMap[f.Key] = f.String
		case "status", "size":
			fieldMap[f.Key] = int(f.Integer)
		case "duration":
			fieldMap[f.Key] = f.Interface
		}
	}

	assert.Equal(t, "/test-url", fieldMap["url"])
	assert.Equal(t, "GET", fieldMap["method"])
	assert.Equal(t, http.StatusOK, fieldMap["status"])
	assert.Equal(t, 5, fieldMap["size"])
}
