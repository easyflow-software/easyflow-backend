package auth

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type RefreshTokenResponse struct {
	JWTPair
	AccessTokenExpires int `json:"accessTokenExpires"`
}
