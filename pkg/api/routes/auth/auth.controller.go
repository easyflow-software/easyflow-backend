package auth

import (
	"easyflow-backend/pkg/api/endpoint"
	"easyflow-backend/pkg/api/errors"
	"easyflow-backend/pkg/api/middleware"
	"easyflow-backend/pkg/enum"
	"easyflow-backend/pkg/jwt"
	"time"

	"net/http"

	"github.com/gin-gonic/gin"
)

func RegisterAuthEndpoints(r *gin.RouterGroup) {
	r.Use(middleware.LoggerMiddleware("Auth"))
	r.POST("/login", middleware.RateLimiterMiddleware(10, 10*time.Minute), loginController)
	r.GET("/check", middleware.RateLimiterMiddleware(100, 10*time.Minute), AuthGuard(), checkLoginController)
	r.GET("/refresh", middleware.RateLimiterMiddleware(25, 10*time.Minute), RefreshAuthGuard(), refreshController)
	r.GET("/logout", middleware.RateLimiterMiddleware(100, 10*time.Minute), AuthGuard(), logoutController)
}

func loginController(c *gin.Context) {
	payload, logger, db, cfg, _, errs := endpoint.SetupEndpoint[LoginRequest](c)
	if len(errs) > 0 {
		c.JSON(http.StatusInternalServerError, errors.ApiError{
			Code:    http.StatusInternalServerError,
			Error:   enum.ApiError,
			Details: errs,
		})
		return
	}

	tokens, user, err := loginService(db, cfg, payload, c.ClientIP(), logger)
	if err != nil {
		c.JSON(err.Code, err)
		return
	}

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("access_token", tokens.AccessToken, cfg.JwtExpirationTime, "/", cfg.Domain, cfg.Stage == "production", true)
	c.SetCookie("refresh_token", tokens.RefreshToken, cfg.RefreshExpirationTime, "/auth/refresh", cfg.Domain, cfg.Stage == "production", true)

	c.JSON(200, user)
}

func checkLoginController(c *gin.Context) {
	_, logger, _, _, _, errs := endpoint.SetupEndpoint[any](c)
	if len(errs) > 0 {
		c.JSON(http.StatusInternalServerError, errors.ApiError{
			Code:    http.StatusInternalServerError,
			Error:   enum.ApiError,
			Details: errs,
		})
		return
	}

	user, ok := c.Get("user")
	if !ok {
		c.JSON(http.StatusInternalServerError, errors.ApiError{
			Code:    http.StatusInternalServerError,
			Error:   enum.ApiError,
			Details: "User not found in context",
		})
		return
	}

	logger.PrintfDebug("User with id: %s is currently logged in", user.(*jwt.JWTTokenPayload).UserID)

	// only returns if it comes through the authguard so we can assume the user is logged in
	c.JSON(200, true)
}

func refreshController(c *gin.Context) {
	_, logger, db, cfg, _, errs := endpoint.SetupEndpoint[any](c)
	if len(errs) > 0 {
		c.JSON(http.StatusInternalServerError, errors.ApiError{
			Code:    http.StatusInternalServerError,
			Error:   enum.ApiError,
			Details: errs,
		})
		return
	}

	payload, ok := c.Get("user")
	if !ok {
		c.JSON(http.StatusInternalServerError, errors.ApiError{
			Code:  http.StatusInternalServerError,
			Error: enum.ApiError,
		})
		return
	}

	tokens, err := refreshService(db, cfg, payload.(*jwt.JWTTokenPayload), logger)

	if err != nil {
		c.JSON(err.Code, err)
		return
	}

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("access_token", tokens.AccessToken, cfg.JwtExpirationTime, "/", cfg.Domain, cfg.Stage == "production", true)
	c.SetCookie("refresh_token", tokens.RefreshToken, cfg.RefreshExpirationTime, "/auth/refresh", cfg.Domain, cfg.Stage == "production", true)

	c.JSON(200, gin.H{
		"accessTokenExpiresIn": cfg.JwtExpirationTime,
	})
}

func logoutController(c *gin.Context) {
	_, logger, db, cfg, _, errs := endpoint.SetupEndpoint[any](c)
	if len(errs) > 0 {
		c.JSON(http.StatusInternalServerError, errors.ApiError{
			Code:    http.StatusInternalServerError,
			Error:   enum.ApiError,
			Details: errs,
		})
		return
	}

	refresh, err := c.Cookie("refresh_token")
	if err != nil {
		c.JSON(http.StatusBadRequest, errors.ApiError{
			Code:    http.StatusBadRequest,
			Error:   enum.InvalidRefreshToken,
			Details: err,
		})
	}

	payload, err := jwt.ValidateToken(cfg.JwtSecret, refresh)
	if err != nil {
		c.JSON(http.StatusInternalServerError, errors.ApiError{
			Code:    http.StatusInternalServerError,
			Error:   enum.ApiError,
			Details: err,
		})
		return
	}

	e := logoutService(db, payload, logger)
	if e != nil {
		c.JSON(e.Code, e)
		return
	}

	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie("access_token", "", -1, "/", cfg.Domain, cfg.Stage == "production", true)
	c.SetCookie("refresh_token", "", -1, "/auth/refresh", cfg.Domain, cfg.Stage == "production", true)

	c.JSON(200, gin.H{})
}
