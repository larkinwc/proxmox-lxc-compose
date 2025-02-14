package recovery

import (
	"context"
	"time"

	"proxmox-lxc-compose/pkg/errors"
	"proxmox-lxc-compose/pkg/logging"
)

// RetryConfig defines retry behavior
type RetryConfig struct {
	MaxAttempts     int
	InitialInterval time.Duration
	MaxInterval     time.Duration
	Multiplier      float64
	MaxElapsedTime  time.Duration
}

// DefaultRetryConfig provides sensible defaults
var DefaultRetryConfig = RetryConfig{
	MaxAttempts:     3,
	InitialInterval: 1 * time.Second,
	MaxInterval:     30 * time.Second,
	Multiplier:      2.0,
	MaxElapsedTime:  5 * time.Minute,
}

// IsRetryable checks if an error should be retried
func IsRetryable(err error) bool {
	if err == nil {
		return false
	}

	// Add specific error types that should be retried
	switch {
	case errors.IsType(err, errors.ErrNetwork):
		return true
	case errors.IsType(err, errors.ErrRegistry):
		return true
	case errors.IsType(err, errors.ErrSystem):
		return true
	default:
		return false
	}
}

// RetryWithBackoff executes the given operation with exponential backoff
func RetryWithBackoff(ctx context.Context, config RetryConfig, op func() error) error {
	var lastErr error
	interval := config.InitialInterval
	startTime := time.Now()

	for attempt := 1; attempt <= config.MaxAttempts; attempt++ {
		// Check if we've exceeded max elapsed time
		if config.MaxElapsedTime > 0 && time.Since(startTime) > config.MaxElapsedTime {
			if lastErr != nil {
				return errors.Wrap(lastErr, errors.ErrSystem, "max retry duration exceeded")
			}
			return errors.New(errors.ErrSystem, "max retry duration exceeded")
		}

		// Check context cancellation
		select {
		case <-ctx.Done():
			if lastErr != nil {
				return errors.Wrap(lastErr, errors.ErrSystem, "operation cancelled")
			}
			return errors.New(errors.ErrSystem, "operation cancelled")
		default:
		}

		// Execute the operation
		err := op()
		if err == nil {
			return nil
		}

		lastErr = err
		if !IsRetryable(err) {
			return err
		}

		// If this is the last attempt, return the error
		if attempt == config.MaxAttempts {
			return errors.Wrap(err, errors.ErrSystem, "max retry attempts exceeded")
		}

		// Log retry attempt
		logging.Warn("Operation failed, retrying",
			"attempt", attempt,
			"maxAttempts", config.MaxAttempts,
			"interval", interval.String(),
			"error", err)

		// Wait before next attempt
		select {
		case <-ctx.Done():
			return errors.Wrap(err, errors.ErrSystem, "operation cancelled during retry delay")
		case <-time.After(interval):
		}

		// Update interval for next attempt
		interval = time.Duration(float64(interval) * config.Multiplier)
		if interval > config.MaxInterval {
			interval = config.MaxInterval
		}
	}

	return lastErr
}