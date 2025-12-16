package token

import (
	"os"
	"time"

	jwt "github.com/golang-jwt/jwt/v4"
)

type JWTGenerator struct {
	Secret   []byte
	ExpireIn time.Duration
}

func NewJWTGenerator() *JWTGenerator {
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "secret key"
	}
	expStr := os.Getenv("JWT_EXPIRE")
	exp := time.Hour
	if expStr != "" {
		if d, err := time.ParseDuration(expStr); err == nil && d > 0 {
			exp = d
		}
	}
	return &JWTGenerator{Secret: []byte(secret), ExpireIn: exp}
}

func (g *JWTGenerator) Generate(userID uint, username string) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"sub":      userID,
		"username": username,
		"iat":      now.Unix(),
		"exp":      now.Add(g.ExpireIn).Unix(),
	}
	t := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return t.SignedString(g.Secret)
}
