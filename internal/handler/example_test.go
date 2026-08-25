package handler_test

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"

	"github.com/alexsey-popov/shorturl/internal/auth"
	"github.com/alexsey-popov/shorturl/internal/config"
	"github.com/alexsey-popov/shorturl/internal/handler"
	"github.com/alexsey-popov/shorturl/internal/service"
	"github.com/alexsey-popov/shorturl/pkg/contentType"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ExampleNewHandler демонстрирует создание хендера и сокращение URL через текстовый эндпоинт HandlePost.
func ExampleNewHandler() {
	logger := zap.NewNop().Sugar()
	cfg := config.NewEmpty()
	shortener := service.NewMemoryShortener(cfg.BaseURL)

	h := handler.NewHandler(logger, shortener, nil, nil)

	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("https://example.com"))
	req.Header.Set("Content-Type", contentType.Plain)
	req = req.WithContext(auth.SetUserId(req.Context(), uuid.NewString()))

	rec := httptest.NewRecorder()
	h.HandlePost(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)
	fmt.Printf("Status: %d\n", res.StatusCode)
	fmt.Printf("Has BaseURL in body: %v\n", strings.HasPrefix(string(body), cfg.BaseURL))

	// Output:
	// Status: 201
	// Has BaseURL in body: true
}

// ExampleHandler_HandlePostJSON демонстрирует работу с JSON-эндпоинтом сокращения URL HandlePostJSON.
func ExampleHandler_HandlePostJSON() {
	logger := zap.NewNop().Sugar()
	cfg := config.NewEmpty()
	shortener := service.NewMemoryShortener(cfg.BaseURL)

	h := handler.NewHandler(logger, shortener, nil, nil)

	reqBody := `{"url":"https://example.com/json"}`
	req := httptest.NewRequest(http.MethodPost, "/api/shorten", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", contentType.JSON)
	req = req.WithContext(auth.SetUserId(req.Context(), uuid.NewString()))

	rec := httptest.NewRecorder()
	h.HandlePostJSON(rec, req)

	res := rec.Result()
	defer res.Body.Close()

	body, _ := io.ReadAll(res.Body)
	fmt.Printf("Status: %d\n", res.StatusCode)
	fmt.Printf("Contains result: %v\n", strings.Contains(string(body), "result"))

	// Output:
	// Status: 201
	// Contains result: true
}
