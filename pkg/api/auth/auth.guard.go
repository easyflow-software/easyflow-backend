package auth

import (
	"easyflow-backend/pkg/api"
	"easyflow-backend/pkg/common"
	"easyflow-backend/pkg/database"
	"easyflow-backend/pkg/enum"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func AuthGuard() gin.HandlerFunc {
	return func(c *gin.Context) {
		_, logger, _, cfg, errs := common.SetupEndpoint[any](c)
		if errs != nil {
			c.JSON(http.StatusInternalServerError, api.ApiError{
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
			c.JSON(http.StatusBadRequest, api.ApiError{
				Code:  http.StatusBadRequest,
				Error: enum.InvalidCookie,
			})
			c.Abort()
			return
		}

		if accessToken == "" {
			logger.PrintfDebug("No access token provided")
			c.JSON(http.StatusBadGateway, api.ApiError{
				Code:  http.StatusBadRequest,
				Error: enum.InvalidAccessToken,
			})
			c.Abort()
			return
		}

		// Validate token
		payload, err := ValidateToken(cfg, accessToken)
		if err != nil {
			logger.PrintfDebug("Error validating token: %s", err.Error())
			if errors.Is(err, jwt.ErrTokenExpired) {
				c.JSON(498, api.ApiError{
					Code:    498, // token expired/invalid
					Error:   enum.ExpiredAccessToken,
					Details: err,
				})
			}
			c.JSON(498, api.ApiError{
				Code:    498, // token expired/invalid
				Error:   enum.InvalidAccessToken,
				Details: err,
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
		_, logger, db, cfg, errs := common.SetupEndpoint[any](c)
		if errs != nil {
			c.JSON(http.StatusInternalServerError, api.ApiError{
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
			c.JSON(http.StatusBadRequest, api.ApiError{
				Code:  http.StatusBadRequest,
				Error: enum.InvalidCookie,
			})
			c.Abort()
			return
		}

		if refreshToken == "" {
			logger.PrintfDebug("No refresh token provided")
			c.JSON(498, api.ApiError{
				Code:  498, // token expired/invalid
				Error: enum.InvalidAccessToken,
			})
			c.Abort()
			return
		}

		token, err := ValidateToken(cfg, refreshToken)
		if err != nil {
			logger.PrintfError("Error validating token: %s", err.Error())
			if errors.Is(err, jwt.ErrTokenExpired) {
				c.JSON(498, api.ApiError{
					Code:  498, // token expired/invalid
					Error: enum.ExpiredRefreshToken,
				})
				c.Abort()
				return
			}
			c.JSON(498, api.ApiError{
				Code:  498, // token expired/invalid
				Error: enum.InvalidRefreshToken,
			})
			c.Abort()
			return
		}

		if err := db.First(&database.UserKeys{}, "user_id = ? AND random = ?", token.UserId, token.RefreshRand).Error; err != nil {
			logger.PrintfDebug("refresh token not found in db")
			c.JSON(498, api.ApiError{
				Code:  498,
				Error: enum.InvalidRefreshToken,
			})

			c.Abort()
			return
		}

		c.Set("user", token)
		c.Next()
	}
}
