package errors

import (
	"fmt"
	"testing"
)

func TestSafeExecute(t *testing.T) {
	collector := NewErrorCollector(10, false)

	// Test successful execution
	err := SafeExecute(collector, func() error {
		return nil
	}, "test operation")

	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	// Test error handling
	expectedErr := fmt.Errorf("test error")
	err = SafeExecute(collector, func() error {
		return expectedErr
	}, "test operation")

	if err != expectedErr {
		t.Errorf("Expected %v, got: %v", expectedErr, err)
	}

	// Use the err variable to avoid unused variable error
	_ = err

	// Check that error was collected
	if !collector.HasErrors() {
		t.Error("Expected error to be collected")
	}
}

func TestPanicRecovery(t *testing.T) {
	collector := NewErrorCollector(10, false)

	// Test panic recovery
	_ = SafeExecute(collector, func() error {
		panic("test panic")
	}, "test operation")

	// Should not panic, but should collect the error
	if !collector.HasErrors() {
		t.Error("Expected panic to be converted to error")
	}

	errors := collector.GetErrors()
	if len(errors) == 0 {
		t.Fatal("Expected at least one error")
	}

	panicErr := errors[0]
	if panicErr.Severity != SeverityFatal {
		t.Errorf("Expected fatal severity, got: %v", panicErr.Severity)
	}

	if panicErr.Category != CategoryInternal {
		t.Errorf("Expected internal category, got: %v", panicErr.Category)
	}
}

func TestProcessWithPartialFailure(t *testing.T) {
	collector := NewErrorCollector(10, false)

	items := []int{1, 2, 3, 4, 5}
	
	result := ProcessWithPartialFailure(
		items,
		func(item int) error {
			if item == 3 {
				return fmt.Errorf("error processing item %d", item)
			}
			if item == 4 {
				panic(fmt.Sprintf("panic on item %d", item))
			}
			return nil
		},
		collector,
		"test processing",
	)

	// Should have processed all items
	if result.SuccessCount != 3 { // items 1, 2, 5
		t.Errorf("Expected 3 successes, got: %d", result.SuccessCount)
	}

	if result.FailureCount != 2 { // items 3, 4
		t.Errorf("Expected 2 failures, got: %d", result.FailureCount)
	}

	if result.IsComplete {
		t.Error("Expected incomplete result due to errors")
	}

	if len(result.Errors) != 2 {
		t.Errorf("Expected 2 errors in result, got: %d", len(result.Errors))
	}
}

func TestRetryWithRecovery(t *testing.T) {
	collector := NewErrorCollector(10, false)
	
	attempts := 0
	err := RetryWithRecovery(
		func() error {
			attempts++
			if attempts < 3 {
				return fmt.Errorf("attempt %d failed", attempts)
			}
			return nil
		},
		ErrorRecoveryOptions{
			MaxRetries:         3,
			ContinueOnError:    true,
			RecordPartialError: true,
		},
		collector,
		"test retry",
	)

	if err != nil {
		t.Errorf("Expected success after retries, got: %v", err)
	}

	if attempts != 3 {
		t.Errorf("Expected 3 attempts, got: %d", attempts)
	}

	// Should have warning messages for retries
	warnings := collector.GetWarnings()
	if len(warnings) != 2 { // 2 retry warnings
		t.Errorf("Expected 2 retry warnings, got: %d", len(warnings))
	}
}

func TestRetryWithRecoveryAllFailed(t *testing.T) {
	collector := NewErrorCollector(10, false)
	
	attempts := 0
	err := RetryWithRecovery(
		func() error {
			attempts++
			return fmt.Errorf("attempt %d failed", attempts)
		},
		ErrorRecoveryOptions{
			MaxRetries:         2,
			ContinueOnError:    true,
			RecordPartialError: true,
		},
		collector,
		"test retry all failed",
	)

	if err == nil {
		t.Error("Expected error after all retries failed")
	}

	if attempts != 3 { // initial + 2 retries
		t.Errorf("Expected 3 attempts, got: %d", attempts)
	}

	// Should have collected final error
	errors := collector.GetErrors()
	if len(errors) == 0 {
		t.Error("Expected final error to be collected")
	}
}

func TestCircuitBreaker(t *testing.T) {
	collector := NewErrorCollector(10, false)
	cb := NewCircuitBreaker(2, 100) // 2 failures, 100ms timeout

	// First failure
	err := cb.Execute(func() error {
		return fmt.Errorf("first failure")
	}, collector, "test")

	if err == nil {
		t.Error("Expected first failure")
	}

	// Second failure - should open circuit
	err = cb.Execute(func() error {
		return fmt.Errorf("second failure")
	}, collector, "test")

	if err == nil {
		t.Error("Expected second failure")
	}

	// Third call - should fail fast due to open circuit
	err = cb.Execute(func() error {
		return nil // This shouldn't be called
	}, collector, "test")

	if err == nil {
		t.Error("Expected circuit breaker to fail fast")
	}

	// Verify the error is from circuit breaker
	if !collector.HasErrors() {
		t.Error("Expected circuit breaker error to be collected")
	}
}