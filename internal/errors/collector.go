package errors

import (
	"fmt"
	"sync"
)

// ErrorCollector collects and manages errors during analysis
type ErrorCollector struct {
	errors     []*AnalysisError
	warnings   []*AnalysisError
	mu         sync.Mutex
	maxErrors  int
	stopOnFatal bool
}

// NewErrorCollector creates a new error collector
func NewErrorCollector(maxErrors int, stopOnFatal bool) *ErrorCollector {
	return &ErrorCollector{
		errors:      make([]*AnalysisError, 0),
		warnings:    make([]*AnalysisError, 0),
		maxErrors:   maxErrors,
		stopOnFatal: stopOnFatal,
	}
}

// Add adds an error to the collector
func (ec *ErrorCollector) Add(err *AnalysisError) error {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	
	switch err.Severity {
	case SeverityFatal:
		ec.errors = append(ec.errors, err)
		if ec.stopOnFatal {
			return err // 即座に処理を停止
		}
	case SeverityError:
		ec.errors = append(ec.errors, err)
		if len(ec.errors) > ec.maxErrors {
			return fmt.Errorf("too many errors: %d", len(ec.errors))
		}
	case SeverityWarning:
		ec.warnings = append(ec.warnings, err)
	}
	
	return nil
}

// HasErrors returns true if there are any errors
func (ec *ErrorCollector) HasErrors() bool {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	return len(ec.errors) > 0
}

// GetErrors returns all errors
func (ec *ErrorCollector) GetErrors() []*AnalysisError {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	
	result := make([]*AnalysisError, len(ec.errors))
	copy(result, ec.errors)
	return result
}

// GetWarnings returns all warnings
func (ec *ErrorCollector) GetWarnings() []*AnalysisError {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	
	result := make([]*AnalysisError, len(ec.warnings))
	copy(result, ec.warnings)
	return result
}

// ErrorReport represents a complete error report
type ErrorReport struct {
	Errors   []*AnalysisError `json:"errors"`
	Warnings []*AnalysisError `json:"warnings"`
	Summary  ErrorSummary     `json:"summary"`
}

// ErrorSummary provides a summary of errors
type ErrorSummary struct {
	TotalErrors   int                      `json:"total_errors"`
	TotalWarnings int                      `json:"total_warnings"`
	ByCategory    map[ErrorCategory]int    `json:"by_category"`
	BySeverity    map[ErrorSeverity]int    `json:"by_severity"`
}

// GetReport returns a complete error report
func (ec *ErrorCollector) GetReport() *ErrorReport {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	
	return &ErrorReport{
		Errors:   ec.errors,
		Warnings: ec.warnings,
		Summary:  ec.generateSummary(),
	}
}

func (ec *ErrorCollector) generateSummary() ErrorSummary {
	summary := ErrorSummary{
		TotalErrors:   len(ec.errors),
		TotalWarnings: len(ec.warnings),
		ByCategory:    make(map[ErrorCategory]int),
		BySeverity:    make(map[ErrorSeverity]int),
	}
	
	// エラーの集計
	for _, err := range ec.errors {
		summary.ByCategory[err.Category]++
		summary.BySeverity[err.Severity]++
	}
	
	// 警告の集計
	for _, warn := range ec.warnings {
		summary.ByCategory[warn.Category]++
		summary.BySeverity[warn.Severity]++
	}
	
	return summary
}