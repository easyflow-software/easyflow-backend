package middleware

import (
	"easyflow-backend/pkg/api/errors"
	"easyflow-backend/pkg/config"
	"easyflow-backend/pkg/enum"
	"easyflow-backend/pkg/logger"

	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	e "errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/valkey-io/valkey-go"
)

func RateLimiterMiddleware(requests int, timeframe time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		rawCfg, ok := c.Get("config")
		if !ok {
			c.JSON(http.StatusInternalServerError, errors.ApiError{
				Code:    http.StatusInternalServerError,
				Error:   "ConfigError",
				Details: "Config not found in context",
			})
			c.Abort()
			return
		}

		cfg, ok := rawCfg.(*config.Config)
		if !ok {
			c.JSON(http.StatusInternalServerError, errors.ApiError{
				Code:    http.StatusInternalServerError,
				Error:   "ConfigError",
				Details: "Config is not of type *common.Config",
			})
			c.Abort()
			return
		}

		rawLogger, ok := c.Get("logger")
		if !ok {
			c.JSON(http.StatusInternalServerError, errors.ApiError{
				Code:    http.StatusInternalServerError,
				Error:   "LoggerError",
				Details: "Logger not found in context",
			})
			c.Abort()
			return
		}

		logger, ok := rawLogger.(*logger.Logger)
		if !ok {
			c.JSON(http.StatusInternalServerError, errors.ApiError{
				Code:    http.StatusInternalServerError,
				Error:   "LoggerError",
				Details: "Logger is not of type *logger.Logger",
			})
			c.Abort()
			return
		}

		rawValkeyClient, ok := c.Get("valkey")
		if !ok {
			c.JSON(http.StatusInternalServerError, errors.ApiError{
				Code:    http.StatusInternalServerError,
				Error:   "ValkeyError",
				Details: "Valkey not found in context",
			})
			c.Abort()
			return
		}

		valkeyClient, ok := rawValkeyClient.(valkey.Client)
		if !ok {
			c.JSON(http.StatusInternalServerError, errors.ApiError{
				Code:    http.StatusInternalServerError,
				Error:   "ValkeyError",
				Details: "Valkey is not of type valkey.Client",
			})
			c.Abort()
			return
		}

		var userID string
		userIDCookie, err := c.Request.Cookie("user_id")
		if err != nil {
			logger.PrintfDebug("Request has no user_id. Setting cookie in response and using alternate user_id instead")
			signedCookie, err := signedUserID(cfg)
			if err != nil {
				c.JSON(http.StatusInternalServerError, errors.ApiError{
					Code:    http.StatusInternalServerError,
					Error:   enum.ApiError,
					Details: err,
				})
				c.Abort()
				return
			}
			c.SetCookie("user_id", signedCookie, 60*60*24*365, "/", cfg.Domain, cfg.Stage == "production", true)
			userID = c.ClientIP()
		} else {
			userID, err = validateSignedUserID(userIDCookie.Value, cfg)
			if err != nil {
				c.JSON(http.StatusBadRequest, errors.ApiError{
					Code:    http.StatusBadRequest,
					Error:   enum.InvalidCookie,
					Details: err,
				})
				c.Abort()
				return
			}
		}

		ctx := context.Background()
		cacheKey := fmt.Sprintf("rate-limiter:%s:%s", c.FullPath(), userID)

		// Get current state
		cmds := make(valkey.Commands, 2)
		cmds[0] = valkeyClient.B().Hget().Key(cacheKey).Field("hits").Build()
		cmds[1] = valkeyClient.B().Hget().Key(cacheKey).Field("first_hit").Build()

		results := valkeyClient.DoMulti(ctx, cmds...)
		hits, hitsErr := results[0].AsInt64()
		firstHit, firstHitErr := results[1].AsInt64()

		// Check if entry exists and is still valid
		if hitsErr != nil || firstHitErr != nil || time.Since(time.Unix(firstHit, 0)) > timeframe {
			// Entry doesn't exist, create new
			logger.PrintfDebug("Creating new rate limit entry for %s", cacheKey)

			cmds = make(valkey.Commands, 3)
			cmds[0] = valkeyClient.B().Hset().Key(cacheKey).FieldValue().FieldValue("hits", "1").Build()
			cmds[1] = valkeyClient.B().Hset().Key(cacheKey).FieldValue().FieldValue("first_hit", fmt.Sprintf("%d", time.Now().Unix())).Build()
			cmds[2] = valkeyClient.B().Expire().Key(cacheKey).Seconds(int64(timeframe.Seconds())).Build()

			results = valkeyClient.DoMulti(ctx, cmds...)
			for _, result := range results {
				if err := result.Error(); err != nil {
					c.JSON(http.StatusInternalServerError, errors.ApiError{
						Code:    http.StatusInternalServerError,
						Error:   enum.ApiError,
						Details: err,
					})
					c.Abort()
					return
				}
			}
		} else {
			if hits >= int64(requests) {
				c.JSON(http.StatusTooManyRequests, errors.ApiError{
					Code:  http.StatusTooManyRequests,
					Error: enum.TooManyRequests,
				})
				c.Abort()
				return
			}

			if err := valkeyClient.Do(ctx, valkeyClient.B().Hincrby().Key(cacheKey).Field("hits").Increment(1).Build()).Error(); err != nil {
				c.JSON(http.StatusInternalServerError, errors.ApiError{
					Code:    http.StatusInternalServerError,
					Error:   enum.ApiError,
					Details: err,
				})
				c.Abort()
				return
			}
		}
		c.Next()
	}
}

func signedUserID(cfg *config.Config) (string, error) {
	random := make([]byte, 32)
	if _, err := rand.Read(random); err != nil {
		return "", err
	}

	hash := hmac.New(sha256.New, []byte(cfg.CookieSecret))
	if _, err := hash.Write(random); err != nil {
		return "", err
	}

	return hex.EncodeToString(random) + "." + hex.EncodeToString(hash.Sum(nil)), nil
}

func validateSignedUserID(signedUserID string, cfg *config.Config) (string, error) {
	signedPices := strings.Split(signedUserID, ".")
	userIDString := signedPices[0]
	signature := signedPices[1]

	decoded, err := hex.DecodeString(userIDString)
	if err != nil {
		return "", err
	}

	hash := hmac.New(sha256.New, []byte(cfg.CookieSecret))
	if _, err := hash.Write(decoded); err != nil {
		return "", err
	}

	if hex.EncodeToString(hash.Sum(nil)) != signature {
		return "", e.New("Invalid signed user ID")
	}

	return userIDString, nil
}
