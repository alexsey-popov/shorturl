package handler

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/alexsey-popov/shorturl/internal/config"
	"github.com/alexsey-popov/shorturl/internal/model"
	"github.com/alexsey-popov/shorturl/pkg/contentType"
	"github.com/go-chi/chi/v5"
)

func TestHandlePost(t *testing.T) {
	type want struct {
		statusCode  int
		contentType string
	}

	tests := []struct {
		name        string
		target      string
		contentType string
		body        string
		want        want
	}{
		{
			name:        "positive #1",
			target:      "/",
			contentType: contentType.Plain,
			body:        "https://example.com/positive-1",
			want: want{
				statusCode:  http.StatusCreated,
				contentType: contentType.Plain,
			},
		},
		{
			name:        "negative #1 - incorrect url",
			target:      "/",
			contentType: contentType.Plain,
			body:        "incorrect url",
			want: want{
				statusCode:  http.StatusBadRequest,
				contentType: contentType.Plain,
			},
		},
		{
			name:        "negative #2 - empty body",
			target:      "/",
			contentType: contentType.Plain,
			body:        "",
			want: want{
				statusCode:  http.StatusBadRequest,
				contentType: contentType.Plain,
			},
		},
		{
			name:        "negative #3 - incorrect Content-type",
			target:      "/",
			contentType: "application/json",
			body:        "incorrect url",
			want: want{
				statusCode:  http.StatusBadRequest,
				contentType: contentType.Plain,
			},
		},
		{
			name:        "negative #4 - empty Content-type",
			target:      "/",
			contentType: "",
			body:        "incorrect url",
			want: want{
				statusCode:  http.StatusBadRequest,
				contentType: contentType.Plain,
			},
		},
	}

	UseMemoryRepository(config.Server.NetAddress)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, tt.target, strings.NewReader(tt.body))
			r.Header.Set("Content-Type", tt.contentType)
			w := httptest.NewRecorder()

			HandlePost(w, r)

			res := w.Result()
			defer res.Body.Close()

			if statusCode := res.StatusCode; statusCode != tt.want.statusCode {
				t.Errorf("StatusCode get %v, want %v", statusCode, tt.want.statusCode)
				if body, err := io.ReadAll(res.Body); err == nil {
					t.Log(string(body))
				}
			}

			if contentType := res.Header.Get("Content-Type"); strings.Contains(contentType, tt.want.contentType) == false {
				t.Errorf("Content-Type get %s, want %s", contentType, tt.want.contentType)
			}
		})
	}
}

// TestHandlePostJson Тесты для /api/shorten
func TestHandlePostJson(t *testing.T) {
	type want struct {
		statusCode  int
		contentType string
	}

	tests := []struct {
		name        string
		target      string
		contentType string
		body        string
		want        want
	}{
		{
			name:        "positive #1",
			target:      "/api/shorten",
			contentType: contentType.JSON,
			body:        `{"url": "https://example.com/positive-1"}`,
			want: want{
				statusCode:  http.StatusCreated,
				contentType: contentType.JSON,
			},
		},
		{
			name:        "negative #1 - incorrect url",
			target:      "/api/shorten",
			contentType: contentType.JSON,
			body:        `{"url": "incorrect-url"}`,
			want: want{
				statusCode:  http.StatusBadRequest,
				contentType: contentType.Plain,
			},
		},
		{
			name:        "negative #2 - empty body",
			target:      "/api/shorten",
			contentType: contentType.JSON,
			body:        "",
			want: want{
				statusCode:  http.StatusBadRequest,
				contentType: contentType.Plain,
			},
		},
		{
			name:        "negative #3 - incorrect Content-type",
			target:      "/api/shorten",
			contentType: contentType.Plain,
			body:        "incorrect url",
			want: want{
				statusCode:  http.StatusBadRequest,
				contentType: contentType.Plain,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, tt.target, strings.NewReader(tt.body))
			r.Header.Set("Content-Type", tt.contentType)
			w := httptest.NewRecorder()

			HandlePostJSON(w, r)

			res := w.Result()
			defer res.Body.Close()

			if statusCode := res.StatusCode; statusCode != tt.want.statusCode {
				t.Errorf("StatusCode get %v, want %v", statusCode, tt.want.statusCode)
				if body, err := io.ReadAll(res.Body); err == nil {
					t.Log(string(body))
				}
			}

			if contentType := res.Header.Get("Content-Type"); strings.Contains(contentType, tt.want.contentType) == false {
				t.Errorf("Content-Type get %s, want %s", contentType, tt.want.contentType)
			}
		})
	}
}

func TestHandleGet(t *testing.T) {
	// Добавляем в shortener заранее известную пару prefix => originalURL
	prefix, originalURL := "positive1", "https://example.com/positive1"
	if err := shortener.Rep.Set(model.New(prefix, originalURL)); err != nil {
		t.Fatal(err)
	}

	type want struct {
		statusCode int
		location   string
	}

	tests := []struct {
		name   string
		target string
		want   want
	}{
		{
			name:   "positive #1",
			target: "/" + prefix,
			want: want{
				statusCode: http.StatusTemporaryRedirect,
				location:   originalURL,
			},
		},
		{
			name:   "negative #1 - invalid target",
			target: "/invalid-target",
			want: want{
				statusCode: http.StatusBadRequest,
				location:   "",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodGet, tt.target, http.NoBody)

			rctx := chi.NewRouteContext()
			rctx.URLParams.Add("id", strings.TrimLeft(tt.target, "/"))
			r = r.WithContext(context.WithValue(r.Context(), chi.RouteCtxKey, rctx))

			w := httptest.NewRecorder()

			HandleGet(w, r)

			res := w.Result()
			defer res.Body.Close()

			if statusCode := res.StatusCode; statusCode != tt.want.statusCode {
				t.Errorf("StatusCode get %v, want %v", statusCode, tt.want.statusCode)
				if body, err := io.ReadAll(res.Body); err == nil {
					t.Log(string(body))
				}
			}

			if location := res.Header.Get("Location"); location != tt.want.location {
				t.Errorf("Location get %v, want %v", location, tt.want.location)
			}
		})
	}
}
