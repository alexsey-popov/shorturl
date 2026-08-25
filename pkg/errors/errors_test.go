package errors

import (
	"errors"
	"testing"

	"github.com/alexsey-popov/shorturl/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestOriginalURLConflictError(t *testing.T) {
	baseErr := errors.New("unique constraint violation")
	diffURL := model.URL{
		UUID:        "123",
		Prefix:      "abc1234",
		OriginalURL: "https://example.com",
		UserID:      "user-1",
		IsDeleted:   false,
	}

	err := NewErrOriginalURLConflict(diffURL, baseErr)

	t.Run("Error string", func(t *testing.T) {
		assert.Equal(t, "unique constraint violation", err.Error())
	})

	t.Run("Unwrap", func(t *testing.T) {
		assert.ErrorIs(t, err, baseErr)

		var conflictErr *OriginalURLConflictError
		assert.True(t, errors.As(err, &conflictErr))
		assert.Equal(t, diffURL, conflictErr.DiffURL)
		assert.Equal(t, baseErr, conflictErr.Err)
	})
}
