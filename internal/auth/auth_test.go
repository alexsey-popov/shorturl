package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

const testSecret = "test-secret-key"

// TestNewUserToken - Тестирование NewUserToken
func TestGetUserId(t *testing.T) {
	userID := "user-123"
	tokenExp := time.Hour

	ut, err := NewUserToken(userID, testSecret, tokenExp)
	require.NoError(t, err)
	assert.NotEmpty(t, ut.Token)
	assert.Equal(t, userID, ut.UserID)
	assert.True(t, ut.ExpiresAt.After(time.Now()))

	parsedUT, err := getUserToken(ut.Token, testSecret)
	require.NoError(t, err)
	assert.Equal(t, userID, parsedUT.UserID)
	assert.Equal(t, ut.Token, parsedUT.Token)
}

// TestGetUserToken - тестирование getUserToken
func TestGetUserToken(t *testing.T) {
	t.Run("negative - некорректный user токен", func(t *testing.T) {
		_, err := getUserToken("invalid.jwt.token", testSecret)
		assert.Error(t, err)
	})

	t.Run("negative - некорректный jwt secret токен", func(t *testing.T) {
		ut, err := NewUserToken("user-123", testSecret, time.Hour)
		require.NoError(t, err)

		_, err = getUserToken(ut.Token, "wrong-secret")
		assert.Error(t, err)
	})

	t.Run("negative - пустой user_id", func(t *testing.T) {
		token, err := buildJWTString(testSecret, time.Now().Add(time.Hour), "")
		require.NoError(t, err)

		_, err = getUserToken(token, testSecret)
		assert.ErrorIs(t, err, ErrEmptyUserID)
	})

	t.Run("negative - некорректный алгоритм подписи", func(t *testing.T) {
		// Create token with none or RS256 instead of HS256
		claims := Claims{
			RegisteredClaims: jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			},
			UserID: "user-123",
		}
		token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
		tokenString, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
		require.NoError(t, err)

		_, err = getUserToken(tokenString, testSecret)
		assert.ErrorIs(t, err, ErrInvalidMethod)
	})
}

// TestContextUser - Тестирование получения токена из контекста
func TestContextUser(t *testing.T) {
	ctx := t.Context()
	userID := "context-user-456"

	_, ok := GetUserId(ctx)
	assert.False(t, ok)

	ctx = SetUserId(ctx, userID)
	retrievedID, ok := GetUserId(ctx)
	assert.True(t, ok)
	assert.Equal(t, userID, retrievedID)
}

// TestNewHTTPMiddleware - Тестирование
func TestNewHTTPMiddleware(t *testing.T) {
	sugar := zap.NewNop().Sugar()
	middleware := NewHTTPMiddleware(testSecret, time.Hour, sugar)

	nextHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userID, ok := GetUserId(r.Context())
		assert.True(t, ok)
		assert.NotEmpty(t, userID)
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	handler := middleware(nextHandler)

	t.Run("positive - запрос без токена создаёт новый токен", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
		cookies := rec.Result().Cookies()
		var authCookie *http.Cookie
		for _, c := range cookies {
			if c.Name == "Authorization" {
				authCookie = c
				break
			}
		}
		require.NotNil(t, authCookie)
		assert.NotEmpty(t, authCookie.Value)
	})

	t.Run("positive - корректный токен в заголовке аутентификации", func(t *testing.T) {
		ut, err := NewUserToken("existing-user", testSecret, time.Hour)
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{
			Name:  "Authorization",
			Value: ut.Token,
		})
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusOK, rec.Code)
	})

	t.Run("negative - запросы к эндпоинтам под аутентификацией должны отдавать 401 код ответа", func(t *testing.T) {
		token, err := buildJWTString(testSecret, time.Now().Add(time.Hour), "")
		require.NoError(t, err)

		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.AddCookie(&http.Cookie{
			Name:  "Authorization",
			Value: token,
		})
		rec := httptest.NewRecorder()

		handler.ServeHTTP(rec, req)

		assert.Equal(t, http.StatusUnauthorized, rec.Code)
	})
}
