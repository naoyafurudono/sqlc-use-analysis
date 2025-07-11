package errors

import (
	"testing"
)

func TestErrorReporter_ReportError(t *testing.T) {
	collector := NewErrorCollector(10, false)
	reporter := NewErrorReporter(collector)
	
	details := map[string]interface{}{
		"test_key": "test_value",
		"number":   42,
	}
	
	err := reporter.ReportError(CategoryAnalysis, SeverityError, "test error", details)
	if err != nil {
		t.Errorf("ReportError() failed: %v", err)
	}
	
	errors := collector.GetAllErrors()
	if len(errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(errors))
		return
	}
	
	error := errors[0]
	if error.Category != CategoryAnalysis {
		t.Errorf("Expected category %s, got %s", CategoryAnalysis, error.Category)
	}
	
	if error.Severity != SeverityError {
		t.Errorf("Expected severity %s, got %s", SeverityError, error.Severity)
	}
	
	if error.Message != "test error" {
		t.Errorf("Expected message 'test error', got '%s'", error.Message)
	}
	
	if error.Details["test_key"] != "test_value" {
		t.Errorf("Expected detail test_key='test_value', got %v", error.Details["test_key"])
	}
}

func TestErrorReporter_ReportErrorf(t *testing.T) {
	collector := NewErrorCollector(10, false)
	reporter := NewErrorReporter(collector)
	
	err := reporter.ReportErrorf(CategoryParse, SeverityWarning, "test error %d: %s", 42, "formatted")
	if err != nil {
		t.Errorf("ReportErrorf() failed: %v", err)
	}
	
	errors := collector.GetAllErrors()
	if len(errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(errors))
		return
	}
	
	if errors[0].Message != "test error 42: formatted" {
		t.Errorf("Expected formatted message, got '%s'", errors[0].Message)
	}
}

func TestErrorReporter_ConvenienceMethods(t *testing.T) {
	collector := NewErrorCollector(10, false)
	reporter := NewErrorReporter(collector)
	
	details := map[string]interface{}{"key": "value"}
	
	// Test convenience methods
	tests := []struct {
		name     string
		method   func() error
		category ErrorCategory
	}{
		{
			name:     "ReportAnalysisError",
			method:   func() error { return reporter.ReportAnalysisError("analysis error", details) },
			category: CategoryAnalysis,
		},
		{
			name:     "ReportParseError",
			method:   func() error { return reporter.ReportParseError("parse error", details) },
			category: CategoryParse,
		},
		{
			name:     "ReportMappingError",
			method:   func() error { return reporter.ReportMappingError("mapping error", details) },
			category: CategoryMapping,
		},
		{
			name:     "ReportValidationError",
			method:   func() error { return reporter.ReportValidationError("validation error", details) },
			category: CategoryValidation,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.method()
			if err != nil {
				t.Errorf("%s failed: %v", tt.name, err)
			}
		})
	}
	
	errors := collector.GetAllErrors()
	if len(errors) != len(tests) {
		t.Errorf("Expected %d errors, got %d", len(tests), len(errors))
	}
}

func TestErrorReporter_WithQueryContext(t *testing.T) {
	collector := NewErrorCollector(10, false)
	reporter := NewErrorReporter(collector)
	
	queryReporter := reporter.WithQueryContext("GetUser", "SELECT * FROM users")
	
	err := queryReporter.Error(CategoryAnalysis, "query analysis failed")
	if err != nil {
		t.Errorf("Query error reporting failed: %v", err)
	}
	
	errors := collector.GetAllErrors()
	if len(errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(errors))
		return
	}
	
	error := errors[0]
	if error.Details["query_name"] != "GetUser" {
		t.Errorf("Expected query_name 'GetUser', got %v", error.Details["query_name"])
	}
	
	if error.Details["sql"] != "SELECT * FROM users" {
		t.Errorf("Expected sql 'SELECT * FROM users', got %v", error.Details["sql"])
	}
}

func TestErrorReporter_WithFunctionContext(t *testing.T) {
	collector := NewErrorCollector(10, false)
	reporter := NewErrorReporter(collector)
	
	functionReporter := reporter.WithFunctionContext("TestFunc", "main", "main.go", 42)
	
	err := functionReporter.Warning(CategoryAnalysis, "function analysis warning")
	if err != nil {
		t.Errorf("Function warning reporting failed: %v", err)
	}
	
	errors := collector.GetAllErrors()
	if len(errors) != 1 {
		t.Errorf("Expected 1 error, got %d", len(errors))
		return
	}
	
	error := errors[0]
	if error.Details["function_name"] != "TestFunc" {
		t.Errorf("Expected function_name 'TestFunc', got %v", error.Details["function_name"])
	}
	
	if error.Details["line"] != 42 {
		t.Errorf("Expected line 42, got %v", error.Details["line"])
	}
	
	if error.Severity != SeverityWarning {
		t.Errorf("Expected warning severity, got %s", error.Severity.String())
	}
}

func TestHelperFunctions(t *testing.T) {
	// Test SQLQueryDetails
	details := SQLQueryDetails("TestQuery", "SELECT 1")
	if details["query_name"] != "TestQuery" {
		t.Errorf("Expected query_name 'TestQuery', got %v", details["query_name"])
	}
	if details["sql"] != "SELECT 1" {
		t.Errorf("Expected sql 'SELECT 1', got %v", details["sql"])
	}
	
	// Test FunctionDetails
	details = FunctionDetails("TestFunc", "main", "main.go", 42)
	if details["function_name"] != "TestFunc" {
		t.Errorf("Expected function_name 'TestFunc', got %v", details["function_name"])
	}
	if details["line"] != 42 {
		t.Errorf("Expected line 42, got %v", details["line"])
	}
	
	// Test TableDetails
	details = TableDetails("users")
	if details["table_name"] != "users" {
		t.Errorf("Expected table_name 'users', got %v", details["table_name"])
	}
	
	// Test MethodDetails
	details = MethodDetails("GetUser", 10, 20)
	if details["method_name"] != "GetUser" {
		t.Errorf("Expected method_name 'GetUser', got %v", details["method_name"])
	}
	if details["line"] != 10 {
		t.Errorf("Expected line 10, got %v", details["line"])
	}
	if details["column"] != 20 {
		t.Errorf("Expected column 20, got %v", details["column"])
	}
}