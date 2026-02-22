package main

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
		if err := operation(); err == nil {
			return nil
		} else {
			lastErr = err
		}
		if attempt < cfg.MaxAttempts {
			time.Sleep(delay)
			delay = calculateNextDelay(delay, cfg)
		}
	}
	return fmt.Errorf("operation failed after %d attempts: %w", cfg.MaxAttempts, lastErr)
}

func calculateNextDelay(current time.Duration, cfg RetryConfig) time.Duration {
	next := time.Duration(float64(current) * cfg.Multiplier)
	if next > cfg.MaxDelay {
		return cfg.MaxDelay
	}
	return next
}

// RetryWithContext executes with retry and progress callback
func RetryWithContext(cfg RetryConfig, onRetry func(attempt int, err error), operation func() error) error {
	var lastErr error
	delay := cfg.InitialDelay
	for attempt := 1; attempt <= cfg.MaxAttempts; attempt++ {
		if err := operation(); err == nil {
			return nil
		} else {
			lastErr = err
			handleRetryError(attempt, err, cfg.MaxAttempts, onRetry)
		}
		if attempt < cfg.MaxAttempts {
			time.Sleep(delay)
			delay = calculateNextDelay(delay, cfg)
		}
	}
	return fmt.Errorf("operation failed after %d attempts: %w", cfg.MaxAttempts, lastErr)
}

func handleRetryError(attempt int, err error, maxAttempts int, onRetry func(int, error)) {
	if onRetry != nil && attempt < maxAttempts {
		onRetry(attempt, err)
	}
}
