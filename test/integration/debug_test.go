package integration

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/naoyafurudono/sqlc-use-analysis/pkg/analyzer"
)

// TestDebugAnalyzer tests the analyzer with debug information
func TestDebugAnalyzer(t *testing.T) {
	// Create a minimal test to debug what's happening
	
	// Create analyzer
	a := analyzer.New()

	// Load queries from the fixture query.sql file
	queries := []analyzer.Query{
		{
			Name: "GetUser",
			SQL:  "SELECT id, name, email, created_at FROM users WHERE id = $1",
		},
		{
			Name: "CreateUser",
			SQL:  "INSERT INTO users (name, email) VALUES ($1, $2) RETURNING id, name, email, created_at",
		},
		{
			Name: "ListUsers",
			SQL:  "SELECT id, name, email, created_at FROM users ORDER BY created_at DESC",
		},
	}

	// Use the existing simple_project fixture
	fixturesPath := filepath.Join("..", "fixtures", "simple_project")
	
	// Verify fixture exists
	if _, err := os.Stat(fixturesPath); os.IsNotExist(err) {
		t.Skipf("Test fixture not found at %s", fixturesPath)
	}

	// Create analysis request
	request := analyzer.AnalysisRequest{
		SQLQueries:   queries,
		GoPackages:   []string{filepath.Join(fixturesPath, "internal/db")},
		OutputFormat: "json",
		PrettyPrint:  true,
	}

	// Run analysis
	ctx := context.Background()
	result, err := a.Analyze(ctx, request)
	
	// Log what we got
	t.Logf("Analysis error: %v", err)
	if result != nil {
		t.Logf("Functions found: %d", len(result.Functions))
		for name, funcInfo := range result.Functions {
			t.Logf("Function: %s, Package: %s, File: %s", name, funcInfo.Package, funcInfo.File)
		}
		t.Logf("Tables found: %d", len(result.Tables))
		for name, tableInfo := range result.Tables {
			t.Logf("Table: %s, AccessedBy: %v", name, tableInfo.AccessedBy)
		}
		t.Logf("Dependencies: %d", len(result.Dependencies))
		for _, dep := range result.Dependencies {
			t.Logf("Dependency: %s -> %s (%s)", dep.Function, dep.Table, dep.Operation)
		}
	}

	// Check errors
	errors := a.GetErrors()
	t.Logf("Errors: %d", len(errors))
	for _, err := range errors {
		t.Logf("Error: %s - %s", err.Category, err.Message)
	}
}

func writeFile(path, content string) error {
	file, err := os.Create(path)
	if err != nil {
		return err
	}
	defer file.Close()
	
	_, err = file.WriteString(content)
	return err
}