package errors

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"strings"
)

// ErrorLogger provides structured logging for analysis errors
type ErrorLogger struct {
	logger *slog.Logger
	level  slog.Level
}

// NewErrorLogger creates a new error logger
func NewErrorLogger(level slog.Level) *ErrorLogger {
	// Create a structured logger with JSON output
	handler := slog.NewJSONHandler(os.Stderr, &slog.HandlerOptions{
		Level: level,
		AddSource: true,
	})
	
	logger := slog.New(handler)
	
	return &ErrorLogger{
		logger: logger,
		level:  level,
	}
}

// NewErrorLoggerWithHandler creates a new error logger with custom handler
func NewErrorLoggerWithHandler(handler slog.Handler) *ErrorLogger {
	return &ErrorLogger{
		logger: slog.New(handler),
		level:  slog.LevelDebug,
	}
}

// LogError logs an analysis error with structured attributes
func (el *ErrorLogger) LogError(ctx context.Context, err *AnalysisError) {
	attrs := []slog.Attr{
		slog.String("error_id", err.ID),
		slog.String("category", string(err.Category)),
		slog.String("severity", err.Severity.String()),
		slog.Time("timestamp", err.Timestamp),
	}

	// Add location information if available
	if err.Location != nil {
		locationAttrs := []slog.Attr{
			slog.String("file", err.Location.File),
			slog.Int("line", err.Location.Line),
		}
		if err.Location.Column > 0 {
			locationAttrs = append(locationAttrs, slog.Int("column", err.Location.Column))
		}
		if err.Location.Function != "" {
			locationAttrs = append(locationAttrs, slog.String("function", err.Location.Function))
		}
		locationGroup := slog.Group("location", convertAttrsToAny(locationAttrs)...)
		attrs = append(attrs, locationGroup)
	}

	// Add details if available
	if err.Details != nil && len(err.Details) > 0 {
		detailAttrs := make([]slog.Attr, 0, len(err.Details))
		for key, value := range err.Details {
			detailAttrs = append(detailAttrs, slog.Any(key, value))
		}
		detailsGroup := slog.Group("details", convertAttrsToAny(detailAttrs)...)
		attrs = append(attrs, detailsGroup)
	}

	// Add stack trace if available and in debug mode
	if err.StackTrace != "" && el.level <= slog.LevelDebug {
		attrs = append(attrs, slog.String("stack_trace", err.StackTrace))
	}

	// Add wrapped error if available
	if err.Wrapped != nil {
		attrs = append(attrs, slog.String("wrapped_error", err.Wrapped.Error()))
	}

	// Determine log level based on error severity
	logLevel := el.severityToLogLevel(err.Severity)
	
	el.logger.LogAttrs(ctx, logLevel, err.Message, attrs...)
}

// LogErrors logs multiple errors
func (el *ErrorLogger) LogErrors(ctx context.Context, errors []*AnalysisError) {
	for _, err := range errors {
		el.LogError(ctx, err)
	}
}

// LogErrorReport logs a complete error report
func (el *ErrorLogger) LogErrorReport(ctx context.Context, report *ErrorReport) {
	// Log summary first
	summaryAttrs := []slog.Attr{
		slog.Int("total_errors", report.Summary.TotalErrors),
		slog.Int("total_warnings", report.Summary.TotalWarnings),
	}

	// Add category breakdown
	if len(report.Summary.ByCategory) > 0 {
		categoryAttrs := make([]slog.Attr, 0, len(report.Summary.ByCategory))
		for category, count := range report.Summary.ByCategory {
			categoryAttrs = append(categoryAttrs, slog.Int(string(category), count))
		}
		categoryGroup := slog.Group("by_category", convertAttrsToAny(categoryAttrs)...)
		summaryAttrs = append(summaryAttrs, categoryGroup)
	}

	// Add severity breakdown
	if len(report.Summary.BySeverity) > 0 {
		severityAttrs := make([]slog.Attr, 0, len(report.Summary.BySeverity))
		for severity, count := range report.Summary.BySeverity {
			severityAttrs = append(severityAttrs, slog.Int(severity.String(), count))
		}
		severityGroup := slog.Group("by_severity", convertAttrsToAny(severityAttrs)...)
		summaryAttrs = append(summaryAttrs, severityGroup)
	}

	el.logger.LogAttrs(ctx, slog.LevelInfo, "Analysis completed with error summary", summaryAttrs...)

	// Log individual errors
	el.LogErrors(ctx, report.Errors)
	el.LogErrors(ctx, report.Warnings)
}

// convertAttrsToAny converts []slog.Attr to []any for slog.Group
func convertAttrsToAny(attrs []slog.Attr) []any {
	result := make([]any, len(attrs))
	for i, attr := range attrs {
		result[i] = attr
	}
	return result
}

// severityToLogLevel converts error severity to slog level
func (el *ErrorLogger) severityToLogLevel(severity ErrorSeverity) slog.Level {
	switch severity {
	case SeverityFatal:
		return slog.LevelError
	case SeverityError:
		return slog.LevelError
	case SeverityWarning:
		return slog.LevelWarn
	case SeverityInfo:
		return slog.LevelInfo
	default:
		return slog.LevelDebug
	}
}

// LogOperationStart logs the start of an operation
func (el *ErrorLogger) LogOperationStart(ctx context.Context, operation string, details map[string]interface{}) {
	attrs := []slog.Attr{
		slog.String("operation", operation),
		slog.String("status", "started"),
	}

	if details != nil {
		for key, value := range details {
			attrs = append(attrs, slog.Any(key, value))
		}
	}

	el.logger.LogAttrs(ctx, slog.LevelInfo, fmt.Sprintf("Starting %s", operation), attrs...)
}

