package infile

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/alexsey-popov/shorturl/internal/model"
	errors2 "github.com/alexsey-popov/shorturl/pkg/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("new file not exist", func(t *testing.T) {
		filename := filepath.Join(tmpDir, "nonexistent.json")
		rep, err := New(filename)
		assert.NoError(t, err)
		assert.NotNil(t, rep)
	})

	t.Run("existing file with valid data", func(t *testing.T) {
		filename := filepath.Join(tmpDir, "existing.json")
		urls := []model.URL{
			model.New("pref1", "https://example.com/1", "user-1"),
		}
		data, err := json.Marshal(urls)
		require.NoError(t, err)
		err = os.WriteFile(filename, data, 0666)
		require.NoError(t, err)

		rep, err := New(filename)
		assert.NoError(t, err)
		assert.NotNil(t, rep)

		url, err := rep.Get("pref1")
		assert.NoError(t, err)
		assert.Equal(t, "https://example.com/1", url.OriginalURL)
	})

	t.Run("existing file with invalid json", func(t *testing.T) {
		filename := filepath.Join(tmpDir, "invalid.json")
		err := os.WriteFile(filename, []byte("invalid-json"), 0666)
		require.NoError(t, err)

		rep, err := New(filename)
		assert.Error(t, err)
		assert.Nil(t, rep)
	})

	t.Run("read file error", func(t *testing.T) {
		// Passing a directory as filename should cause read error
		rep, err := New(tmpDir)
		assert.Error(t, err)
		assert.Nil(t, rep)
	})
}

func TestPing(t *testing.T) {
	rep := &InFile{}
	assert.NoError(t, rep.Ping())
}

func TestGet(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test.json")
	rep, err := New(filename)
	require.NoError(t, err)

	url := model.New("pref1", "https://example.com", "user-1")
	err = rep.Set(url)
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		got, err := rep.Get("pref1")
		assert.NoError(t, err)
		assert.Equal(t, url.OriginalURL, got.OriginalURL)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := rep.Get("unknown")
		assert.Error(t, err)
	})
}

func TestSet(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test.json")
	rep, err := New(filename)
	require.NoError(t, err)

	url1 := model.New("pref1", "https://example.com/1", "user-1")

	t.Run("success", func(t *testing.T) {
		err := rep.Set(url1)
		assert.NoError(t, err)

		// Check file was updated
		data, err := os.ReadFile(filename)
		assert.NoError(t, err)
		assert.NotEmpty(t, data)
	})

	t.Run("original url conflict", func(t *testing.T) {
		url2 := model.New("pref2", "https://example.com/1", "user-2")
		err := rep.Set(url2)
		assert.Error(t, err)
		var conflictErr *errors2.OriginalURLConflictError
		assert.True(t, errors.As(err, &conflictErr))
	})

	t.Run("update file error", func(t *testing.T) {
		badRep := &InFile{
			filename: tmpDir, // directory cannot be written as file
			data:     rep.data,
		}
		err := badRep.UpdateFile()
		assert.Error(t, err)
	})
}

func TestSetMany(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test.json")
	rep, err := New(filename)
	require.NoError(t, err)

	urls := []model.URL{
		model.New("pref1", "https://example.com/1", "user-1"),
		model.New("pref2", "https://example.com/2", "user-1"),
	}

	t.Run("success", func(t *testing.T) {
		err := rep.SetMany(urls)
		assert.NoError(t, err)

		got1, err := rep.Get("pref1")
		assert.NoError(t, err)
		assert.Equal(t, "https://example.com/1", got1.OriginalURL)
	})
}

func TestFindMethods(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test.json")
	rep, err := New(filename)
	require.NoError(t, err)

	url := model.New("pref1", "https://example.com", "user-1")
	err = rep.Set(url)
	require.NoError(t, err)

	t.Run("FindFromOriginal success", func(t *testing.T) {
		got, err := rep.FindFromOriginal("https://example.com")
		assert.NoError(t, err)
		assert.Equal(t, "pref1", got.Prefix)
	})

	t.Run("FindFromUserID success", func(t *testing.T) {
		got, err := rep.FindFromUserID("user-1")
		assert.NoError(t, err)
		assert.Len(t, got, 1)
	})
}

func TestDeleteManyFromUserId(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test.json")
	rep, err := New(filename)
	require.NoError(t, err)

	url := model.New("pref1", "https://example.com", "user-1")
	err = rep.Set(url)
	require.NoError(t, err)

	t.Run("success", func(t *testing.T) {
		err := rep.DeleteManyFromUserId([]string{"pref1"}, "user-1")
		assert.NoError(t, err)

		got, err := rep.Get("pref1")
		assert.NoError(t, err)
		assert.True(t, got.IsDeleted)
	})
}

func TestMarshalJSON(t *testing.T) {
	tmpDir := t.TempDir()
	filename := filepath.Join(tmpDir, "test.json")
	rep, err := New(filename)
	require.NoError(t, err)

	url := model.New("pref1", "https://example.com", "user-1")
	err = rep.Set(url)
	require.NoError(t, err)

	data, err := rep.MarshalJSON()
	assert.NoError(t, err)
	assert.NotEmpty(t, data)
}
