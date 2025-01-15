package auth

import (
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

type JWTAccessTokenPayload struct {
	jwt.RegisteredClaims
	UserId      string     `json:"userId"`
	RefreshRand *uuid.UUID `json:"refreshRand"`
}

type JWTPair struct {
	AccessToken  string `json:"accessToken"`
	RefreshToken string `json:"refreshToken"`
}
