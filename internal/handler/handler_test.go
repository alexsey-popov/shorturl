package handler

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/alexsey-popov/shorturl/internal/audit"
	"github.com/alexsey-popov/shorturl/internal/auth"
	"github.com/alexsey-popov/shorturl/internal/config"
	"github.com/alexsey-popov/shorturl/internal/model"
	"github.com/alexsey-popov/shorturl/internal/service"
	"github.com/alexsey-popov/shorturl/pkg/contentType"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"go.uber.org/zap"
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
				statusCode:  http.StatusInternalServerError,
				contentType: contentType.Plain,
			},
		},
		{
			name:        "negative #2 - empty body",
			target:      "/",
			contentType: contentType.Plain,
			body:        "",
			want: want{
				statusCode:  http.StatusInternalServerError,
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

	h := NewHandler(zap.S(), service.NewMemoryShortener(config.Server.BaseURL), nil, nil)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := httptest.NewRequest(http.MethodPost, tt.target, strings.NewReader(tt.body))
			r.Header.Set("Content-Type", tt.contentType)

			// Прокидываем id пользователя в контекст
			ctx := auth.SetUserId(r.Context(), uuid.NewString())
			r = r.WithContext(ctx)

			w := httptest.NewRecorder()

			h.HandlePost(w, r)

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
	h := NewHandler(zap.S(), service.NewMemoryShortener(config.Server.BaseURL), nil, nil)

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
				statusCode:  http.StatusInternalServerError,
				contentType: contentType.Plain,
			},
		},
		{
			name:        "negative #2 - empty body",
			target:      "/api/shorten",
			contentType: contentType.JSON,
			body:        "",
			want: want{
				statusCode:  http.StatusInternalServerError,
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

			// Прокидываем id пользователя в контекст
			ctx := auth.SetUserId(r.Context(), uuid.NewString())
			r = r.WithContext(ctx)

			w := httptest.NewRecorder()

			h.HandlePostJSON(w, r)

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
	h := NewHandler(zap.S(), service.NewMemoryShortener(config.Server.BaseURL), nil, nil)

	// Добавляем в shortener заранее известную пару prefix => originalURL
	prefix, originalURL, userID := "positive1", "https://example.com/positive1", ""
	if err := h.shortener.Rep.Set(model.New(prefix, originalURL, userID)); err != nil {
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
				statusCode: http.StatusNotFound,
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

			h.HandleGet(w, r)

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

type fakeObserver struct {
	mu     sync.Mutex
	events []audit.Event
}

func (f *fakeObserver) Update(event audit.Event) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.events = append(f.events, event)
}

func TestHandlerAudit(t *testing.T) {
	l := zap.NewNop().Sugar()
	p := audit.NewPublisher(l)
	spy := &fakeObserver{}
	p.Register(spy)

	h := NewHandler(zap.S(), service.NewMemoryShortener(config.Server.BaseURL), nil, p)

	// 1. Test POST /
	r1 := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://example.com/test1"))
	r1.Header.Set("Content-Type", contentType.Plain)
	ctx1 := auth.SetUserId(r1.Context(), "user1")
	r1 = r1.WithContext(ctx1)
	w1 := httptest.NewRecorder()
	h.HandlePost(w1, r1)

	// 2. Test POST /api/shorten
	r2 := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(`{"url":"https://example.com/test2"}`))
	r2.Header.Set("Content-Type", contentType.JSON)
	ctx2 := auth.SetUserId(r2.Context(), "user1")
	r2 = r2.WithContext(ctx2)
	w2 := httptest.NewRecorder()
	h.HandlePostJSON(w2, r2)

	// 3. Test GET /{id}
	prefix := "test123"
	originalURL := "https://example.com/test1"
	err := h.shortener.Rep.Set(model.New(prefix, originalURL, "user1"))
	if err != nil {
		t.Fatal(err)
	}

	r3 := httptest.NewRequest(http.MethodGet, "/"+prefix, http.NoBody)
	rctx := chi.NewRouteContext()
	rctx.URLParams.Add("id", prefix)
	r3 = r3.WithContext(context.WithValue(r3.Context(), chi.RouteCtxKey, rctx))
	w3 := httptest.NewRecorder()
	h.HandleGet(w3, r3)

	spy.mu.Lock()
	defer spy.mu.Unlock()

	if len(spy.events) != 3 {
		t.Fatalf("expected 3 audit events, got %d", len(spy.events))
	}

	if spy.events[0].Action != "shorten" || spy.events[0].URL != "https://example.com/test1" {
		t.Errorf("unexpected event 0: %+v", spy.events[0])
	}
	if spy.events[1].Action != "shorten" || spy.events[1].URL != "https://example.com/test2" {
		t.Errorf("unexpected event 1: %+v", spy.events[1])
	}
	if spy.events[2].Action != "follow" || spy.events[2].URL != originalURL {
		t.Errorf("unexpected event 2: %+v", spy.events[2])
	}
}
