package service

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/alexsey-popov/shorturl/internal/model"
	"github.com/stretchr/testify/assert"
)

type mockRepository struct {
	setFunc          func(url model.URL) error
	setManyFunc      func(urls []model.URL) error
	getFunc          func(prefix string) (model.URL, error)
	findFromUserID   func(userID string) ([]model.URL, error)
	findFromOriginal func(originalURL string) (model.URL, error)
	deleteManyFunc   func(prefixes []string, userID string) error
	pingFunc         func() error
}

func (m *mockRepository) Set(url model.URL) error {
	if m.setFunc != nil {
		return m.setFunc(url)
	}
	return nil
}

func (m *mockRepository) SetMany(urls []model.URL) error {
	if m.setManyFunc != nil {
		return m.setManyFunc(urls)
	}
	return nil
}

func (m *mockRepository) Get(prefix string) (model.URL, error) {
	if m.getFunc != nil {
		return m.getFunc(prefix)
	}
	return model.URL{}, nil
}

func (m *mockRepository) FindFromUserID(userID string) ([]model.URL, error) {
	if m.findFromUserID != nil {
		return m.findFromUserID(userID)
	}
	return nil, nil
}

func (m *mockRepository) FindFromOriginal(originalURL string) (model.URL, error) {
	if m.findFromOriginal != nil {
		return m.findFromOriginal(originalURL)
	}
	return model.URL{}, nil
}

func (m *mockRepository) DeleteManyFromUserId(prefixes []string, userID string) error {
	if m.deleteManyFunc != nil {
		return m.deleteManyFunc(prefixes, userID)
	}
	return nil
}

func (m *mockRepository) Ping() error {
	if m.pingFunc != nil {
		return m.pingFunc()
	}
	return nil
}

func TestNewMemoryShortener(t *testing.T) {
	s := NewMemoryShortener("http://localhost:8080")
	assert.NotNil(t, s.Rep)
	assert.Equal(t, "http://localhost:8080", s.BaseURL)
}

func TestNewFileShortener(t *testing.T) {
	tmpDir := t.TempDir()
	filePath := filepath.Join(tmpDir, "urls.json")

	t.Run("success", func(t *testing.T) {
		s, err := NewFileShortener("http://localhost:8080", filePath)
		assert.NoError(t, err)
		assert.NotNil(t, s.Rep)
		assert.Equal(t, "http://localhost:8080", s.BaseURL)
	})

	t.Run("error (invalid path)", func(t *testing.T) {
		// Passing directory as file path should fail
		s, err := NewFileShortener("http://localhost:8080", tmpDir)
		assert.Error(t, err)
		assert.NotNil(t, s)
	})
}

func TestNewDBShortener(t *testing.T) {
	s := NewDBShortener("http://localhost:8080", nil)
	assert.NotNil(t, s.Rep)
	assert.Equal(t, "http://localhost:8080", s.BaseURL)
}

func TestCheckValidURL(t *testing.T) {
	s := NewMemoryShortener("http://localhost:8080")

	t.Run("valid url", func(t *testing.T) {
		err := s.CheckValidURL("https://example.com/path?query=1")
		assert.NoError(t, err)
	})

	t.Run("invalid url", func(t *testing.T) {
		err := s.CheckValidURL("://invalid-url")
		assert.Error(t, err)
	})
}

func TestGetURLFromPrefix(t *testing.T) {
	s := NewMemoryShortener("http://localhost:8080")
	u, err := s.GetURLFromPrefix("abc123")
	assert.NoError(t, err)
	assert.Equal(t, "http://localhost:8080/abc123", u)
}

func TestAdd(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockRep := &mockRepository{
			setFunc: func(url model.URL) error {
				return nil
			},
		}
		s := Shortener{
			Rep:     mockRep,
			BaseURL: "http://localhost:8080",
		}

		shortURL, err := s.Add("https://example.com", "user-1")
		assert.NoError(t, err)
		assert.Contains(t, shortURL, "http://localhost:8080/")
	})

	t.Run("invalid url error", func(t *testing.T) {
		s := NewMemoryShortener("http://localhost:8080")
		_, err := s.Add("not-a-valid-url", "user-1")
		assert.Error(t, err)
	})

	t.Run("repository set error", func(t *testing.T) {
		mockRep := &mockRepository{
			setFunc: func(url model.URL) error {
				return errors.New("db error")
			},
		}
		s := Shortener{
			Rep:     mockRep,
			BaseURL: "http://localhost:8080",
		}

		_, err := s.Add("https://example.com", "user-1")
		assert.Error(t, err)
	})
}

func TestAddMany(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		mockRep := &mockRepository{
			setManyFunc: func(urls []model.URL) error {
				return nil
			},
		}
		s := Shortener{
			Rep:     mockRep,
			BaseURL: "http://localhost:8080",
		}

		urls := []string{"https://example.com/1", "https://example.com/2"}
		mapURLs, err := s.AddMany(urls, "user-1")
		assert.NoError(t, err)
		assert.Len(t, mapURLs, 2)
		for _, shortURL := range mapURLs {
			assert.Contains(t, shortURL, "http://localhost:8080/")
		}
	})

	t.Run("invalid url in batch error", func(t *testing.T) {
		s := NewMemoryShortener("http://localhost:8080")
		urls := []string{"https://example.com/1", "://invalid"}
		_, err := s.AddMany(urls, "user-1")
		assert.Error(t, err)
	})

	t.Run("repository setMany error", func(t *testing.T) {
		mockRef := &mockRepository{
			setManyFunc: func(urls []model.URL) error {
				return errors.New("db set many error")
			},
		}
		s := Shortener{
			Rep:     mockRef,
			BaseURL: "http://localhost:8080",
		}

		urls := []string{"https://example.com/1"}
		_, err := s.AddMany(urls, "user-1")
		assert.Error(t, err)
	})
}
