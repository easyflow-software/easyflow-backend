package retry

import (
	"easyflow-backend/pkg/logger"
	"reflect"
	"runtime"
	"time"
)

// RetryConfig holds the configuration for retry behavior
type RetryConfig struct {
	// MaxAttempts is the maximum number of attempts before giving up
	MaxAttempts int
	// Delay is the initial delay between attempts
	Delay time.Duration
	// MaxDelay is the maximum delay between attempts
	MaxDelay time.Duration
	// Multiplier is the factor by which the delay is multiplied between attempts
	Multiplier float64
	// RetryableErr is a function that determines if an error is retryable
	RetryableErr func(error) bool
}

// DefaultRetryConfig provides sensible default values
func DefaultRetryConfig() *RetryConfig {
	return &RetryConfig{
		MaxAttempts: 5,
		Delay:       time.Second,
		MaxDelay:    time.Second * 30,
		Multiplier:  2.0,
		RetryableErr: func(err error) bool {
			return true // By default, retry all errors
		},
	}
}

// WithRetry wraps a function with retry logic
func WithRetry[T any](fn func() (T, error), logger *logger.Logger, config *RetryConfig) func() (T, error) {
	if config == nil {
		config = DefaultRetryConfig()
	}

	return func() (T, error) {
		var lastErr error
		currentDelay := config.Delay
		functionName := runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()

		for attempt := 0; attempt < config.MaxAttempts; attempt++ {
			result, err := fn()
			if err == nil {
				return result, nil
			}

			lastErr = err
			if !config.RetryableErr(err) {
				var zero T
				return zero, err
			}

			if attempt < config.MaxAttempts-1 {
				time.Sleep(currentDelay)
				currentDelay = time.Duration(float64(currentDelay) * config.Multiplier)
				if currentDelay > config.MaxDelay {
					currentDelay = config.MaxDelay
				}
				logger.PrintfWarning("Failed to complete function %s successfully retring again in %f. Attempt %d", functionName, currentDelay.Seconds(), attempt)
			}
		}

		var zero T
		logger.PrintfError("Reached max retry attempts for func: %s", functionName)
		return zero, lastErr
	}
}
