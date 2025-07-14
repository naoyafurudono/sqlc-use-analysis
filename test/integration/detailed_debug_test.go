package integration

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/naoyafurudono/sqlc-use-analysis/internal/analyzer/dependency"
	gostatic "github.com/naoyafurudono/sqlc-use-analysis/internal/analyzer/go"
	"github.com/naoyafurudono/sqlc-use-analysis/internal/analyzer/sql"
	"github.com/naoyafurudono/sqlc-use-analysis/internal/errors"
	"github.com/naoyafurudono/sqlc-use-analysis/pkg/types"
)

// TestDetailedDebugAnalyzer tests each component separately to identify issues
func TestDetailedDebugAnalyzer(t *testing.T) {
	// Use the existing simple_project fixture
	fixturesPath := filepath.Join("..", "fixtures", "simple_project")
	
	// Verify fixture exists
	if _, err := os.Stat(fixturesPath); os.IsNotExist(err) {
		t.Skipf("Test fixture not found at %s", fixturesPath)
	}

	// Step 1: Test SQL Analysis
	t.Log("=== Testing SQL Analysis ===")
	errorCollector := errors.NewErrorCollector(100, false)
	sqlAnalyzer := sql.NewAnalyzer("postgresql", false, errorCollector)
	
	queries := []types.QueryInfo{
		{Name: "GetUser", SQL: "SELECT id, name, email, created_at FROM users WHERE id = $1"},
		{Name: "CreateUser", SQL: "INSERT INTO users (name, email) VALUES ($1, $2) RETURNING id, name, email, created_at"},
		{Name: "ListUsers", SQL: "SELECT id, name, email, created_at FROM users ORDER BY created_at DESC"},
		{Name: "GetPost", SQL: "SELECT p.id, p.title, p.content, p.author_id, p.created_at, u.name as author_name FROM posts p JOIN users u ON p.author_id = u.id WHERE p.id = $1"},
		{Name: "ListPostsByUser", SQL: "SELECT id, title, content, author_id, created_at FROM posts WHERE author_id = $1 ORDER BY created_at DESC"},
		{Name: "CreatePost", SQL: "INSERT INTO posts (title, content, author_id) VALUES ($1, $2, $3) RETURNING id, title, content, author_id, created_at"},
		{Name: "GetCommentsByPost", SQL: "SELECT c.id, c.content, c.author_id, c.created_at, u.name as author_name FROM comments c JOIN users u ON c.author_id = u.id WHERE c.post_id = $1 ORDER BY c.created_at"},
		{Name: "CreateComment", SQL: "INSERT INTO comments (post_id, author_id, content) VALUES ($1, $2, $3) RETURNING id, post_id, author_id, content, created_at"},
	}
	
	// Test individual SQL queries
	sqlMethods := make(map[string]types.SQLMethodInfo)
	for _, queryInfo := range queries {
		query := sql.Query{
			Text: queryInfo.SQL,
			Name: queryInfo.Name,
			Cmd:  ":exec",
		}
		
		result, err := sqlAnalyzer.AnalyzeQuery(query)
		if err != nil {
			t.Logf("SQL analysis error for %s: %v", queryInfo.Name, err)
		} else {
			t.Logf("SQL Method: %s", result.MethodName)
			for _, table := range result.Tables {
				t.Logf("  Table: %s, Operations: %v", table.TableName, table.Operations)
			}
			sqlMethods[result.MethodName] = result
		}
	}
	t.Logf("Total SQL methods: %d", len(sqlMethods))

	// Step 2: Test Go Analysis
	t.Log("=== Testing Go Analysis ===")
	goAnalyzer := gostatic.NewAnalyzer("", errorCollector)
	
	// Load packages (including service layer where SQL calls are made)
	dbPackagePath := filepath.Join(fixturesPath, "internal/db")
	servicePackagePath := filepath.Join(fixturesPath, "internal/service")
	handlerPackagePath := filepath.Join(fixturesPath, "internal/handler")
	
	err := goAnalyzer.LoadPackages(dbPackagePath, servicePackagePath, handlerPackagePath)
	if err != nil {
		t.Fatalf("Failed to load packages: %v", err)
	}
	
	// Analyze packages
	goFunctions, err := goAnalyzer.AnalyzePackages()
	if err != nil {
		t.Fatalf("Failed to analyze packages: %v", err)
	}
	
	t.Logf("Total Go functions: %d", len(goFunctions))
	for name, funcInfo := range goFunctions {
		t.Logf("Go Function: %s", name)
		t.Logf("  Package: %s, File: %s", funcInfo.PackageName, funcInfo.FileName)
		t.Logf("  SQL Calls: %d", len(funcInfo.SQLCalls))
		for _, sqlCall := range funcInfo.SQLCalls {
			t.Logf("    - %s at line %d", sqlCall.MethodName, sqlCall.Line)
		}
	}

	// Step 3: Test Dependency Mapping
	t.Log("=== Testing Dependency Mapping ===")
	mapper := gostatic.NewDependencyMapper(errorCollector)
	
	result, err := mapper.MapDependencies(goFunctions, sqlMethods)
	if err != nil {
		t.Fatalf("Failed to map dependencies: %v", err)
	}
	
	t.Logf("Function View entries: %d", len(result.FunctionView))
	for name, entry := range result.FunctionView {
		t.Logf("Function View: %s", name)
		t.Logf("  Table Access: %d", len(entry.TableAccess))
		for tableName, access := range entry.TableAccess {
			t.Logf("    Table: %s", tableName)
			for operation, calls := range access.Operations {
				t.Logf("      Operation: %s, Calls: %d", operation, len(calls))
			}
		}
	}
	
	t.Logf("Table View entries: %d", len(result.TableView))
	for name, entry := range result.TableView {
		t.Logf("Table View: %s", name)
		t.Logf("  Accessed by: %d functions", len(entry.AccessedBy))
		t.Logf("  Operations: %v", entry.OperationSummary)
	}

	// Step 4: Check errors
	t.Log("=== Checking Errors ===")
	allErrors := errorCollector.GetAllErrors()
	t.Logf("Total errors: %d", len(allErrors))
	for i, err := range allErrors {
		if i < 5 { // Limit to first 5 errors
			t.Logf("Error: [%s] %s - %s", err.Category, err.Severity.String(), err.Message)
			if err.Details != nil {
				for key, value := range err.Details {
					t.Logf("  %s: %v", key, value)
				}
			}
		}
	}
	if len(allErrors) > 5 {
		t.Logf("... and %d more errors", len(allErrors)-5)
	}

	// Step 5: Test Complete Workflow
	t.Log("=== Testing Complete Workflow ===")
	engine := dependency.NewEngine(errors.NewErrorCollector(100, false))
	allPackagePaths := []string{dbPackagePath, servicePackagePath, handlerPackagePath}
	finalResult, err := engine.AnalyzeDependencies(queries, allPackagePaths)
	if err != nil {
		t.Logf("Complete workflow error: %v", err)
	} else {
		t.Logf("Final result - Functions: %d, Tables: %d", 
			len(finalResult.FunctionView), len(finalResult.TableView))
	}
}