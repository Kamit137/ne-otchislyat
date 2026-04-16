package token

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var jwtKey = []byte("bibaaboba")

type Claims struct {
	Email string `json:"email"`
	jwt.RegisteredClaims
}

func GenerateToken(email string) (string, error) {
	expirationTime := time.Now().Add(24 * time.Hour)
	claims := &Claims{
		Email: email,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtKey)
}

func ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}
	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return jwtKey, nil
	})
	if err != nil || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Определяем, API ли это запрос
		isAPI := strings.HasPrefix(r.URL.Path, "/api/") ||
			r.Header.Get("Accept") == "application/json" ||
			r.Header.Get("Content-Type") == "application/json"

		cookie, err := r.Cookie("token")
		if err != nil {
			if isAPI {
				// Для API возвращаем JSON ошибку
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{
					"error":   "Unauthorized",
					"message": "Authorization required",
				})
				return
			}
			// Для обычных страниц - редирект
			http.Redirect(w, r, "/registration", http.StatusFound)
			return
		}

		claims, err := ValidateToken(cookie.Value)
		if err != nil {
			// Удаляем испорченную куку
			http.SetCookie(w, &http.Cookie{
				Name:     "token",
				Value:    "",
				Path:     "/",
				Expires:  time.Unix(0, 0),
				MaxAge:   -1,
				HttpOnly: true,
			})

			if isAPI {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusUnauthorized)
				json.NewEncoder(w).Encode(map[string]string{
					"error":   "Unauthorized",
					"message": "Invalid or expired token",
				})
				return
			}
			http.Redirect(w, r, "/registration", http.StatusFound)
			return
		}

		ctx := context.WithValue(r.Context(), "email", claims.Email)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}

func AuthOptionalMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		cookie, err := r.Cookie("token")
		if err == nil {
			claims, err := ValidateToken(cookie.Value)
			if err == nil {
				ctx := context.WithValue(r.Context(), "email", claims.Email)
				next.ServeHTTP(w, r.WithContext(ctx))
				return
			}
		}
		// Если нет токена или он невалидный - просто передаем пустой контекст
		ctx := context.WithValue(r.Context(), "email", "")
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}