// LogOperationEnd logs the end of an operation
func (el *ErrorLogger) LogOperationEnd(ctx context.Context, operation string, success bool, details map[string]interface{}) {
	status := "completed"
	if !success {
		status = "failed"
	}

	attrs := []slog.Attr{
		slog.String("operation", operation),
		slog.String("status", status),
		slog.Bool("success", success),
	}

	if details != nil {
		for key, value := range details {
			attrs = append(attrs, slog.Any(key, value))
		}
	}

	level := slog.LevelInfo
	if !success {
		level = slog.LevelError
	}

	el.logger.LogAttrs(ctx, level, fmt.Sprintf("Finished %s", operation), attrs...)
}

// LogProgress logs progress information
func (el *ErrorLogger) LogProgress(ctx context.Context, operation string, current, total int, details map[string]interface{}) {
	percentage := float64(current) / float64(total) * 100

	attrs := []slog.Attr{
		slog.String("operation", operation),
		slog.Int("current", current),
		slog.Int("total", total),
		slog.Float64("percentage", percentage),
	}

	if details != nil {
		for key, value := range details {
			attrs = append(attrs, slog.Any(key, value))
		}
	}

	el.logger.LogAttrs(ctx, slog.LevelDebug, 
		fmt.Sprintf("Progress: %s (%d/%d - %.1f%%)", operation, current, total, percentage), 
		attrs...)
}

// StructuredErrorCollector combines error collection with structured logging
type StructuredErrorCollector struct {
	*ErrorCollector
	logger *ErrorLogger
	ctx    context.Context
}

// NewStructuredErrorCollector creates a new structured error collector
func NewStructuredErrorCollector(maxErrors int, stopOnFatal bool, logger *ErrorLogger) *StructuredErrorCollector {
	return &StructuredErrorCollector{
		ErrorCollector: NewErrorCollector(maxErrors, stopOnFatal),
		logger:         logger,
		ctx:           context.Background(),
	}
}

// WithContext sets the context for logging
func (sec *StructuredErrorCollector) WithContext(ctx context.Context) *StructuredErrorCollector {
	sec.ctx = ctx
	return sec
}

// Add adds an error and logs it
func (sec *StructuredErrorCollector) Add(err *AnalysisError) error {
	// Log the error immediately
	if sec.logger != nil {
		sec.logger.LogError(sec.ctx, err)
	}

	// Add to collection
	return sec.ErrorCollector.Add(err)
}

// GetStructuredReport returns a report and logs the summary
func (sec *StructuredErrorCollector) GetStructuredReport() *ErrorReport {
	report := sec.ErrorCollector.GetReport()
	
	if sec.logger != nil {
		sec.logger.LogErrorReport(sec.ctx, report)
	}
	
	return report
}

// UserFriendlyMessageProvider provides user-friendly error messages
type UserFriendlyMessageProvider struct {
	messages map[string]string
}

// NewUserFriendlyMessageProvider creates a new message provider
func NewUserFriendlyMessageProvider() *UserFriendlyMessageProvider {
	return &UserFriendlyMessageProvider{
		messages: getDefaultErrorMessages(),
	}
}

// getDefaultErrorMessages returns default user-friendly error messages
func getDefaultErrorMessages() map[string]string {
	return map[string]string{
		"CONFIG_MISSING_ROOT": "Configuration error: 'root_path' is required. Please specify the project root directory in your configuration.",
		"PARSE_INVALID_SQL":   "SQL parsing error: The query contains invalid SQL syntax. Please check your query definition.",
		"ANALYSIS_CYCLIC_DEP": "Analysis warning: Circular dependency detected in function calls. This may indicate a design issue.",
		"MAPPING_NO_MATCH":    "Mapping error: Unable to match SQL method with Go function. Please verify your sqlc configuration.",
		"IO_FILE_NOT_FOUND":   "File not found: The specified file could not be read. Please check the file path and permissions.",
		"INTERNAL_PANIC":      "Internal error: An unexpected error occurred. Please report this issue.",
	}
}

// AddMessage adds or updates a user-friendly message
func (ufmp *UserFriendlyMessageProvider) AddMessage(errorType, message string) {
	ufmp.messages[errorType] = message
}

// GetUserFriendlyMessage returns a user-friendly message for an error
func (ufmp *UserFriendlyMessageProvider) GetUserFriendlyMessage(err *AnalysisError) string {
	// Try to find a specific message for this error
	if msg, ok := ufmp.messages[err.ID]; ok {
		return msg
	}

	// Try to find a message based on category and type
	categoryKey := fmt.Sprintf("%s_%s", err.Category, strings.ToUpper(strings.ReplaceAll(err.Message, " ", "_")))
	if msg, ok := ufmp.messages[categoryKey]; ok {
		return msg
	}

	// Fall back to original message
	return err.Message
}

// GetUserFriendlyReport generates a user-friendly error report
func (ufmp *UserFriendlyMessageProvider) GetUserFriendlyReport(report *ErrorReport) *ErrorReport {
	// Create a copy of the report with user-friendly messages
	friendlyReport := &ErrorReport{
		Summary: report.Summary,
		Errors:  make([]*AnalysisError, len(report.Errors)),
		Warnings: make([]*AnalysisError, len(report.Warnings)),
	}

	// Convert errors
	for i, err := range report.Errors {
		friendlyErr := *err // Copy the error
		friendlyErr.Message = ufmp.GetUserFriendlyMessage(err)
		friendlyReport.Errors[i] = &friendlyErr
	}

	// Convert warnings
	for i, warn := range report.Warnings {
		friendlyWarn := *warn // Copy the warning
		friendlyWarn.Message = ufmp.GetUserFriendlyMessage(warn)
		friendlyReport.Warnings[i] = &friendlyWarn
	}

	return friendlyReport
}