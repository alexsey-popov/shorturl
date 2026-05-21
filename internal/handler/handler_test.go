package handler

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/alexsey-popov/shorturl/internal/config"
)

func TestRouterFunc(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		path           string
		body           string
		contentType    string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "GET method without ID",
			method:         http.MethodGet,
			path:           "/",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Отсутствует идентификатор ссылки\n",
		},
		{
			name:           "GET method with invalid ID",
			method:         http.MethodGet,
			path:           "/invalid-id",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "url не найден\n",
		},
		{
			name:           "POST with valid URL",
			method:         http.MethodPost,
			path:           "/",
			body:           "https://example.com",
			contentType:    config.ContentType,
			expectedStatus: http.StatusCreated,
		},
		{
			name:           "POST with invalid content type",
			method:         http.MethodPost,
			path:           "/",
			body:           "https://example.com",
			contentType:    "application/json",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Некорректный тип содержимого запроса\n",
		},
		{
			name:           "POST to invalid path",
			method:         http.MethodPost,
			path:           "/invalid-path",
			body:           "https://example.com",
			contentType:    config.ContentType,
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Некорректный запрос\n",
		},
		{
			name:           "Unsupported HTTP method",
			method:         http.MethodPut,
			path:           "/",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Метод не поддерживается\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var body io.Reader
			if tt.body != "" {
				body = bytes.NewBuffer([]byte(tt.body))
			}

			req := httptest.NewRequest(tt.method, tt.path, body)
			if tt.contentType != "" {
				req.Header.Set("Content-Type", tt.contentType)
			}

			rr := httptest.NewRecorder()

			RouterFunc(rr, req)

			if rr.Code != tt.expectedStatus {
				t.Errorf("expected status %d, got %d", tt.expectedStatus, rr.Code)
			}

			if tt.expectedBody != "" && rr.Body.String() != tt.expectedBody {
				t.Errorf("expected body %q, got %q", tt.expectedBody, rr.Body.String())
			}
		})
	}
}
