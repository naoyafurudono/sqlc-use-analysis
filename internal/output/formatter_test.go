package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/naoyafurudono/sqlc-use-analysis/pkg/types"
)

func TestFormatter_FormatJSON(t *testing.T) {
	formatter := NewFormatter(types.FormatJSON, true)
	report := createTestReport()
	
	var buffer bytes.Buffer
	err := formatter.Format(&report, &buffer)
	if err != nil {
		t.Errorf("Format() error = %v", err)
		return
	}
	
	// Check if output is valid JSON
	var result map[string]interface{}
	err = json.Unmarshal(buffer.Bytes(), &result)
	if err != nil {
		t.Errorf("Output is not valid JSON: %v", err)
		return
	}
	
	// Check if required fields are present
	if _, exists := result["metadata"]; !exists {
		t.Error("Output missing metadata field")
	}
	
	if _, exists := result["summary"]; !exists {
		t.Error("Output missing summary field")
	}
	
	if _, exists := result["dependencies"]; !exists {
		t.Error("Output missing dependencies field")
	}
}

func TestFormatter_FormatCSV(t *testing.T) {
	formatter := NewFormatter(types.FormatCSV, false)
	report := createTestReport()
	
	var buffer bytes.Buffer
	err := formatter.Format(&report, &buffer)
	if err != nil {
		t.Errorf("Format() error = %v", err)
		return
	}
	
	output := buffer.String()
	
	// Check if output contains expected CSV headers
	if !strings.Contains(output, "Function,Package,File,Tables,Operations") {
		t.Error("CSV output missing function view header")
	}
	
	if !strings.Contains(output, "Table,Functions,Operations") {
		t.Error("CSV output missing table view header")
	}
	
	// Check if function and table names are present
	if !strings.Contains(output, "TestFunction") {
		t.Error("CSV output missing test function")
	}
	
	if !strings.Contains(output, "users") {
		t.Error("CSV output missing test table")
	}
}

func TestFormatter_FormatHTML(t *testing.T) {
	formatter := NewFormatter(types.FormatHTML, false)
	report := createTestReport()
	
	var buffer bytes.Buffer
	err := formatter.Format(&report, &buffer)
	if err != nil {
		t.Errorf("Format() error = %v", err)
		return
	}
	
	output := buffer.String()
	
	// Check if output contains expected HTML structure
	if !strings.Contains(output, "<!DOCTYPE html>") {
		t.Error("HTML output missing DOCTYPE")
	}
	
	if !strings.Contains(output, "<title>SQLC Dependency Analysis Report</title>") {
		t.Error("HTML output missing title")
	}
	
	if !strings.Contains(output, "Function View") {
		t.Error("HTML output missing function view section")
	}
	
	if !strings.Contains(output, "Table View") {
		t.Error("HTML output missing table view section")
	}
	
	// Check if function and table names are present
	if !strings.Contains(output, "TestFunction") {
		t.Error("HTML output missing test function")
	}
	
	if !strings.Contains(output, "users") {
		t.Error("HTML output missing test table")
	}
}

func TestFormatter_UnsupportedFormat(t *testing.T) {
	formatter := NewFormatter("unsupported", false)
	report := createTestReport()
	
	var buffer bytes.Buffer
	err := formatter.Format(&report, &buffer)
	if err == nil {
		t.Error("Expected error for unsupported format")
	}
}

func TestFormatter_PrettyJSON(t *testing.T) {
	formatter := NewFormatter(types.FormatJSON, true)
	report := createTestReport()
	
	var buffer bytes.Buffer
	err := formatter.Format(&report, &buffer)
	if err != nil {
		t.Errorf("Format() error = %v", err)
		return
	}
	
	output := buffer.String()
	
	// Check if output is pretty-printed (contains indentation)
	if !strings.Contains(output, "  \"metadata\"") {
		t.Error("Pretty JSON output missing indentation")
	}
}

