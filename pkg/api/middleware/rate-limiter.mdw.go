package middleware

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"easyflow-backend/pkg/api/errors"
	"easyflow-backend/pkg/config"
	"easyflow-backend/pkg/enum"
	"easyflow-backend/pkg/logger"
	"encoding/hex"
	"encoding/json"
	e "errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/valkey-io/valkey-go"
)

type rateLimiter struct {
	FirstHit time.Time `json:"first_hit"`
	Hits     int       `json:"hits"`
}

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
				Details: "Valkey is not of type *valkey.Client",
			})
			c.Abort()
			return
		}

		var userID string
		var cacheKey string

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

		cacheKey = fmt.Sprintf("rate-limiter:%s:%s", c.FullPath(), userID)

		limit, err := valkeyClient.Do(context.Background(), valkeyClient.B().Get().Key(cacheKey).Build()).ToString()
		if err != nil {
			logger.PrintfDebug("No rate limiting found for %s. Creating new entry", cacheKey)
			rateLimit := rateLimiter{
				FirstHit: time.Now(),
				Hits:     1,
			}
			rateLimitBytes, err := json.Marshal(rateLimit)
			if err != nil {
				c.JSON(http.StatusInternalServerError, errors.ApiError{
					Code:    http.StatusInternalServerError,
					Error:   enum.ApiError,
					Details: err,
				})
				c.Abort()
				return
			}

			rateLimitString := string(rateLimitBytes)

			if err := valkeyClient.Do(context.Background(), valkeyClient.B().Set().Key(cacheKey).Value(rateLimitString).Ex(timeframe+1*time.Minute).Build()).Error(); err != nil {
				c.JSON(http.StatusInternalServerError, errors.ApiError{
					Code:    http.StatusInternalServerError,
					Error:   enum.ApiError,
					Details: err,
				})
				c.Abort()
				return
			}
		} else {
			var rateLimiter rateLimiter
			err := json.Unmarshal([]byte(limit), &rateLimiter)
			if err != nil {
				c.JSON(http.StatusInternalServerError, errors.ApiError{
					Code:    http.StatusInternalServerError,
					Error:   enum.ApiError,
					Details: err,
				})
				c.Abort()
				return
			}

			// Check if time window has expired
			if time.Since(rateLimiter.FirstHit) > timeframe {
				// Reset the counter if the time window has expired
				rateLimiter.FirstHit = time.Now()
				rateLimiter.Hits = 1
			} else {
				// Check if limit is exceeded within the current window
				if rateLimiter.Hits >= requests {
					c.JSON(http.StatusTooManyRequests, errors.ApiError{
						Code:  http.StatusTooManyRequests,
						Error: enum.TooManyRequests,
					})
					c.Abort()
					return
				}
				// Increment hits if within limits
				rateLimiter.Hits++
			}

			rateLimiterBytes, err := json.Marshal(rateLimiter)
			if err != nil {
				c.JSON(http.StatusInternalServerError, errors.ApiError{
					Code:    http.StatusInternalServerError,
					Error:   enum.ApiError,
					Details: err,
				})
				c.Abort()
				return
			}

			rateLimiterString := string(rateLimiterBytes)

			if err := valkeyClient.Do(context.Background(), valkeyClient.B().Set().Key(cacheKey).Value(rateLimiterString).Ex(timeframe+1*time.Minute).Build()).Error(); err != nil {
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
	_, err := rand.Read(random)
	if err != nil {
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
