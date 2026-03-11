package retry

import (
	"fmt"
	"time"
)

// RetryConfig defines retry behavior
type RetryConfig struct {
	MaxAttempts  int
	InitialDelay time.Duration
	MaxDelay     time.Duration
	Multiplier   float64
}

// DefaultRetryConfig returns sensible defaults
func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 1 * time.Second,
		MaxDelay:     30 * time.Second,
		Multiplier:   2.0,
	}
}

// RetryOperation executes an operation with exponential backoff
func RetryOperation(cfg RetryConfig, operation func() error) error {
	var lastErr error
	delay := cfg.InitialDelay

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		err := operation()
		if err == nil {
			return nil
		}

		lastErr = err

		// Don't wait after last attempt
		if attempt == cfg.MaxAttempts {
			break
		}

		// Wait before retry with exponential backoff
		time.Sleep(delay)
		delay = time.Duration(float64(delay) * cfg.Multiplier)
		if delay > cfg.MaxDelay {
			delay = cfg.MaxDelay
		}
	}

	return fmt.Errorf("operation failed after %d attempts: %w", cfg.MaxAttempts, lastErr)
}

// RetryWithContext executes with retry and progress callback
func RetryWithContext(cfg RetryConfig, onRetry func(attempt int, err error), operation func() error) error {
	var lastErr error
	delay := cfg.InitialDelay

	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		err := operation()
		if err == nil {
			return nil
		}

		lastErr = err

		// Notify about retry
		if onRetry != nil && attempt < cfg.MaxAttempts {
			onRetry(attempt, err)
		}

		// Don't wait after last attempt
		if attempt == cfg.MaxAttempts {
			break
		}

		// Wait before retry with exponential backoff
		time.Sleep(delay)
		delay = time.Duration(float64(delay) * cfg.Multiplier)
		if delay > cfg.MaxDelay {
			delay = cfg.MaxDelay
		}
	}

	return fmt.Errorf("operation failed after %d attempts: %w", cfg.MaxAttempts, lastErr)
}
