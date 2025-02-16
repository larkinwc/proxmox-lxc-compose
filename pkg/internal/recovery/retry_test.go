package recovery

import (
	"context"
	"testing"
	"time"

	"proxmox-lxc-compose/pkg/errors"
	"proxmox-lxc-compose/pkg/logging"
)

func init() {
	// Initialize logger for tests
	if err := logging.Init(logging.Config{
		Level:       "debug", // Set to debug to test logging
		Development: true,
	}); err != nil {
		panic("Failed to initialize logger for tests: " + err.Error())
	}
}

type temporaryError struct {
	attemptsUntilSuccess int
	currentAttempt       int
}

func (e *temporaryError) Error() string {
	return "temporary error"
}

// Make temporaryError implement our error type system
func (e *temporaryError) Type() errors.ErrorType {
	return errors.ErrNetwork
}

func (e *temporaryError) IsTemporary() bool {
	e.currentAttempt++
	return e.currentAttempt < e.attemptsUntilSuccess
}

func TestRetryWithBackoff(t *testing.T) {
	tests := []struct {
		name          string
		attempts      int
		shouldSucceed bool
	}{
		{
			name:          "succeeds_after_retries",
			attempts:      2,
			shouldSucceed: true,
		},
		{
			name:          "fails_after_max_retries",
			attempts:      5,
			shouldSucceed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tempErr := &temporaryError{attemptsUntilSuccess: tt.attempts}
			ctx := context.Background()
			cfg := RetryConfig{
				MaxAttempts:     3, // 3 retries max
				InitialInterval: time.Millisecond,
				MaxInterval:     time.Millisecond * 10,
				MaxElapsedTime:  time.Second,
			}

			var attempts int
			result := RetryWithBackoff(ctx, cfg, func() error {
				attempts++
				if attempts >= tt.attempts {
					return nil
				}
				return tempErr
			})

			if tt.shouldSucceed && result != nil {
				t.Errorf("expected success after %d attempts, got error: %v", tt.attempts, result)
			}
			if !tt.shouldSucceed && result == nil {
				t.Errorf("expected failure after %d attempts, got success", tt.attempts)
			}
		})
	}
}

func TestRetryWithBackoffTimeout(t *testing.T) {
	tests := []struct {
		name          string
		attempts      int
		timeout       time.Duration
		shouldTimeout bool
	}{
		{
			name:          "succeeds_before_timeout",
			attempts:      2,
			timeout:       time.Second,
			shouldTimeout: false,
		},
		{
			name:          "context_timeout",
			attempts:      5,
			timeout:       time.Millisecond * 10,
			shouldTimeout: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), tt.timeout)
			defer cancel()

			cfg := RetryConfig{
				MaxAttempts:     3,
				InitialInterval: time.Millisecond * 20,
				MaxInterval:     time.Millisecond * 50,
				MaxElapsedTime:  time.Second,
			}

			var attempts int
			result := RetryWithBackoff(ctx, cfg, func() error {
				attempts++
				if attempts >= tt.attempts {
					return nil
				}
				time.Sleep(time.Millisecond * 20) // Simulate work
				return &temporaryError{attemptsUntilSuccess: tt.attempts}
			})

			if tt.shouldTimeout {
				if ctx.Err() != context.DeadlineExceeded {
					t.Errorf("expected context.DeadlineExceeded error, got: %v", result)
				}
			} else if result != nil {
				t.Errorf("expected success, got error: %v", result)
			}
		})
	}
}
