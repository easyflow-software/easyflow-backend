package auth

import (
	"easyflow-backend/pkg/api"
	"easyflow-backend/pkg/common"
	"easyflow-backend/pkg/enum"
	"easyflow-backend/pkg/middleware"
	"net/http"

	"github.com/gin-gonic/gin"
)

func RegisterAuthEndpoints(r *gin.RouterGroup) {
	r.Use(middleware.LoggerMiddleware("Auth"))
	r.Use(middleware.RateLimiter(1, 2))
	r.POST("/login", LoginController)
	r.GET("/check", AuthGuard(), CheckLoginController)
	r.GET("/refresh", RefreshAuthGuard(), RefreshController)
	r.GET("/logout", AuthGuard(), LogoutController)
}

func LoginController(c *gin.Context) {
	payload, logger, db, cfg, errors := common.SetupEndpoint[LoginRequest](c)
	if errors != nil {
		c.JSON(http.StatusInternalServerError, api.ApiError{
			Code:    http.StatusInternalServerError,
			Error:   enum.ApiError,
			Details: errors,
		})
		return
	}

	tokens, err := LoginService(db, cfg, payload, logger)
	if err != nil {
		c.JSON(err.Code, err)
		return
	}

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("access_token", tokens.AccessToken, cfg.JwtExpirationTime, "/", cfg.Domain, cfg.Stage == "production", true)
	c.SetCookie("refresh_token", tokens.RefreshToken, cfg.RefreshExpirationTime, "/", cfg.Domain, cfg.Stage == "production", true)

	c.JSON(200, gin.H{
		"accessTokenExpiresIn": cfg.JwtExpirationTime,
	})
}

func CheckLoginController(c *gin.Context) {
	_, ok := c.Get("user")
	if !ok {
		c.JSON(http.StatusInternalServerError, api.ApiError{
			Code:    http.StatusInternalServerError,
			Error:   enum.ApiError,
			Details: "User not found in context",
		})
		return
	}

	// only returns if it comes through the authguard so we can assume the user is logged in
	c.JSON(200, true)
}

func RefreshController(c *gin.Context) {
	_, logger, db, cfg, errors := common.SetupEndpoint[any](c)
	if errors != nil {
		c.JSON(http.StatusInternalServerError, api.ApiError{
			Code:    http.StatusInternalServerError,
			Error:   enum.ApiError,
			Details: errors,
		})
		return
	}

	payload, ok := c.Get("user")
	if !ok {
		c.JSON(http.StatusInternalServerError, api.ApiError{
			Code:  http.StatusInternalServerError,
			Error: enum.ApiError,
		})
		return
	}

	tokens, err := RefreshService(db, cfg, payload.(*JWTAccessTokenPayload), logger)

	if err != nil {
		c.JSON(err.Code, err)
		return
	}

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("access_token", tokens.AccessToken, cfg.JwtExpirationTime, "/", cfg.Domain, cfg.Stage == "production", true)
	c.SetCookie("refresh_token", tokens.RefreshToken, cfg.RefreshExpirationTime, "/", cfg.Domain, cfg.Stage == "production", true)

	c.JSON(200, gin.H{
		"accessTokenExpiresIn": cfg.JwtExpirationTime,
	})
}

func LogoutController(c *gin.Context) {
	_, logger, db, cfg, errors := common.SetupEndpoint[any](c)
	if errors != nil {
		c.JSON(http.StatusInternalServerError, api.ApiError{
			Code:    http.StatusInternalServerError,
			Error:   enum.ApiError,
			Details: errors,
		})
		return
	}

	refresh, err := c.Cookie("refresh_token")
	if err != nil {
		c.JSON(http.StatusBadRequest, api.ApiError{
			Code:    http.StatusBadRequest,
			Error:   enum.InvalidRefreshToken,
			Details: err,
		})
	}

	payload, err := ValidateToken(cfg, refresh)
	if err != nil {
		c.JSON(http.StatusInternalServerError, api.ApiError{
			Code:    http.StatusInternalServerError,
			Error:   enum.ApiError,
			Details: err,
		})
		return
	}

	e := LogoutService(db, payload, logger)
	if e != nil {
		c.JSON(e.Code, e)
		return
	}

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("access_token", "", -1, "/", cfg.Domain, cfg.Stage == "production", true)
	c.SetCookie("refresh_token", "", -1, "/", cfg.Domain, cfg.Stage == "production", true)

	c.JSON(200, gin.H{})
}
