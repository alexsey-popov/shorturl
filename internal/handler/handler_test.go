package handler

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alexsey-popov/shorturl/internal/config"
	"github.com/go-chi/chi/v5"
)

func TestHandlePost(t *testing.T) {
	type want struct {
		contentType string
		statusCode  int
	}
	tests := []struct {
		name        string
		contentType string
		body        string
		want        want
	}{
		{
			name:        "positive test #1",
			contentType: "text/plain",
			body:        "https://practicum.yandex.ru/",
			want: want{
				contentType: "text/plain",
				statusCode:  201,
			},
		},
		{
			name:        "negative test #1 - wrong content type",
			contentType: "application/json",
			body:        "https://practicum.yandex.ru/",
			want: want{
				contentType: "text/plain; charset=utf-8",
				statusCode:  400,
			},
		},
		{
			name:        "negative test #2 - invalid url",
			contentType: "text/plain",
			body:        "invalid-url",
			want: want{
				contentType: "text/plain; charset=utf-8",
				statusCode:  400,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(tt.body))
			request.Header.Set("Content-Type", tt.contentType)
			w := httptest.NewRecorder()
			HandlePost(w, request)

			res := w.Result()
			defer res.Body.Close()

			if res.StatusCode != tt.want.statusCode {
				t.Errorf("HandlePost() status code = %v, want %v", res.StatusCode, tt.want.statusCode)
			}

			if tt.want.contentType != "" {
				if got := res.Header.Get("Content-Type"); got != tt.want.contentType {
					t.Errorf("HandlePost() content type = %v, want %v", got, tt.want.contentType)
				}
			}

			if res.StatusCode == http.StatusCreated {
				resBody, err := io.ReadAll(res.Body)
				if err != nil {
					t.Fatal(err)
				}
				if !strings.Contains(string(resBody), config.Protocol+config.ServerAddr) {
					t.Errorf("HandlePost() body = %v, want it to contain %v", string(resBody), config.Protocol+config.ServerAddr)
				}
			}
		})
	}
}

func TestHandleGet(t *testing.T) {
	// Pre-seed some data
	originalURL := "https://practicum.yandex.ru/"
	shortURL, err := links.Add(originalURL)
	if err != nil {
		t.Fatal(err)
	}

	// extract id from shortURL
	parts := strings.Split(shortURL, "/")
	id := parts[len(parts)-1]

	type want struct {
		statusCode int
		location   string
	}
	tests := []struct {
		name string
		id   string
		want want
	}{
		{
			name: "positive test #1",
			id:   id,
			want: want{
				statusCode: 307,
				location:   originalURL,
			},
		},
		{
			name: "negative test #1 - non-existent id",
			id:   "nonexistent",
			want: want{
				statusCode: 400,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			request := httptest.NewRequest(http.MethodGet, "/"+tt.id, nil)

			// Set chi context for URL parameter
			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", tt.id)
			request = request.WithContext(context.WithValue(request.Context(), chi.RouteCtxKey, rctx))

			w := httptest.NewRecorder()
			HandleGet(w, request)

			res := w.Result()
			defer res.Body.Close()

			if res.StatusCode != tt.want.statusCode {
				t.Errorf("HandleGet() status code = %v, want %v", res.StatusCode, tt.want.statusCode)
			}

			if tt.want.location != "" {
				if got := res.Header.Get("Location"); got != tt.want.location {
					t.Errorf("HandleGet() location = %v, want %v", got, tt.want.location)
				}
			}
		})
	}
}
