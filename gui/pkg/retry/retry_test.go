package retry

import (
	"errors"
	"testing"
	"time"
)

func TestRetryOperation_Success(t *testing.T) {
	cfg := RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2.0,
	}

	attempts := 0
	operation := func() error {
		attempts++
		return nil
	}

	err := RetryOperation(cfg, operation)
	if err != nil {
		t.Errorf("Expected success, got error: %v", err)
	}

	if attempts != 1 {
		t.Errorf("Expected 1 attempt, got %d", attempts)
	}
}

func TestRetryOperation_SuccessAfterRetries(t *testing.T) {
	cfg := RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2.0,
	}

	attempts := 0
	operation := func() error {
		attempts++
		if attempts < 3 {
			return errors.New("temporary failure")
		}
		return nil
	}

	err := RetryOperation(cfg, operation)
	if err != nil {
		t.Errorf("Expected success after retries, got error: %v", err)
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}
}

func TestRetryOperation_ExhaustsAttempts(t *testing.T) {
	cfg := RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2.0,
	}

	attempts := 0
	expectedErr := errors.New("persistent failure")
	operation := func() error {
		attempts++
		return expectedErr
	}

	err := RetryOperation(cfg, operation)
	if err == nil {
		t.Error("Expected error after exhausting attempts, got nil")
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got %d", attempts)
	}

	if !errors.Is(err, expectedErr) {
		t.Errorf("Expected wrapped error containing original, got: %v", err)
	}
}

func TestRetryOperation_ExponentialBackoff(t *testing.T) {
	cfg := RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 50 * time.Millisecond,
		MaxDelay:     500 * time.Millisecond,
		Multiplier:   2.0,
	}

	attempts := 0
	timestamps := []time.Time{}
	operation := func() error {
		attempts++
		timestamps = append(timestamps, time.Now())
		return errors.New("fail")
	}

	start := time.Now()
	_ = RetryOperation(cfg, operation)
	elapsed := time.Since(start)

	// Should have 3 attempts with delays: 50ms + 100ms = 150ms minimum
	if elapsed < 150*time.Millisecond {
		t.Errorf("Expected at least 150ms delay, got %v", elapsed)
	}

	// Check exponential growth
	if len(timestamps) >= 3 {
		delay1 := timestamps[1].Sub(timestamps[0])
		delay2 := timestamps[2].Sub(timestamps[1])

		if delay2 <= delay1 {
			t.Errorf("Expected exponential backoff, delay1=%v delay2=%v", delay1, delay2)
		}
	}
}

func TestRetryOperation_MaxDelayRespected(t *testing.T) {
	cfg := RetryConfig{
		MaxAttempts:  4,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     30 * time.Millisecond,
		Multiplier:   10.0, // Large multiplier to test capping
	}

	attempts := 0
	timestamps := []time.Time{}
	operation := func() error {
		attempts++
		timestamps = append(timestamps, time.Now())
		return errors.New("fail")
	}

	_ = RetryOperation(cfg, operation)

	// Check that delays are capped at maxDelay
	for i := 2; i < len(timestamps); i++ {
		delay := timestamps[i].Sub(timestamps[i-1])
		if delay > 50*time.Millisecond { // Give some buffer
			t.Errorf("Delay exceeded max: %v", delay)
		}
	}
}

func TestRetryWithContext_CallsCallback(t *testing.T) {
	cfg := RetryConfig{
		MaxAttempts:  3,
		InitialDelay: 10 * time.Millisecond,
		MaxDelay:     100 * time.Millisecond,
		Multiplier:   2.0,
	}

	retryCallbacks := 0
	var lastAttempt int
	var lastErr error

	onRetry := func(attempt int, err error) {
		retryCallbacks++
		lastAttempt = attempt
		lastErr = err
	}

	attempts := 0
	expectedErr := errors.New("test error")
	operation := func() error {
		attempts++
		return expectedErr
	}

	_ = RetryWithContext(cfg, onRetry, operation)

	// Should have 2 retry callbacks (not called on last attempt)
	if retryCallbacks != 2 {
		t.Errorf("Expected 2 retry callbacks, got %d", retryCallbacks)
	}

	if lastAttempt != 2 {
		t.Errorf("Expected last attempt to be 2, got %d", lastAttempt)
	}

	if !errors.Is(lastErr, expectedErr) {
		t.Errorf("Expected last error to be test error, got %v", lastErr)
	}
}

func TestDefaultRetryConfig(t *testing.T) {
	cfg := DefaultRetryConfig()

	if cfg.MaxAttempts != 3 {
		t.Errorf("Expected max attempts 3, got %d", cfg.MaxAttempts)
	}

	if cfg.InitialDelay != 1*time.Second {
		t.Errorf("Expected initial delay 1s, got %v", cfg.InitialDelay)
	}

	if cfg.MaxDelay != 30*time.Second {
		t.Errorf("Expected max delay 30s, got %v", cfg.MaxDelay)
	}

	if cfg.Multiplier != 2.0 {
		t.Errorf("Expected multiplier 2.0, got %f", cfg.Multiplier)
	}
}
