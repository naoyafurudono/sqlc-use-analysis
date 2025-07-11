package errors

import (
	"testing"
)

func TestNewError(t *testing.T) {
	err := NewError(CategoryConfig, SeverityError, "test error")
	
	if err == nil {
		t.Fatal("NewError() returned nil")
	}
	
	if err.Category != CategoryConfig {
		t.Errorf("Expected Category to be %v, got %v", CategoryConfig, err.Category)
	}
	
	if err.Severity != SeverityError {
		t.Errorf("Expected Severity to be %v, got %v", SeverityError, err.Severity)
	}
	
	if err.Message != "test error" {
		t.Errorf("Expected Message to be 'test error', got '%s'", err.Message)
	}
	
	if err.ID == "" {
		t.Error("Expected ID to be set")
	}
	
	if err.Timestamp.IsZero() {
		t.Error("Expected Timestamp to be set")
	}
	
	if err.Location == nil {
		t.Error("Expected Location to be set")
	}
	
	if err.Details == nil {
		t.Error("Expected Details to be initialized")
	}
}

func TestAnalysisError_Error(t *testing.T) {
	err := NewError(CategoryConfig, SeverityError, "test error")
	
	errorStr := err.Error()
	if errorStr == "" {
		t.Error("Expected Error() to return non-empty string")
	}
	
	// エラーメッセージに必要な情報が含まれているかチェック
	if !contains(errorStr, string(CategoryConfig)) {
		t.Errorf("Expected error string to contain category, got '%s'", errorStr)
	}
	
	if !contains(errorStr, "test error") {
		t.Errorf("Expected error string to contain message, got '%s'", errorStr)
	}
}

func TestWrap(t *testing.T) {
	originalErr := NewError(CategoryParse, SeverityError, "original error")
	wrappedErr := Wrap(originalErr, "wrapped")
	
	if wrappedErr == nil {
		t.Fatal("Wrap() returned nil")
	}
	
	if !contains(wrappedErr.Message, "wrapped") {
		t.Errorf("Expected wrapped message to contain 'wrapped', got '%s'", wrappedErr.Message)
	}
	
	if !contains(wrappedErr.Message, "original error") {
		t.Errorf("Expected wrapped message to contain original message, got '%s'", wrappedErr.Message)
	}
}

func TestWrap_NilError(t *testing.T) {
	wrappedErr := Wrap(nil, "wrapped")
	
	if wrappedErr != nil {
		t.Error("Expected Wrap(nil, ...) to return nil")
	}
}

func TestErrorSeverity_String(t *testing.T) {
	tests := []struct {
		severity ErrorSeverity
		want     string
	}{
		{SeverityFatal, "FATAL"},
		{SeverityError, "ERROR"},
		{SeverityWarning, "WARNING"},
		{SeverityInfo, "INFO"},
	}
	
	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			if got := tt.severity.String(); got != tt.want {
				t.Errorf("ErrorSeverity.String() = %v, want %v", got, tt.want)
			}
		})
	}
}

// Helper function to check if a string contains a substring
func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}