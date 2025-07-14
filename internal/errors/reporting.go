package errors

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"
)

// ErrorAggregator groups similar errors together
type ErrorAggregator struct {
	groups map[string][]*AnalysisError
}

// NewErrorAggregator creates a new error aggregator
func NewErrorAggregator() *ErrorAggregator {
	return &ErrorAggregator{
		groups: make(map[string][]*AnalysisError),
	}
}

// Add adds an error to the aggregator
func (ea *ErrorAggregator) Add(err *AnalysisError) {
	key := ea.generateKey(err)
	ea.groups[key] = append(ea.groups[key], err)
}

// generateKey generates a grouping key for similar errors
func (ea *ErrorAggregator) generateKey(err *AnalysisError) string {
	// Group by category, severity, and a simplified message
	baseMessage := ea.simplifyMessage(err.Message)
	return fmt.Sprintf("%s:%s:%s", err.Category, err.Severity.String(), baseMessage)
}

// simplifyMessage removes instance-specific details from error messages
func (ea *ErrorAggregator) simplifyMessage(message string) string {
	// Replace specific values with placeholders for grouping
	simplified := message
	
	// Replace file paths with placeholder
	simplified = strings.ReplaceAll(simplified, "/", "PATH_SEP")
	
	// Replace numbers with placeholder (for line numbers, etc.)
	for i := 0; i < 10; i++ {
		simplified = strings.ReplaceAll(simplified, fmt.Sprintf("%d", i), "N")
	}
	
	return simplified
}

// AggregatedError represents a group of similar errors
type AggregatedError struct {
	Key        string           `json:"key"`
	Count      int              `json:"count"`
	FirstError *AnalysisError   `json:"first_error"`
	Locations  []ErrorLocation  `json:"locations"`
	Category   ErrorCategory    `json:"category"`
	Severity   ErrorSeverity    `json:"severity"`
}

// GetAggregatedReport returns aggregated errors
func (ea *ErrorAggregator) GetAggregatedReport() []AggregatedError {
	var result []AggregatedError

	for key, errors := range ea.groups {
		if len(errors) == 0 {
			continue
		}

		locations := ea.extractLocations(errors)
		
		result = append(result, AggregatedError{
			Key:        key,
			Count:      len(errors),
			FirstError: errors[0],
			Locations:  locations,
			Category:   errors[0].Category,
			Severity:   errors[0].Severity,
		})
	}

	// Sort by count (descending) then by severity
	sort.Slice(result, func(i, j int) bool {
		if result[i].Count != result[j].Count {
			return result[i].Count > result[j].Count
		}
		return result[i].Severity < result[j].Severity
	})

	return result
}

// extractLocations extracts unique locations from a group of errors
func (ea *ErrorAggregator) extractLocations(errors []*AnalysisError) []ErrorLocation {
	locationMap := make(map[string]ErrorLocation)

	for _, err := range errors {
		if err.Location != nil {
			key := fmt.Sprintf("%s:%d", err.Location.File, err.Location.Line)
			locationMap[key] = *err.Location
		}
	}

	locations := make([]ErrorLocation, 0, len(locationMap))
	for _, loc := range locationMap {
		locations = append(locations, loc)
	}

	return locations
}

// ReportFormatter provides different formatting options for error reports
type ReportFormatter struct {
	includeStackTrace bool
	includeDetails    bool
	maxDetailsLength  int
}

// NewReportFormatter creates a new report formatter
func NewReportFormatter() *ReportFormatter {
	return &ReportFormatter{
		includeStackTrace: false,
		includeDetails:    true,
		maxDetailsLength:  500,
	}
}

// WithStackTrace enables stack trace inclusion
func (rf *ReportFormatter) WithStackTrace() *ReportFormatter {
	rf.includeStackTrace = true
	return rf
}

// WithoutDetails disables detail inclusion
func (rf *ReportFormatter) WithoutDetails() *ReportFormatter {
	rf.includeDetails = false
	return rf
}

// WithMaxDetailsLength sets the maximum length for details
func (rf *ReportFormatter) WithMaxDetailsLength(length int) *ReportFormatter {
	rf.maxDetailsLength = length
	return rf
}

// FormatReport formats an error report in the specified format
func (rf *ReportFormatter) FormatReport(report *ErrorReport, format string) (string, error) {
	switch strings.ToLower(format) {
	case "json":
		return rf.formatJSON(report)
	case "text", "":
		return rf.formatText(report), nil
	case "markdown", "md":
		return rf.formatMarkdown(report), nil
	case "csv":
		return rf.formatCSV(report), nil
	case "summary":
		return rf.formatSummary(report), nil
	default:
		return "", fmt.Errorf("unsupported format: %s", format)
	}
}

