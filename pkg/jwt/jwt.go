package jwt

import (
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

type JWTTokenPayload struct {
	jwt.RegisteredClaims
	UserID      string `json:"userId"`
	RefreshRand string `json:"refreshRand"`
	IsRefresh   bool   `json:"isRefresh"`
}

type JWTPair struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}

func CreateTokenPayload(userId string, random string, expires time.Time, isRefresh bool) JWTTokenPayload {
	return JWTTokenPayload{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expires),
			Issuer:    "easyflow",
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
		UserID:      userId,
		RefreshRand: random,
		IsRefresh:   isRefresh,
	}
}

func GenerateJwt[T interface{ jwt.Claims }](secret string, payload T) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)
	signedToken, err := token.SignedString([]byte(secret))
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return signedToken, nil
}

func ValidateToken(secret string, token string) (*JWTTokenPayload, error) {
	var claims JWTTokenPayload
	_, err := jwt.ParseWithClaims(
		token,
		&claims,
		func(token *jwt.Token) (interface{}, error) {
			// Verify that the signing method is what we expect
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return []byte(secret), nil
		},
	)

	if err != nil {
		return nil, err
	}

	return &claims, nil
}
