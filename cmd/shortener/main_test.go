package main

import (
	"sync"
	"testing"

	"github.com/alexsey-popov/shorturl/internal/handler"
	"github.com/alexsey-popov/shorturl/internal/model"
	"github.com/alexsey-popov/shorturl/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestConnectDB(t *testing.T) {
	// Некорректный DSN должен приводить к ошибке подключения или миграций
	db, err := connectDB("postgres://invalid:invalid@localhost:5432/nonexistent?sslmode=disable")
	assert.Error(t, err)
	if db != nil {
		db.Close()
	}
}

func TestStartDeleteWorkers(t *testing.T) {
	logger, err := zap.NewDevelopment()
	require.NoError(t, err)
	defer logger.Sync()
	sugar := logger.Sugar()

	s := service.NewMemoryShortener("http://localhost:8080")
	url := model.URL{
		Prefix:      "abc1234",
		OriginalURL: "https://example.com",
		UserID:      "user-1",
		IsDeleted:   false,
	}
	require.NoError(t, s.Rep.Set(url))

	delCh := make(chan handler.DeleteTask, 10)
	var wg sync.WaitGroup
	startDeleteWorkers(2, delCh, &wg, s, sugar)

	delCh <- handler.DeleteTask{
		Prefixes: []string{"abc1234"},
		UserID:   "user-1",
	}

	close(delCh)
	wg.Wait()

	got, err := s.Rep.Get("abc1234")
	assert.NoError(t, err)
	assert.True(t, got.IsDeleted)
}
