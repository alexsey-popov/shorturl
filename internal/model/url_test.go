package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	prefix, orig, user := "test", "http://test.test/test", "user_1"

	url := New(prefix, orig, user)

	assert.Equal(t, url.UserID, user)
	assert.Equal(t, url.Prefix, prefix)
	assert.Equal(t, url.OriginalURL, orig)
	assert.NotEmpty(t, url.UUID)
}
