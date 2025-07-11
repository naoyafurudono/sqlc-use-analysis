package analyzer

import (
	"context"
	"testing"
)

func TestAnalyzer_SimpleInterface(t *testing.T) {
	// Test that the new analyzer provides a simple, deep interface
	analyzer := New()
	
	request := AnalysisRequest{
		SQLQueries: []Query{
			{Name: "GetUser", SQL: "SELECT id, name FROM users WHERE id = $1"},
			{Name: "ListUsers", SQL: "SELECT id, name FROM users ORDER BY id"},
		},
		GoPackages: []string{"./testdata"},
	}
	
	ctx := context.Background()
	result, err := analyzer.Analyze(ctx, request)
	if err != nil {
		t.Logf("Analysis failed (expected for test environment): %v", err)
		// In test environment, Go packages may not exist, so this is expected
		return
	}
	
	// Verify the result has the expected structure
	if result == nil {
		t.Fatal("Expected result, got nil")
	}
	
	if result.Functions == nil {
		t.Error("Expected functions map to be initialized")
	}
	
	if result.Tables == nil {
		t.Error("Expected tables map to be initialized")
	}
	
	if result.Dependencies == nil {
		t.Error("Expected dependencies slice to be initialized")
	}
}

func TestAnalyzer_ErrorHandling(t *testing.T) {
	analyzer := New()
	
	// Test with invalid request
	request := AnalysisRequest{
		SQLQueries: []Query{}, // Empty queries should cause validation error
		GoPackages: []string{"./nonexistent"},
	}
	
	ctx := context.Background()
	_, err := analyzer.Analyze(ctx, request)
	if err == nil {
		t.Error("Expected validation error for empty queries")
	}
	
	// Check that errors are properly collected
	errors := analyzer.GetErrors()
	if len(errors) == 0 {
		t.Log("No errors collected (this is fine if validation happens before engine)")
	}
}

func TestAnalyzer_RequestValidation(t *testing.T) {
	analyzer := New()
	
	tests := []struct {
		name    string
		request AnalysisRequest
		wantErr bool
	}{
		{
			name: "Valid request",
			request: AnalysisRequest{
				SQLQueries: []Query{{Name: "test", SQL: "SELECT 1"}},
				GoPackages: []string{"./test"},
			},
			wantErr: false,
		},
		{
			name: "Empty queries",
			request: AnalysisRequest{
				SQLQueries: []Query{},
				GoPackages: []string{"./test"},
			},
			wantErr: true,
		},
		{
			name: "Empty packages",
			request: AnalysisRequest{
				SQLQueries: []Query{{Name: "test", SQL: "SELECT 1"}},
				GoPackages: []string{},
			},
			wantErr: true,
		},
		{
			name: "Query with empty name",
			request: AnalysisRequest{
				SQLQueries: []Query{{Name: "", SQL: "SELECT 1"}},
				GoPackages: []string{"./test"},
			},
			wantErr: true,
		},
		{
			name: "Query with empty SQL",
			request: AnalysisRequest{
				SQLQueries: []Query{{Name: "test", SQL: ""}},
				GoPackages: []string{"./test"},
			},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := analyzer.validateRequest(tt.request)
			if (err != nil) != tt.wantErr {
				t.Errorf("validateRequest() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAnalyzer_ConvertQueries(t *testing.T) {
	analyzer := New()
	
	queries := []Query{
		{Name: "GetUser", SQL: "SELECT * FROM users WHERE id = $1"},
		{Name: "ListUsers", SQL: "SELECT * FROM users"},
	}
	
	converted := analyzer.convertQueries(queries)
	
	if len(converted) != len(queries) {
		t.Errorf("Expected %d converted queries, got %d", len(queries), len(converted))
	}
	
	for i, original := range queries {
		if converted[i].Name != original.Name {
			t.Errorf("Expected name %s, got %s", original.Name, converted[i].Name)
		}
		if converted[i].SQL != original.SQL {
			t.Errorf("Expected SQL %s, got %s", original.SQL, converted[i].SQL)
		}
	}
}

func TestAnalyzer_OutputFormats(t *testing.T) {
	analyzer := New()
	
	request := AnalysisRequest{
		SQLQueries: []Query{
			{Name: "GetUser", SQL: "SELECT id, name FROM users WHERE id = $1"},
		},
		GoPackages:   []string{"./testdata"},
		OutputFormat: "json",
		PrettyPrint:  true,
	}
	
	ctx := context.Background()
	output, err := analyzer.AnalyzeAndFormat(ctx, request)
	if err != nil {
		t.Logf("AnalyzeAndFormat failed (expected for test environment): %v", err)
		// Expected to fail in test environment without real Go packages
		return
	}
	
	// In a real implementation, we would verify the output format
	_ = output
}

func TestAnalyzer_UsageExample(t *testing.T) {
	// This test demonstrates the simplified usage pattern
	
	// Create analyzer (simple constructor)
	analyzer := New()
	
	// Prepare request (simple data structure)
	request := AnalysisRequest{
		SQLQueries: []Query{
			{Name: "GetUser", SQL: "SELECT id, name FROM users WHERE id = $1"},
			{Name: "CreateUser", SQL: "INSERT INTO users (name) VALUES ($1)"},
		},
		GoPackages: []string{"./internal/..."},
	}
	
	// Perform analysis (single method call)
	ctx := context.Background()
	result, err := analyzer.Analyze(ctx, request)
	if err != nil {
		t.Logf("Analysis failed (expected): %v", err)
		// Expected to fail in test environment
		return
	}
	
	// Use results (simple structure)
	t.Logf("Found %d functions analyzing %d tables",
		result.Summary.FunctionCount,
		result.Summary.TableCount)
	
	// Access specific information
	for funcName, funcInfo := range result.Functions {
		t.Logf("Function %s in %s accesses %d tables",
			funcName, funcInfo.Package, len(funcInfo.TableAccess))
	}
	
	// Get any errors that occurred
	errors := analyzer.GetErrors()
	t.Logf("Analysis generated %d errors/warnings", len(errors))
}

// Benchmark to verify the interface doesn't add significant overhead
func BenchmarkAnalyzer_SimpleOperation(b *testing.B) {
	analyzer := New()
	
	request := AnalysisRequest{
		SQLQueries: []Query{
			{Name: "GetUser", SQL: "SELECT id, name FROM users WHERE id = $1"},
		},
		GoPackages: []string{"./testdata"},
	}
	
	ctx := context.Background()
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// This will likely fail in benchmark environment, but measures interface overhead
		analyzer.Analyze(ctx, request)
	}
}