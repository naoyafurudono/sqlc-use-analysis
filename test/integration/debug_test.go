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

	// Load all queries to match the fixture
	queries := []analyzer.Query{
		{Name: "GetUser", SQL: "SELECT id, name, email, created_at FROM users WHERE id = $1"},
		{Name: "CreateUser", SQL: "INSERT INTO users (name, email) VALUES ($1, $2) RETURNING id, name, email, created_at"},
		{Name: "ListUsers", SQL: "SELECT id, name, email, created_at FROM users ORDER BY created_at DESC"},
		{Name: "GetPost", SQL: "SELECT p.id, p.title, p.content, p.author_id, p.created_at, u.name as author_name FROM posts p JOIN users u ON p.author_id = u.id WHERE p.id = $1"},
		{Name: "ListPostsByUser", SQL: "SELECT id, title, content, author_id, created_at FROM posts WHERE author_id = $1 ORDER BY created_at DESC"},
		{Name: "CreatePost", SQL: "INSERT INTO posts (title, content, author_id) VALUES ($1, $2, $3) RETURNING id, title, content, author_id, created_at"},
		{Name: "GetCommentsByPost", SQL: "SELECT c.id, c.content, c.author_id, c.created_at, u.name as author_name FROM comments c JOIN users u ON c.author_id = u.id WHERE c.post_id = $1 ORDER BY c.created_at"},
		{Name: "CreateComment", SQL: "INSERT INTO comments (post_id, author_id, content) VALUES ($1, $2, $3) RETURNING id, post_id, author_id, content, created_at"},
	}

	// Use the existing simple_project fixture
	fixturesPath := filepath.Join("..", "fixtures", "simple_project")
	
	// Verify fixture exists
	if _, err := os.Stat(fixturesPath); os.IsNotExist(err) {
		t.Skipf("Test fixture not found at %s", fixturesPath)
	}

	// Create analysis request (include all packages)
	request := analyzer.AnalysisRequest{
		SQLQueries: queries,
		GoPackages: []string{
			filepath.Join(fixturesPath, "internal/db"),
			filepath.Join(fixturesPath, "internal/service"),
			filepath.Join(fixturesPath, "internal/handler"),
		},
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
			// Log table access details for functions
			for tableName, access := range funcInfo.TableAccess {
				t.Logf("  - Table: %s, Operations: %v, Methods: %v", tableName, access.Operations, access.Methods)
			}
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
	for i, err := range errors {
		if i < 10 { // Limit to first 10 errors to avoid spam
			t.Logf("Error: %s - %s", err.Category, err.Message)
			if err.Details != nil {
				if method, ok := err.Details["method"]; ok {
					t.Logf("  Method: %v", method)
				}
				if function, ok := err.Details["function"]; ok {
					t.Logf("  Function: %v", function)
				}
			}
		}
	}
	if len(errors) > 10 {
		t.Logf("... and %d more errors", len(errors)-10)
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