package auth

import (
	"easyflow-backend/pkg/jwt"
)

type LoginRequest struct {
	Email          string `json:"email" validate:"required,email"`
	Password       string `json:"password" validate:"required"`
	TurnstileToken string `json:"turnstileToken" validate:"required"`
}

type RefreshTokenResponse struct {
	jwt.JWTPair
	AccessTokenExpires int `json:"accessTokenExpires"`
}
