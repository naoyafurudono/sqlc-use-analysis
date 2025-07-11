package errors

import "fmt"

// ErrorReporter provides convenient methods for reporting errors
// This eliminates repetitive error creation and collection code
type ErrorReporter struct {
	collector *ErrorCollector
}

// NewErrorReporter creates a new error reporter
func NewErrorReporter(collector *ErrorCollector) *ErrorReporter {
	return &ErrorReporter{
		collector: collector,
	}
}

// ReportError creates and collects an error with the given parameters
// This method hides the complexity of error creation and collection
func (r *ErrorReporter) ReportError(category ErrorCategory, severity ErrorSeverity, message string, details map[string]interface{}) error {
	err := NewError(category, severity, message)
	
	// Add details if provided
	if details != nil {
		for key, value := range details {
			err.Details[key] = value
		}
	}
	
	return r.collector.Add(err)
}

// ReportErrorf creates and collects a formatted error
func (r *ErrorReporter) ReportErrorf(category ErrorCategory, severity ErrorSeverity, format string, args ...interface{}) error {
	message := fmt.Sprintf(format, args...)
	return r.ReportError(category, severity, message, nil)
}

// ReportErrorWithContext creates and collects an error with context details
func (r *ErrorReporter) ReportErrorWithContext(category ErrorCategory, severity ErrorSeverity, message string, context map[string]interface{}) error {
	return r.ReportError(category, severity, message, context)
}

// Convenient wrapper methods for common error types

// ReportAnalysisError reports an analysis-related error
func (r *ErrorReporter) ReportAnalysisError(message string, details map[string]interface{}) error {
	return r.ReportError(CategoryAnalysis, SeverityError, message, details)
}

// ReportParseError reports a parsing error
func (r *ErrorReporter) ReportParseError(message string, details map[string]interface{}) error {
	return r.ReportError(CategoryParse, SeverityError, message, details)
}

// ReportMappingError reports a mapping error
func (r *ErrorReporter) ReportMappingError(message string, details map[string]interface{}) error {
	return r.ReportError(CategoryMapping, SeverityError, message, details)
}

// ReportValidationError reports a validation error
func (r *ErrorReporter) ReportValidationError(message string, details map[string]interface{}) error {
	return r.ReportError(CategoryValidation, SeverityError, message, details)
}

// ReportWarning reports a warning
func (r *ErrorReporter) ReportWarning(category ErrorCategory, message string, details map[string]interface{}) error {
	return r.ReportError(category, SeverityWarning, message, details)
}

// ReportInfo reports an informational message
func (r *ErrorReporter) ReportInfo(category ErrorCategory, message string, details map[string]interface{}) error {
	return r.ReportError(category, SeverityInfo, message, details)
}

// Contextual error reporting methods

// WithQueryContext adds query context to error details
func (r *ErrorReporter) WithQueryContext(queryName, sql string) *QueryErrorReporter {
	return &QueryErrorReporter{
		reporter: r,
		context: map[string]interface{}{
			"query_name": queryName,
			"sql":        sql,
		},
	}
}

// WithFunctionContext adds function context to error details
func (r *ErrorReporter) WithFunctionContext(functionName, packageName, fileName string, line int) *FunctionErrorReporter {
	return &FunctionErrorReporter{
		reporter: r,
		context: map[string]interface{}{
			"function_name": functionName,
			"package_name":  packageName,
			"file_name":     fileName,
			"line":          line,
		},
	}
}

// QueryErrorReporter provides query-specific error reporting
type QueryErrorReporter struct {
	reporter *ErrorReporter
	context  map[string]interface{}
}

// Error reports an error with query context
func (qr *QueryErrorReporter) Error(category ErrorCategory, message string) error {
	return qr.reporter.ReportError(category, SeverityError, message, qr.context)
}

// Warning reports a warning with query context
func (qr *QueryErrorReporter) Warning(category ErrorCategory, message string) error {
	return qr.reporter.ReportError(category, SeverityWarning, message, qr.context)
}

// FunctionErrorReporter provides function-specific error reporting
type FunctionErrorReporter struct {
	reporter *ErrorReporter
	context  map[string]interface{}
}

// Error reports an error with function context
func (fr *FunctionErrorReporter) Error(category ErrorCategory, message string) error {
	return fr.reporter.ReportError(category, SeverityError, message, fr.context)
}

// Warning reports a warning with function context
func (fr *FunctionErrorReporter) Warning(category ErrorCategory, message string) error {
	return fr.reporter.ReportError(category, SeverityWarning, message, fr.context)
}

// Helper functions for creating common error details

// SQLQueryDetails creates standard details for SQL query errors
func SQLQueryDetails(queryName, sql string) map[string]interface{} {
	return map[string]interface{}{
		"query_name": queryName,
		"sql":        sql,
	}
}

// FunctionDetails creates standard details for function errors
func FunctionDetails(functionName, packageName, fileName string, line int) map[string]interface{} {
	return map[string]interface{}{
		"function_name": functionName,
		"package_name":  packageName,
		"file_name":     fileName,
		"line":          line,
	}
}

// TableDetails creates standard details for table errors
func TableDetails(tableName string) map[string]interface{} {
	return map[string]interface{}{
		"table_name": tableName,
	}
}

// MethodDetails creates standard details for method errors
func MethodDetails(methodName string, line, column int) map[string]interface{} {
	return map[string]interface{}{
		"method_name": methodName,
		"line":        line,
		"column":      column,
	}
}