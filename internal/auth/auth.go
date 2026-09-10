package auth

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"go.uber.org/zap"
)

// ctxKey - Специальный тип для использования в контексте
type ctxKey string

// userIDKey - константа для извлечения id пользователя
const userIDKey ctxKey = "user_id"

// UserToken - Токен аутентификации пользователя
type UserToken struct {
	UserID    string
	Token     string
	ExpiresAt time.Time
}

// Claims — Кастомные утверждения для Jwt токена
type Claims struct {
	jwt.RegisteredClaims
	UserID string
}

var (
	ErrInvalidToken  = errors.New("некорректный токен")
	ErrEmptyUserID   = errors.New("пустой id пользователя")
	ErrInvalidMethod = errors.New("некорректный метод шифрования ключа")
)

// NewUserToken - Создание нового пользователя с токеном аутентификации
func NewUserToken(userID string, secret string, tokenExp time.Duration) (UserToken, error) {
	ut := UserToken{
		UserID:    userID,
		ExpiresAt: time.Now().Add(tokenExp),
	}

	// Получаем токен аутентификации
	token, err := buildJWTString(secret, ut.ExpiresAt, userID)
	if err != nil {
		return ut, err
	}

	ut.Token = token

	return ut, nil
}

// buildJWTString создаёт токен и возвращает его в виде строки.
func buildJWTString(secret string, expiresAt time.Time, userID string) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
		UserID: userID,
	})

	return token.SignedString([]byte(secret))
}

// getUserToken - Получаем UserToken по токену аутентификации пользователя и приватному ключу
func getUserToken(authUserToken string, secret string) (UserToken, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(authUserToken, claims, jwtKeyFunc(secret))
	if err != nil {
		return UserToken{}, err
	}

	ut := UserToken{
		UserID:    claims.UserID,
		Token:     authUserToken,
		ExpiresAt: claims.ExpiresAt.Time,
	}

	// Проверяем токен на корректность
	if !token.Valid {
		return ut, ErrInvalidToken
	}

	// Проверяем наличие значения в поле userID
	if strings.Trim(ut.UserID, " ") == "" {
		return ut, ErrEmptyUserID
	}

	return ut, nil
}

// jwtKeyFunc - возвращает функцию декодирования jwt ключа
func jwtKeyFunc(secret string) func(t *jwt.Token) (interface{}, error) {

	return func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidMethod
		}

		return []byte(secret), nil
	}
}

// SetUserID - Добавление id пользователя в контекст
func SetUserID(ctx context.Context, key string) context.Context {
	return context.WithValue(ctx, userIDKey, key)
}

// GetUserID - Извлечение user_id из контекста
func GetUserID(ctx context.Context) (string, bool) {
	user, ok := ctx.Value(userIDKey).(string)
	return user, ok
}

// NewHTTPMiddleware Принимаем приватный ключ, срок жизни токена и логгер, возвращаем функцию-замыкание, которая будет аутентифицировать пользователя
func NewHTTPMiddleware(secret string, tokenExp time.Duration, sugar *zap.SugaredLogger) func(http.Handler) http.Handler {

	return func(h http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			var (
				ut         UserToken
				isNewToken bool
			)

			token, err := r.Cookie("Authorization")

			// Если куки нет - создаём новый токен, иначе пробуем аутентифицировать пользователя
			if err != nil {
				ut, err = NewUserToken(uuid.NewString(), secret, tokenExp)
				if err != nil {
					sugar.Fatalf("ошибка при создании токена : %v", err)
				}
				isNewToken = true
			} else {
				ut, err = getUserToken(token.Value, secret)
				if err != nil {
					// Если при попытке извлечения id пользователя мы получили ошибку "Пустой id" - сразу возвращаем ответ
					if errors.Is(err, ErrEmptyUserID) {
						http.Error(w, http.StatusText(http.StatusUnauthorized), http.StatusUnauthorized)
						return
					}

					//Иначе - создаём новый
					ut, err = NewUserToken(uuid.NewString(), secret, tokenExp)
					if err != nil {
						sugar.Fatalf("ошибка при создании токена : %v", err)
					}
					isNewToken = true
				}
			}

			// Если токен новый - добавляем в ответ новую куку с токеном пользователя
			if isNewToken {
				http.SetCookie(w, &http.Cookie{
					Name:     "Authorization",
					Value:    ut.Token,
					Expires:  ut.ExpiresAt,
					Path:     "/",                     // Действие куки распространяется с корня сайта
					HttpOnly: true,                    // закрывает доступ к куке из JavaScript
					SameSite: http.SameSiteStrictMode, // защита от CSRF атак
				})
			}

			// Добавляем id пользователя в контекст
			ctx := SetUserID(r.Context(), ut.UserID)

			h.ServeHTTP(w, r.WithContext(ctx))

		})
	}
}
