package inmemory

import (
	"encoding/json"
	"sync"
	"testing"

	"github.com/alexsey-popov/shorturl/internal/model"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	rep := New()
	assert.NotNil(t, rep)
	assert.NotNil(t, rep.data)
}

func TestPing(t *testing.T) {
	rep := New()
	assert.NoError(t, rep.Ping())
}

func TestSetAndGet(t *testing.T) {
	rep := New()

	url := model.URL{
		Prefix:      "abc12345",
		OriginalURL: "https://example.com",
		UserID:      "user-1",
		IsDeleted:   false,
	}

	t.Run("success", func(t *testing.T) {
		err := rep.Set(url)
		assert.NoError(t, err)

		got, err := rep.Get("abc12345")
		assert.NoError(t, err)
		assert.Equal(t, url, got)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := rep.Get("nonexistent")
		assert.ErrorIs(t, err, ErrURLNotFound)
	})
}

func TestSetMany(t *testing.T) {
	rep := New()

	urls := []model.URL{
		{Prefix: "pref1", OriginalURL: "https://example.com/1", UserID: "user-1"},
		{Prefix: "pref2", OriginalURL: "https://example.com/2", UserID: "user-1"},
	}

	err := rep.SetMany(urls)
	assert.NoError(t, err)

	got1, err := rep.Get("pref1")
	assert.NoError(t, err)
	assert.Equal(t, "https://example.com/1", got1.OriginalURL)

	got2, err := rep.Get("pref2")
	assert.NoError(t, err)
	assert.Equal(t, "https://example.com/2", got2.OriginalURL)
}

func TestFindFromOriginal(t *testing.T) {
	rep := New()

	url := model.URL{
		Prefix:      "pref1",
		OriginalURL: "https://example.com/orig",
		UserID:      "user-1",
	}
	require.NoError(t, rep.Set(url))

	t.Run("found", func(t *testing.T) {
		got, err := rep.FindFromOriginal("https://example.com/orig")
		assert.NoError(t, err)
		assert.Equal(t, "pref1", got.Prefix)
	})

	t.Run("not found", func(t *testing.T) {
		_, err := rep.FindFromOriginal("https://example.com/unknown")
		assert.ErrorIs(t, err, ErrURLNotFound)
	})
}

func TestFindFromUserID(t *testing.T) {
	rep := New()

	urls := []model.URL{
		{Prefix: "pref1", OriginalURL: "https://example.com/1", UserID: "user-1"},
		{Prefix: "pref2", OriginalURL: "https://example.com/2", UserID: "user-1"},
		{Prefix: "pref3", OriginalURL: "https://example.com/3", UserID: "user-2"},
	}
	require.NoError(t, rep.SetMany(urls))

	t.Run("user-1 urls", func(t *testing.T) {
		got, err := rep.FindFromUserID("user-1")
		assert.NoError(t, err)
		assert.Len(t, got, 2)
	})

	t.Run("user-3 no urls", func(t *testing.T) {
		got, err := rep.FindFromUserID("user-3")
		assert.NoError(t, err)
		assert.Empty(t, got)
	})
}

func TestDeleteUserUrlFromPrefix(t *testing.T) {
	rep := New()

	url := model.URL{
		Prefix:      "pref1",
		OriginalURL: "https://example.com",
		UserID:      "user-1",
		IsDeleted:   false,
	}
	require.NoError(t, rep.Set(url))

	t.Run("wrong user id", func(t *testing.T) {
		err := rep.DeleteUserUrlFromPrefix("pref1", "user-2")
		assert.Error(t, err)
	})

	t.Run("success", func(t *testing.T) {
		err := rep.DeleteUserUrlFromPrefix("pref1", "user-1")
		assert.NoError(t, err)

		got, err := rep.Get("pref1")
		assert.NoError(t, err)
		assert.True(t, got.IsDeleted)
	})

	t.Run("already deleted", func(t *testing.T) {
		err := rep.DeleteUserUrlFromPrefix("pref1", "user-1")
		assert.Error(t, err)
	})

	t.Run("not found", func(t *testing.T) {
		err := rep.DeleteUserUrlFromPrefix("unknown", "user-1")
		assert.ErrorIs(t, err, ErrURLNotFound)
	})
}