// formatJSON formats the report as JSON
func (rf *ReportFormatter) formatJSON(report *ErrorReport) (string, error) {
	// Prepare report for JSON serialization
	jsonReport := map[string]interface{}{
		"summary": report.Summary,
		"errors":  rf.prepareErrorsForJSON(report.Errors),
		"warnings": rf.prepareErrorsForJSON(report.Warnings),
		"generated_at": time.Now().Format(time.RFC3339),
	}

	data, err := json.MarshalIndent(jsonReport, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}

	return string(data), nil
}

// prepareErrorsForJSON prepares errors for JSON serialization
func (rf *ReportFormatter) prepareErrorsForJSON(errors []*AnalysisError) []map[string]interface{} {
	result := make([]map[string]interface{}, len(errors))

	for i, err := range errors {
		errMap := map[string]interface{}{
			"id":        err.ID,
			"category":  err.Category,
			"severity":  err.Severity.String(),
			"message":   err.Message,
			"timestamp": err.Timestamp.Format(time.RFC3339),
		}

		if err.Location != nil {
			errMap["location"] = map[string]interface{}{
				"file":     err.Location.File,
				"line":     err.Location.Line,
				"column":   err.Location.Column,
				"function": err.Location.Function,
			}
		}

		if rf.includeDetails && err.Details != nil && len(err.Details) > 0 {
			errMap["details"] = err.Details
		}

		if rf.includeStackTrace && err.StackTrace != "" {
			errMap["stack_trace"] = err.StackTrace
		}

		result[i] = errMap
	}

	return result
}

// formatText formats the report as plain text
func (rf *ReportFormatter) formatText(report *ErrorReport) string {
	var buf strings.Builder

	buf.WriteString("=== Analysis Error Report ===\n\n")
	buf.WriteString(fmt.Sprintf("Generated: %s\n\n", time.Now().Format("2006-01-02 15:04:05")))

	if len(report.Errors) > 0 {
		buf.WriteString("ERRORS:\n")
		buf.WriteString(strings.Repeat("=", 50) + "\n")
		for i, err := range report.Errors {
			buf.WriteString(fmt.Sprintf("%d. [%s] %s\n", i+1, err.Severity.String(), err.Message))
			if err.Location != nil {
				buf.WriteString(fmt.Sprintf("   Location: %s:%d\n", err.Location.File, err.Location.Line))
				if err.Location.Function != "" {
					buf.WriteString(fmt.Sprintf("   Function: %s\n", err.Location.Function))
				}
			}
			if rf.includeDetails && err.Details != nil && len(err.Details) > 0 {
				buf.WriteString("   Details:\n")
				for key, value := range err.Details {
					valueStr := fmt.Sprintf("%v", value)
					if len(valueStr) > rf.maxDetailsLength {
						valueStr = valueStr[:rf.maxDetailsLength] + "..."
					}
					buf.WriteString(fmt.Sprintf("     %s: %s\n", key, valueStr))
				}
			}
			buf.WriteString("\n")
		}
	}

	if len(report.Warnings) > 0 {
		buf.WriteString("WARNINGS:\n")
		buf.WriteString(strings.Repeat("=", 50) + "\n")
		for i, warn := range report.Warnings {
			buf.WriteString(fmt.Sprintf("%d. %s\n", i+1, warn.Message))
			if warn.Location != nil {
				buf.WriteString(fmt.Sprintf("   Location: %s:%d\n", warn.Location.File, warn.Location.Line))
			}
		}
		buf.WriteString("\n")
	}

	buf.WriteString("SUMMARY:\n")
	buf.WriteString(strings.Repeat("=", 50) + "\n")
	buf.WriteString(fmt.Sprintf("Total Errors: %d\n", report.Summary.TotalErrors))
	buf.WriteString(fmt.Sprintf("Total Warnings: %d\n", report.Summary.TotalWarnings))

	if len(report.Summary.ByCategory) > 0 {
		buf.WriteString("\nBy Category:\n")
		for category, count := range report.Summary.ByCategory {
			buf.WriteString(fmt.Sprintf("  %s: %d\n", category, count))
		}
	}

	if len(report.Summary.BySeverity) > 0 {
		buf.WriteString("\nBy Severity:\n")
		for severity, count := range report.Summary.BySeverity {
			buf.WriteString(fmt.Sprintf("  %s: %d\n", severity.String(), count))
		}
	}

	return buf.String()
}

