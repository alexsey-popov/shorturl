package compact

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"go.uber.org/zap"
)

// TestHTTPMiddleware - Тестирование сжатия
func TestHTTPMiddleware(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read body: %v", err)
		}
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		w.Write(body)
	})

	mw := HTTPMiddleware(zap.S())(handler)

	t.Run("positive - gzip input", func(t *testing.T) {
		content := "hello world"
		var buf bytes.Buffer
		gz := gzip.NewWriter(&buf)
		gz.Write([]byte(content))
		gz.Close()

		req := httptest.NewRequest("POST", "/", &buf)
		req.Header.Set("Content-Type", "text/plain")
		req.Header.Set("Content-Encoding", "gzip")
		rec := httptest.NewRecorder()

		mw.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}
		if rec.Body.String() != content {
			t.Errorf("expected body %q, got %q", content, rec.Body.String())
		}
	})

	t.Run("positive - gzip output", func(t *testing.T) {
		content := "hello world"
		req := httptest.NewRequest("POST", "/", strings.NewReader(content))
		req.Header.Set("Content-Type", "text/plain")
		req.Header.Set("Accept-Encoding", "gzip")
		rec := httptest.NewRecorder()

		mw.ServeHTTP(rec, req)

		if rec.Code != http.StatusOK {
			t.Errorf("expected status 200, got %d", rec.Code)
		}

		gz, err := gzip.NewReader(rec.Body)
		if err != nil {
			t.Fatalf("failed to create gzip reader: %v", err)
		}
		defer gz.Close()

		decompressed, _ := io.ReadAll(gz)
		if string(decompressed) != content {
			t.Errorf("expected decompressed body %q, got %q", content, string(decompressed))
		}
	})
}
