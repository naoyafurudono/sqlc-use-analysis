package errors

import (
	"fmt"
	"runtime"
	"time"
)

// RecoverWithError recovers from panic and converts it to an AnalysisError
func RecoverWithError(collector *ErrorCollector) {
	if r := recover(); r != nil {
		stack := make([]byte, 4096)
		n := runtime.Stack(stack, false)

		err := NewError(CategoryInternal, SeverityFatal,
			fmt.Sprintf("panic recovered: %v", r))
		err.StackTrace = string(stack[:n])

		collector.Add(err)
	}
}

// SafeExecute safely executes a function with panic recovery
func SafeExecute(collector *ErrorCollector, fn func() error, context string) error {
	defer RecoverWithError(collector)

	if err := fn(); err != nil {
		// ラップして詳細な情報を追加
		wrappedErr := Wrap(err, fmt.Sprintf("error in %s", context))
		if collectErr := collector.Add(wrappedErr); collectErr != nil {
			return collectErr
		}
		return err
	}

	return nil
}

// PartialResult represents the result of an operation that may have partial failures
type PartialResult struct {
	Result     interface{}      `json:"result"`
	Errors     []*AnalysisError `json:"errors"`
	IsComplete bool             `json:"is_complete"`
	SuccessCount int            `json:"success_count"`
	FailureCount int            `json:"failure_count"`
}

// ProcessWithPartialFailure processes a slice of items, continuing even if some fail
func ProcessWithPartialFailure[T any](
	items []T,
	processor func(T) error,
	collector *ErrorCollector,
	context string,
) *PartialResult {
	result := &PartialResult{
		Result:     make([]interface{}, 0),
		Errors:     make([]*AnalysisError, 0),
		IsComplete: true,
	}

	for i, item := range items {
		func() {
			defer func() {
				if r := recover(); r != nil {
					stack := make([]byte, 2048)
					n := runtime.Stack(stack, false)

					panicErr := NewError(CategoryInternal, SeverityError,
						fmt.Sprintf("panic in %s processing item %d: %v", context, i, r))
					panicErr.StackTrace = string(stack[:n])
					panicErr.Details["item_index"] = i
					panicErr.Details["context"] = context

					result.Errors = append(result.Errors, panicErr)
					result.FailureCount++
					result.IsComplete = false

					if collector != nil {
						collector.Add(panicErr)
					}
				}
			}()

			if err := processor(item); err != nil {
				if ae, ok := err.(*AnalysisError); ok {
					result.Errors = append(result.Errors, ae)
					if ae.Severity == SeverityFatal {
						result.IsComplete = false
					}
				} else {
					// 通常のエラーをAnalysisErrorにラップ
					wrappedErr := Wrap(err, fmt.Sprintf("error processing item %d in %s", i, context))
					wrappedErr.Details["item_index"] = i
					wrappedErr.Details["context"] = context
					result.Errors = append(result.Errors, wrappedErr)
				}
				result.FailureCount++

				if collector != nil {
					if ae, ok := err.(*AnalysisError); ok {
						collector.Add(ae)
					} else {
						collector.Add(Wrap(err, fmt.Sprintf("error processing item %d", i)))
					}
				}
			} else {
				result.SuccessCount++
			}
		}()
	}

	return result
}

// ErrorRecoveryOptions defines options for error recovery
type ErrorRecoveryOptions struct {
	MaxRetries         int  // 最大リトライ回数
	ContinueOnError    bool // エラー時も処理を継続するか
	RecordPartialError bool // 部分的なエラーも記録するか
}

// DefaultRecoveryOptions returns default error recovery options
func DefaultRecoveryOptions() ErrorRecoveryOptions {
	return ErrorRecoveryOptions{
		MaxRetries:         3,
		ContinueOnError:    true,
		RecordPartialError: true,
	}
}