func TestDeleteManyFromUserId(t *testing.T) {
	rep := New()

	urls := []model.URL{
		{Prefix: "pref1", OriginalURL: "https://example.com/1", UserID: "user-1"},
		{Prefix: "pref2", OriginalURL: "https://example.com/2", UserID: "user-1"},
		{Prefix: "pref3", OriginalURL: "https://example.com/3", UserID: "user-2"},
	}
	require.NoError(t, rep.SetMany(urls))

	t.Run("batch delete success", func(t *testing.T) {
		err := rep.DeleteManyFromUserId([]string{"pref1", "pref2"}, "user-1")
		assert.NoError(t, err)

		got1, _ := rep.Get("pref1")
		got2, _ := rep.Get("pref2")
		assert.True(t, got1.IsDeleted)
		assert.True(t, got2.IsDeleted)
	})

	t.Run("batch delete with error (wrong user)", func(t *testing.T) {
		// pref3 belongs to user-2, trying to delete as user-1
		err := rep.DeleteManyFromUserId([]string{"pref3"}, "user-1")
		assert.Error(t, err)
	})
}

func TestString(t *testing.T) {
	rep := New()
	url := model.URL{Prefix: "pref1", OriginalURL: "https://example.com", UserID: "user-1"}
	require.NoError(t, rep.Set(url))

	str := rep.String()
	assert.Contains(t, str, "pref1")
	assert.Contains(t, str, "https://example.com")
}

func TestMarshalJSON(t *testing.T) {
	rep := New()
	url := model.URL{Prefix: "pref1", OriginalURL: "https://example.com", UserID: "user-1"}
	require.NoError(t, rep.Set(url))

	data, err := rep.MarshalJSON()
	assert.NoError(t, err)
	assert.NotEmpty(t, data)

	var decoded []model.URL
	err = json.Unmarshal(data, &decoded)
	assert.NoError(t, err)
	assert.Len(t, decoded, 1)
	assert.Equal(t, "pref1", decoded[0].Prefix)
}

func TestConcurrentAccess(t *testing.T) {
	rep := New()
	var wg sync.WaitGroup
	workers := 10
	iterations := 100

	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for j := 0; j < iterations; j++ {
				prefix := string(rune('a'+(workerID*iterations+j)%26)) + string(rune('0'+j%10))
				_ = rep.Set(model.URL{Prefix: prefix, OriginalURL: "https://example.com", UserID: "user-1"})
				_, _ = rep.Get(prefix)
				_, _ = rep.FindFromOriginal("https://example.com")
				_, _ = rep.FindFromUserID("user-1")
				_ = rep.Ping()
				_ = rep.String()
				_, _ = rep.MarshalJSON()
			}
		}(i)
	}

	wg.Wait()
}

func BenchmarkInMemory_Set(b *testing.B) {
	rep := New()
	url := model.URL{
		Prefix:      "abc12345",
		OriginalURL: "https://example.com",
		UserID:      "user-1",
		IsDeleted:   false,
	}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = rep.Set(url)
	}
}

func BenchmarkInMemory_Get(b *testing.B) {
	rep := New()
	url := model.URL{
		Prefix:      "abc12345",
		OriginalURL: "https://example.com",
		UserID:      "user-1",
		IsDeleted:   false,
	}
	_ = rep.Set(url)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = rep.Get("abc12345")
	}
}

func BenchmarkInMemory_FindFromOriginal(b *testing.B) {
	rep := New()
	url := model.URL{
		Prefix:      "abc12345",
		OriginalURL: "https://example.com",
		UserID:      "user-1",
		IsDeleted:   false,
	}
	_ = rep.Set(url)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = rep.FindFromOriginal("https://example.com")
	}
}

func BenchmarkInMemory_FindFromUserID(b *testing.B) {
	rep := New()
	urls := []model.URL{
		{Prefix: "pref1", OriginalURL: "https://example.com/1", UserID: "user-1"},
		{Prefix: "pref2", OriginalURL: "https://example.com/2", UserID: "user-1"},
	}
	_ = rep.SetMany(urls)
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = rep.FindFromUserID("user-1")
	}
}
