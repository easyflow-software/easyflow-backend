package middleware

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"easyflow-backend/pkg/api/errors"
	"easyflow-backend/pkg/config"
	"easyflow-backend/pkg/enum"
	"encoding/hex"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"golang.org/x/time/rate"
)

// Stores rate limiting data for a single user
type UserLimit struct {
	Limiter     *rate.Limiter
	LastRequest time.Time
}

// Creates a rate limiting middleware that restricts requests per user.
// Uses signed cookies to identify users and applies rate limiting based on the provided limit and burst parameters.
func NewRateLimiter(limit float64, burst int) gin.HandlerFunc {
	userLimitMap := make(map[string]UserLimit)
	var userLimitMapMutex sync.RWMutex

	// Start cleanup goroutine
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

// Retrieves or creates a rate limiter for the given user ID
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
		limiter.LastRequest = time.Now()
		userLimitMap[userID] = limiter
	}
	return limiter
}

// Removes rate limiters that haven't been used in the last minute
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

// Creates a random hex string to use as a user identifier
func generateUniqueID() string {
	bytes := make([]byte, 16)
	_, err := rand.Read(bytes)
	if err != nil {
		return "unknown"
	}
	return hex.EncodeToString(bytes)
}

// Creates an HMAC signature using the app's cookie secret
func signCookie(data string, cfg *config.Config) string {
	h := hmac.New(sha256.New, []byte(cfg.CookieSecret))
	h.Write([]byte(data))
	return data + "." + hex.EncodeToString(h.Sum(nil))
}

// Validates the HMAC signature and returns the original data
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
