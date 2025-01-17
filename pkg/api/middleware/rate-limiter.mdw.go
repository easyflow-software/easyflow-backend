package middleware

import (
	"easyflow-backend/pkg/api/errors"
	"easyflow-backend/pkg/config"
	"easyflow-backend/pkg/enum"

	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

type UserLimit struct {
	Limiter     *rate.Limiter
	LastRequest time.Time
}

func NewRateLimiter(limit float64, burst int) gin.HandlerFunc {
	userLimitMap := make(map[string]UserLimit)
	var userLimitMapMutex sync.RWMutex

	// Cleanup function to remove old entries from the map
	go func() {
		for {
			time.Sleep(time.Minute)
			cleanupOldEntries(userLimitMap, &userLimitMapMutex)
		}
	}()

	return func(c *gin.Context) {
		rawCfg, ok := c.Get("config")
		if !ok {
			c.JSON(500, errors.ApiError{
				Code:    http.StatusInternalServerError,
				Error:   enum.ApiError,
				Details: "Config not found in context",
			})
			c.Abort()
			return
		}

		cfg, ok := rawCfg.(*config.Config)
		if !ok {
			c.JSON(500, errors.ApiError{
				Code:    http.StatusInternalServerError,
				Error:   enum.ApiError,
				Details: "Config could not be cast to *common.Config",
			})
			c.Abort()
			return
		}

		cookieName := "user_id"
		userID, err := c.Cookie(cookieName)
		if err != nil || userID == "" {
			// Generate a new user ID and set the cookie with a signature
			userID = generateUniqueID()
			signedUserID := signCookie(userID, cfg)
			c.SetCookie(cookieName, signedUserID, 3600, "/", "", false, true)
			// sleep to keep the rate limiter from being bypassed
			time.Sleep(time.Duration(1/limit) * time.Second)
		} else {
			// Verify the cookie signature
			userID, err = verifyCookie(userID, cfg)
			if err != nil {
				// If verification fails, generate a new user ID
				userID = generateUniqueID()
				signedUserID := signCookie(userID, cfg)
				c.SetCookie(cookieName, signedUserID, 3600, "/", "", false, true)
				// sleep to keep the rate limiter from being bypassed
				time.Sleep(time.Duration(1/limit) * time.Second)
			}
		}

		userLimit := getUserLimiter(userID, limit, burst, userLimitMap, &userLimitMapMutex)

		if userLimit.Limiter.Allow() {
			c.Next()
		} else {
			time.Sleep(time.Duration(1/limit) * time.Second)
			c.Next()
		}
	}
}

func getUserLimiter(userID string, limit float64, burst int, userLimitMap map[string]UserLimit, mutex *sync.RWMutex) UserLimit {
	mutex.Lock()
	defer mutex.Unlock()

	limiter, exists := userLimitMap[userID]
	if !exists {
		limiter = UserLimit{
			Limiter:     rate.NewLimiter(rate.Limit(limit), burst),
			LastRequest: time.Now(),
		}
		userLimitMap[userID] = limiter
	} else {
		// Update the last request time
		limiter.LastRequest = time.Now()
		userLimitMap[userID] = limiter
	}

	return limiter
}

func cleanupOldEntries(userLimitMap map[string]UserLimit, mutex *sync.RWMutex) {
	mutex.Lock()
	defer mutex.Unlock()

	cutoff := time.Now().Add(-time.Minute)
	for userID, limiter := range userLimitMap {
		if limiter.LastRequest.Before(cutoff) {
			delete(userLimitMap, userID)
		}
	}
}

func generateUniqueID() string {
	bytes := make([]byte, 16)
	_, err := rand.Read(bytes)
	if err != nil {
		return "unknown"
	}
	return hex.EncodeToString(bytes)
}

func signCookie(data string, cfg *config.Config) string {
	h := hmac.New(sha256.New, []byte(cfg.CookieSecret))
	h.Write([]byte(data))
	return data + "." + hex.EncodeToString(h.Sum(nil))
}

func verifyCookie(signedData string, cfg *config.Config) (string, error) {
	parts := strings.Split(signedData, ".")
	if len(parts) != 2 {
		return "", fmt.Errorf("invalid cookie format")
	}

	data, signature := parts[0], parts[1]
	h := hmac.New(sha256.New, []byte(cfg.CookieSecret))
	h.Write([]byte(data))

	expectedSignature := hex.EncodeToString(h.Sum(nil))
	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return "", fmt.Errorf("invalid signature")
	}

	return data, nil
}