// RetryWithRecovery retries a function with error recovery
func RetryWithRecovery(
	fn func() error,
	options ErrorRecoveryOptions,
	collector *ErrorCollector,
	context string,
) error {
	var lastErr error

	for attempt := 0; attempt <= options.MaxRetries; attempt++ {
		err := SafeExecute(collector, fn, fmt.Sprintf("%s (attempt %d)", context, attempt+1))
		if err == nil {
			return nil // 成功
		}

		lastErr = err

		// 最後の試行でない場合は継続
		if attempt < options.MaxRetries {
			if options.RecordPartialError {
				retryErr := NewError(CategoryInternal, SeverityWarning,
					fmt.Sprintf("retrying %s after error (attempt %d/%d): %v",
						context, attempt+1, options.MaxRetries+1, err))
				retryErr.Details["attempt"] = attempt + 1
				retryErr.Details["max_retries"] = options.MaxRetries + 1
				retryErr.Details["context"] = context
				retryErr.Details["original_error"] = err.Error()
				
				if collector != nil {
					collector.Add(retryErr)
				}
			}
		}
	}

	// 全ての試行が失敗した場合
	finalErr := NewError(CategoryInternal, SeverityError,
		fmt.Sprintf("all retry attempts failed for %s: %v", context, lastErr))
	finalErr.Details["max_retries"] = options.MaxRetries + 1
	finalErr.Details["context"] = context
	finalErr.Details["final_error"] = lastErr.Error()
	finalErr.Wrapped = lastErr

	if collector != nil {
		collector.Add(finalErr)
	}

	return finalErr
}

// CircuitBreaker implements a simple circuit breaker pattern for error recovery
type CircuitBreaker struct {
	failureThreshold int
	resetTimeout     int64 // nanoseconds
	failureCount     int
	lastFailureTime  int64
	state            CircuitState
}

// CircuitState represents the state of a circuit breaker
type CircuitState int

const (
	CircuitClosed CircuitState = iota
	CircuitOpen
	CircuitHalfOpen
)

// NewCircuitBreaker creates a new circuit breaker
func NewCircuitBreaker(failureThreshold int, resetTimeoutMs int64) *CircuitBreaker {
	return &CircuitBreaker{
		failureThreshold: failureThreshold,
		resetTimeout:     resetTimeoutMs * 1_000_000, // convert to nanoseconds
		state:            CircuitClosed,
	}
}

// Execute executes a function through the circuit breaker
func (cb *CircuitBreaker) Execute(
	fn func() error,
	collector *ErrorCollector,
	context string,
) error {
	currentTime := time.Now().UnixNano()

	// Check if circuit should be reset
	if cb.state == CircuitOpen && (currentTime-cb.lastFailureTime) > cb.resetTimeout {
		cb.state = CircuitHalfOpen
		cb.failureCount = 0
	}

	// If circuit is open, fail fast
	if cb.state == CircuitOpen {
		err := NewError(CategoryInternal, SeverityError,
			fmt.Sprintf("circuit breaker is open for %s", context))
		err.Details["context"] = context
		err.Details["failure_count"] = cb.failureCount
		err.Details["failure_threshold"] = cb.failureThreshold

		if collector != nil {
			collector.Add(err)
		}
		return err
	}

	// Execute the function
	err := fn()
	if err != nil {
		cb.failureCount++
		cb.lastFailureTime = currentTime

		if cb.failureCount >= cb.failureThreshold {
			cb.state = CircuitOpen
			
			circuitErr := NewError(CategoryInternal, SeverityWarning,
				fmt.Sprintf("circuit breaker opened for %s after %d failures",
					context, cb.failureCount))
			circuitErr.Details["context"] = context
			circuitErr.Details["failure_count"] = cb.failureCount
			circuitErr.Details["failure_threshold"] = cb.failureThreshold

			if collector != nil {
				collector.Add(circuitErr)
			}
		}

		return err
	}

	// Success - reset failure count and close circuit
	if cb.state == CircuitHalfOpen {
		cb.state = CircuitClosed
	}
	cb.failureCount = 0

	return nil
}