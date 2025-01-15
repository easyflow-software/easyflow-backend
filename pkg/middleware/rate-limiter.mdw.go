package middleware

import (
	"sync"
	"time"

	"github.com/gin-gonic/gin"

	"golang.org/x/time/rate"
)

var userLimiterMap = make(map[string]*rate.Limiter)
var userLimiterMapMutex sync.Mutex

// returns the rate limiter for the client IP address.
func getUserLimiter(clientIPAddress string, limit float64, burst int) *rate.Limiter {
	userLimiterMapMutex.Lock()
	defer userLimiterMapMutex.Unlock()

	limiter, ok := userLimiterMap[clientIPAddress]
	if !ok {
		limiter = rate.NewLimiter(rate.Limit(limit), burst)
		userLimiterMap[clientIPAddress] = limiter
	}
	return limiter
}

// RateLimiter is a middleware that limits the number of requests a client can make
func RateLimiter(limit float64, burst int) gin.HandlerFunc {
	return func(c *gin.Context) {
		clientIPAddress := c.ClientIP()

		limiter := getUserLimiter(clientIPAddress, limit, burst)
		if limiter.Allow() {
			c.Next()
		} else {
			time.Sleep(time.Duration(1/limit) * time.Second)
			c.Next()
		}

	}
}
