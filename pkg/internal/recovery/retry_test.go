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
		Level:       "error", // Set to error to minimize test output
		Development: false,
	}); err != nil {
		panic("Failed to initialize logger for tests: " + err.Error())
	}
}

type temporaryError struct {
	attemptsUntilSuccess int
	currentAttempt      int
}

func (e *temporaryError) Error() string {
	return "temporary error"
}

func TestRetryWithBackoff(t *testing.T) {
	tests := []struct {
		name          string
		config        RetryConfig
		error         error
		shouldSucceed bool
	}{
		{
			name: "succeeds after retries",
			config: RetryConfig{
				MaxAttempts:     3,
				InitialInterval: 10 * time.Millisecond,
				MaxInterval:     100 * time.Millisecond,
				Multiplier:      2.0,
			},
			error: &temporaryError{
				attemptsUntilSuccess: 2,
			},
			shouldSucceed: true,
		},
		{
			name: "fails after max attempts",
			config: RetryConfig{
				MaxAttempts:     2,
				InitialInterval: 10 * time.Millisecond,
				MaxInterval:     100 * time.Millisecond,
				Multiplier:      2.0,
			},
			error: &temporaryError{
				attemptsUntilSuccess: 3,
			},
			shouldSucceed: false,
		},
		{
			name: "non-retryable error fails immediately",
			config: RetryConfig{
				MaxAttempts:     3,
				InitialInterval: 10 * time.Millisecond,
				MaxInterval:     100 * time.Millisecond,
				Multiplier:      2.0,
			},
			error:         errors.New(errors.ErrValidation, "validation error"),
			shouldSucceed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			var tempErr *temporaryError
			if e, ok := tt.error.(*temporaryError); ok {
				tempErr = e
			}

			operation := func() error {
				if tempErr != nil {
					tempErr.currentAttempt++
					if tempErr.currentAttempt >= tempErr.attemptsUntilSuccess {
						return nil
					}
					return errors.Wrap(tempErr, errors.ErrNetwork, "temporary network error")
				}
				return tt.error
			}

			err := RetryWithBackoff(ctx, tt.config, operation)
			if tt.shouldSucceed && err != nil {
				t.Errorf("expected success, got error: %v", err)
			}
			if !tt.shouldSucceed && err == nil {
				t.Error("expected error, got success")
			}
		})
	}
}

func TestRetryWithBackoffContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	
	config := RetryConfig{
		MaxAttempts:     3,
		InitialInterval: 100 * time.Millisecond,
		MaxInterval:     1 * time.Second,
		Multiplier:      2.0,
	}

	operationCalls := 0
	operation := func() error {
		operationCalls++
		return errors.New(errors.ErrNetwork, "network error")
	}

	// Cancel after first attempt
	go func() {
		time.Sleep(50 * time.Millisecond)
		cancel()
	}()

	err := RetryWithBackoff(ctx, config, operation)
	if err == nil {
		t.Error("expected error due to context cancellation")
	}
	if operationCalls > 1 {
		t.Errorf("expected 1 operation call, got %d", operationCalls)
	}
}