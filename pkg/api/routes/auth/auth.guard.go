package auth

import (
	"easyflow-backend/pkg/api/endpoint"
	"easyflow-backend/pkg/api/errors"
	"easyflow-backend/pkg/database"
	"easyflow-backend/pkg/enum"
	"easyflow-backend/pkg/jwt"

	e "errors"

	"net/http"

	"github.com/gin-gonic/gin"
	jwtlib "github.com/golang-jwt/jwt/v5"
)

func AuthGuard() gin.HandlerFunc {
	return func(c *gin.Context) {
		_, logger, _, cfg, _, errs := endpoint.SetupEndpoint[any](c)
		if errs != nil {
			c.JSON(http.StatusInternalServerError, errors.ApiError{
				Code:    http.StatusInternalServerError,
				Error:   enum.ApiError,
				Details: errs,
			})
			c.Abort()
			return
		}

		// Get access_token from cookies
		accessToken, err := c.Cookie("access_token")
		if err != nil {
			logger.PrintfDebug("Error while getting access token cookie: %s", err.Error())
			c.JSON(http.StatusBadRequest, errors.ApiError{
				Code:  http.StatusBadRequest,
				Error: enum.InvalidCookie,
			})
			c.Abort()
			return
		}

		if accessToken == "" {
			logger.PrintfDebug("No access token provided")
			c.JSON(http.StatusBadGateway, errors.ApiError{
				Code:  http.StatusBadRequest,
				Error: enum.InvalidAccessToken,
			})
			c.Abort()
			return
		}

		// Validate token
		payload, err := jwt.ValidateToken(cfg.JwtSecret, accessToken)
		if err != nil {
			logger.PrintfDebug("Error validating token: %s", err.Error())
			if e.Is(err, jwtlib.ErrTokenExpired) {
				c.JSON(http.StatusUnauthorized, errors.ApiError{
					Code:    http.StatusUnauthorized, // token expired/invalid
					Error:   enum.ExpiredAccessToken,
					Details: err,
				})
			}
			c.JSON(http.StatusUnauthorized, errors.ApiError{
				Code:    http.StatusUnauthorized, // token expired/invalid
				Error:   enum.InvalidAccessToken,
				Details: err,
			})
			c.Abort()
			return
		}

		if payload.IsRefresh {
			c.JSON(http.StatusBadRequest, errors.ApiError{
				Code:  http.StatusBadRequest,
				Error: enum.InvalidAccessToken,
			})
			c.Abort()
			return
		}

		// Set user payload in context
		c.Set("user", payload)
		c.Next()
	}
}

func RefreshAuthGuard() gin.HandlerFunc {
	return func(c *gin.Context) {
		_, logger, db, cfg, _, errs := endpoint.SetupEndpoint[any](c)
		if errs != nil {
			c.JSON(http.StatusInternalServerError, errors.ApiError{
				Code:    http.StatusInternalServerError,
				Error:   enum.ApiError,
				Details: errs,
			})
			c.Abort()
			return
		}

		refreshToken, err := c.Cookie("refresh_token")
		if err != nil {
			logger.PrintfDebug("Error while getting refresh token cookie: %s", err.Error())
			c.JSON(http.StatusBadRequest, errors.ApiError{
				Code:  http.StatusBadRequest,
				Error: enum.InvalidCookie,
			})
			c.Abort()
			return
		}

		if refreshToken == "" {
			logger.PrintfDebug("No refresh token provided")
			c.JSON(http.StatusUnauthorized, errors.ApiError{
				Code:  http.StatusUnauthorized, // token expired/invalid
				Error: enum.InvalidAccessToken,
			})
			c.Abort()
			return
		}

		token, err := jwt.ValidateToken(cfg.JwtSecret, refreshToken)
		if err != nil {
			logger.PrintfError("Error validating token: %s", err.Error())
			if e.Is(err, jwtlib.ErrTokenExpired) {
				c.JSON(http.StatusUnauthorized, errors.ApiError{
					Code:  http.StatusUnauthorized, // token expired/invalid
					Error: enum.ExpiredRefreshToken,
				})
				c.Abort()
				return
			}
			c.JSON(http.StatusUnauthorized, errors.ApiError{
				Code:  http.StatusUnauthorized, // token expired/invalid
				Error: enum.InvalidRefreshToken,
			})
			c.Abort()
			return
		}

		if !token.IsRefresh {
			c.JSON(http.StatusBadRequest, errors.ApiError{
				Code:  http.StatusBadRequest,
				Error: enum.InvalidRefreshToken,
			})
			c.Abort()
			return
		}

		if err := db.First(&database.UserKeys{}, "user_id = ? AND random = ?", token.UserID, token.RefreshRand).Error; err != nil {
			logger.PrintfDebug("Refresh token with user id: %s and random: %s not found in db", token.UserID, token.RefreshRand)
			c.JSON(http.StatusUnauthorized, errors.ApiError{
				Code:  http.StatusUnauthorized,
				Error: enum.InvalidRefreshToken,
			})

			c.Abort()
			return
		}

		c.Set("user", token)
		c.Next()
	}
}