// formatMarkdown formats the report as Markdown
func (rf *ReportFormatter) formatMarkdown(report *ErrorReport) string {
	var buf strings.Builder

	buf.WriteString("# Analysis Error Report\n\n")
	buf.WriteString(fmt.Sprintf("**Generated:** %s\n\n", time.Now().Format("2006-01-02 15:04:05")))

	if len(report.Errors) > 0 {
		buf.WriteString("## Errors\n\n")
		for i, err := range report.Errors {
			buf.WriteString(fmt.Sprintf("### %d. [%s] %s\n\n", i+1, err.Severity.String(), err.Message))
			if err.Location != nil {
				buf.WriteString(fmt.Sprintf("**Location:** `%s:%d`\n\n", err.Location.File, err.Location.Line))
				if err.Location.Function != "" {
					buf.WriteString(fmt.Sprintf("**Function:** `%s`\n\n", err.Location.Function))
				}
			}
			if rf.includeDetails && err.Details != nil && len(err.Details) > 0 {
				buf.WriteString("**Details:**\n\n")
				for key, value := range err.Details {
					buf.WriteString(fmt.Sprintf("- **%s:** `%v`\n", key, value))
				}
				buf.WriteString("\n")
			}
		}
	}

	if len(report.Warnings) > 0 {
		buf.WriteString("## Warnings\n\n")
		for i, warn := range report.Warnings {
			buf.WriteString(fmt.Sprintf("### %d. %s\n\n", i+1, warn.Message))
			if warn.Location != nil {
				buf.WriteString(fmt.Sprintf("**Location:** `%s:%d`\n\n", warn.Location.File, warn.Location.Line))
			}
		}
	}

	buf.WriteString("## Summary\n\n")
	buf.WriteString(fmt.Sprintf("- **Total Errors:** %d\n", report.Summary.TotalErrors))
	buf.WriteString(fmt.Sprintf("- **Total Warnings:** %d\n", report.Summary.TotalWarnings))

	if len(report.Summary.ByCategory) > 0 {
		buf.WriteString("\n### By Category\n\n")
		for category, count := range report.Summary.ByCategory {
			buf.WriteString(fmt.Sprintf("- **%s:** %d\n", category, count))
		}
	}

	return buf.String()
}

// formatCSV formats the report as CSV
func (rf *ReportFormatter) formatCSV(report *ErrorReport) string {
	var buf strings.Builder

	// CSV Header
	buf.WriteString("Severity,Category,Message,File,Line,Function,Timestamp\n")

	// Errors
	for _, err := range report.Errors {
		buf.WriteString(rf.formatErrorAsCSVRow(err))
	}

	// Warnings
	for _, warn := range report.Warnings {
		buf.WriteString(rf.formatErrorAsCSVRow(warn))
	}

	return buf.String()
}

// formatErrorAsCSVRow formats a single error as CSV row
func (rf *ReportFormatter) formatErrorAsCSVRow(err *AnalysisError) string {
	file := ""
	line := ""
	function := ""

	if err.Location != nil {
		file = err.Location.File
		line = fmt.Sprintf("%d", err.Location.Line)
		function = err.Location.Function
	}

	// Escape CSV values
	severity := rf.escapeCSVValue(err.Severity.String())
	category := rf.escapeCSVValue(string(err.Category))
	message := rf.escapeCSVValue(err.Message)
	fileEscaped := rf.escapeCSVValue(file)
	functionEscaped := rf.escapeCSVValue(function)
	timestamp := err.Timestamp.Format("2006-01-02 15:04:05")

	return fmt.Sprintf("%s,%s,%s,%s,%s,%s,%s\n",
		severity, category, message, fileEscaped, line, functionEscaped, timestamp)
}

// escapeCSVValue escapes CSV values
func (rf *ReportFormatter) escapeCSVValue(value string) string {
	if strings.Contains(value, ",") || strings.Contains(value, "\"") || strings.Contains(value, "\n") {
		return fmt.Sprintf("\"%s\"", strings.ReplaceAll(value, "\"", "\"\""))
	}
	return value
}

// formatSummary formats a summary report
func (rf *ReportFormatter) formatSummary(report *ErrorReport) string {
	var buf strings.Builder

	buf.WriteString("Error Summary\n")
	buf.WriteString("=============\n\n")

	buf.WriteString(fmt.Sprintf("Total Issues: %d (%d errors, %d warnings)\n\n",
		report.Summary.TotalErrors+report.Summary.TotalWarnings,
		report.Summary.TotalErrors,
		report.Summary.TotalWarnings))

	if len(report.Summary.ByCategory) > 0 {
		buf.WriteString("By Category:\n")
		for category, count := range report.Summary.ByCategory {
			buf.WriteString(fmt.Sprintf("  %s: %d\n", category, count))
		}
		buf.WriteString("\n")
	}

	if len(report.Summary.BySeverity) > 0 {
		buf.WriteString("By Severity:\n")
		for severity, count := range report.Summary.BySeverity {
			buf.WriteString(fmt.Sprintf("  %s: %d\n", severity.String(), count))
		}
	}

	return buf.String()
}