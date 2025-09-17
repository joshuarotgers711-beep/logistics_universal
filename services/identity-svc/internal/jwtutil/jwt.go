package jwtutil

import (
	"os"
	"strings"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type Claims = jwt.MapClaims

func secret() []byte {
	if f := os.Getenv("JWT_SECRET_FILE"); f != "" {
		if b, err := os.ReadFile(f); err == nil {
			return []byte(strings.TrimSpace(string(b)))
		}
	}
	s := os.Getenv("JWT_SECRET")
	if s == "" {
		s = "devsecret"
	}
	return []byte(s)
}

func Sign(claims Claims) (string, error) {
	if _, ok := claims["exp"]; !ok {
		claims["exp"] = time.Now().Add(15 * time.Minute).Unix()
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(secret())
}

func Parse(token string) (Claims, error) {
	parsed, err := jwt.Parse(token, func(t *jwt.Token) (interface{}, error) { return secret(), nil })
	if err != nil || !parsed.Valid {
		return nil, err
	}
	if c, ok := parsed.Claims.(jwt.MapClaims); ok {
		return c, nil
	}
	return nil, jwt.ErrTokenInvalidClaims
}