func TestFormatter_MinifiedJSON(t *testing.T) {
	formatter := NewFormatter(types.FormatJSON, false)
	report := createTestReport()
	
	var buffer bytes.Buffer
	err := formatter.Format(&report, &buffer)
	if err != nil {
		t.Errorf("Format() error = %v", err)
		return
	}
	
	output := buffer.String()
	
	// Check if output is minified (no extra whitespace)
	if strings.Contains(output, "  \"metadata\"") {
		t.Error("Minified JSON output contains unnecessary indentation")
	}
}

func TestFormatter_HelperFunctions(t *testing.T) {
	// Test joinStrings
	result := joinStrings([]string{"a", "b", "c"}, ",")
	if result != "a,b,c" {
		t.Errorf("joinStrings() = %s, want a,b,c", result)
	}
	
	// Test empty slice
	result = joinStrings([]string{}, ",")
	if result != "" {
		t.Errorf("joinStrings() = %s, want empty string", result)
	}
	
	// Test sumOperations
	operationCounts := map[string]int{
		"SELECT": 5,
		"INSERT": 3,
		"UPDATE": 2,
	}
	sum := sumOperations(operationCounts)
	if sum != 10 {
		t.Errorf("sumOperations() = %d, want 10", sum)
	}
}

func TestFormatter_EdgeCases(t *testing.T) {
	// Test with empty report
	formatter := NewFormatter(types.FormatJSON, false)
	report := types.AnalysisReport{
		Summary: types.AnalysisSummary{
			FunctionCount: 0,
			TableCount:    0,
		},
		Dependencies: types.AnalysisResult{
			FunctionView: make(map[string]types.FunctionViewEntry),
			TableView:    make(map[string]types.TableViewEntry),
		},
	}
	
	var buffer bytes.Buffer
	err := formatter.Format(&report, &buffer)
	if err != nil {
		t.Errorf("Format() error = %v", err)
	}
	
	// Check if output is valid JSON
	var result map[string]interface{}
	err = json.Unmarshal(buffer.Bytes(), &result)
	if err != nil {
		t.Errorf("Output is not valid JSON: %v", err)
	}
}

// Helper function to create test report
func createTestReport() types.AnalysisReport {
	return types.AnalysisReport{
		Summary: types.AnalysisSummary{
			FunctionCount: 1,
			TableCount:    1,
			OperationCounts: map[string]int{
				"SELECT": 2,
				"INSERT": 1,
			},
			PackageCounts: map[string]int{
				"main": 1,
			},
		},
		Dependencies: types.AnalysisResult{
			FunctionView: map[string]types.FunctionViewEntry{
				"TestFunction": {
					FunctionName: "TestFunction",
					PackageName:  "main",
					FileName:     "main.go",
					StartLine:    10,
					EndLine:      20,
					TableAccess: map[string]types.TableAccessInfo{
						"users": {
							TableName: "users",
							Operations: map[string][]types.OperationCall{
								"SELECT": {
									{
										MethodName: "GetUser",
										Line:       15,
										Column:     10,
									},
								},
								"INSERT": {
									{
										MethodName: "CreateUser",
										Line:       18,
										Column:     10,
									},
								},
							},
						},
					},
				},
			},
			TableView: map[string]types.TableViewEntry{
				"users": {
					TableName: "users",
					AccessedBy: map[string]types.FunctionAccess{
						"TestFunction": {
							Function:   "TestFunction",
							Operations: []string{"SELECT", "INSERT"},
						},
					},
					OperationSummary: map[string]int{
						"SELECT": 1,
						"INSERT": 1,
					},
				},
			},
		},
		Circular: []types.CircularDependency{
			{
				Functions: []string{"FuncA", "FuncB", "FuncA"},
				Type:      "table_access",
			},
		},
		Suggestions: []types.OptimizationSuggestion{
			{
				Type:        "high_table_access",
				Function:    "TestFunction",
				Description: "Function accesses many tables",
				Severity:    "medium",
			},
		},
	}
}