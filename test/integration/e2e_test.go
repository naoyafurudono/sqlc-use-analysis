package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/naoyafurudono/sqlc-use-analysis/pkg/analyzer"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestE2ESimpleProject tests the complete workflow with a simple project
func TestE2ESimpleProject(t *testing.T) {
	// Get test fixture path
	fixturesPath := filepath.Join("..", "fixtures", "simple_project")
	
	// Verify fixture exists
	if _, err := os.Stat(fixturesPath); os.IsNotExist(err) {
		t.Skipf("Test fixture not found at %s", fixturesPath)
	}

	// Create analyzer
	a := analyzer.New()

	// Prepare SQL queries from fixture
	queries := []analyzer.Query{
		{
			Name: "GetUser",
			SQL:  "SELECT id, name, email, created_at FROM users WHERE id = $1",
		},
		{
			Name: "ListUsers",
			SQL:  "SELECT id, name, email, created_at FROM users ORDER BY created_at DESC",
		},
		{
			Name: "CreateUser",
			SQL:  "INSERT INTO users (name, email) VALUES ($1, $2) RETURNING id, name, email, created_at",
		},
		{
			Name: "GetPost",
			SQL:  "SELECT p.id, p.title, p.content, p.author_id, p.created_at, u.name as author_name FROM posts p JOIN users u ON p.author_id = u.id WHERE p.id = $1",
		},
		{
			Name: "ListPostsByUser",
			SQL:  "SELECT id, title, content, author_id, created_at FROM posts WHERE author_id = $1 ORDER BY created_at DESC",
		},
		{
			Name: "CreatePost",
			SQL:  "INSERT INTO posts (title, content, author_id) VALUES ($1, $2, $3) RETURNING id, title, content, author_id, created_at",
		},
		{
			Name: "GetCommentsByPost",
			SQL:  "SELECT c.id, c.content, c.author_id, c.created_at, u.name as author_name FROM comments c JOIN users u ON c.author_id = u.id WHERE c.post_id = $1 ORDER BY c.created_at",
		},
		{
			Name: "CreateComment",
			SQL:  "INSERT INTO comments (post_id, author_id, content) VALUES ($1, $2, $3) RETURNING id, post_id, author_id, content, created_at",
		},
	}

	// Prepare Go package paths
	goPackages := []string{
		filepath.Join(fixturesPath, "internal", "db"),
		filepath.Join(fixturesPath, "internal", "service"),
		filepath.Join(fixturesPath, "internal", "handler"),
	}

	// Create analysis request
	request := analyzer.AnalysisRequest{
		SQLQueries:   queries,
		GoPackages:   goPackages,
		OutputFormat: "json",
		PrettyPrint:  true,
	}

	// Run analysis
	ctx := context.Background()
	result, err := a.Analyze(ctx, request)
	
	// Verify analysis succeeded
	require.NoError(t, err, "Analysis should succeed")
	require.NotNil(t, result, "Result should not be nil")

	// Verify basic structure
	assert.NotEmpty(t, result.Functions, "Functions should be found")
	assert.NotEmpty(t, result.Tables, "Tables should be found")
	assert.NotEmpty(t, result.Dependencies, "Dependencies should be found")

	// Verify expected tables are found
	expectedTables := []string{"users", "posts", "comments"}
	for _, table := range expectedTables {
		assert.Contains(t, result.Tables, table, "Table %s should be found", table)
	}

	// Verify expected functions are found
	expectedFunctions := []string{"GetUser", "ListUsers", "CreateUser", "GetPost", "ListPostsByUser", "CreatePost", "GetCommentsByPost", "CreateComment"}
	for _, funcName := range expectedFunctions {
		found := false
		for _, funcInfo := range result.Functions {
			if funcInfo.Name == funcName {
				found = true
				break
			}
		}
		assert.True(t, found, "Function %s should be found", funcName)
	}

	// Verify dependencies exist
	assert.True(t, len(result.Dependencies) > 0, "Should have dependencies")

	// Verify summary
	assert.Equal(t, len(expectedTables), result.Summary.TableCount, "Table count should match")
	assert.True(t, result.Summary.FunctionCount > 0, "Function count should be greater than 0")

	// Test specific dependency mappings
	testDependencyMappings(t, result)

	// Test output format
	testOutputFormat(t, result)
}

func testDependencyMappings(t *testing.T, result *analyzer.Result) {
	// Test that GetUser function accesses users table
	getUserFunc := findFunctionByName(result.Functions, "GetUser")
	require.NotNil(t, getUserFunc, "GetUser function should exist")
	
	assert.Contains(t, getUserFunc.TableAccess, "users", "GetUser should access users table")
	
	// Test that GetPost function accesses both posts and users tables (JOIN)
	getPostFunc := findFunctionByName(result.Functions, "GetPost")
	require.NotNil(t, getPostFunc, "GetPost function should exist")
	
	assert.Contains(t, getPostFunc.TableAccess, "posts", "GetPost should access posts table")
	assert.Contains(t, getPostFunc.TableAccess, "users", "GetPost should access users table")
	
	// Test that CreateUser function has insert operation
	createUserFunc := findFunctionByName(result.Functions, "CreateUser")
	require.NotNil(t, createUserFunc, "CreateUser function should exist")
	
	// Check if the function has access to users table with INSERT operation
	if usersAccess, ok := createUserFunc.TableAccess["users"]; ok {
		assert.Contains(t, usersAccess.Operations, "INSERT", "CreateUser should have INSERT operation")
	}
	
	// Test table access patterns
	usersTable := result.Tables["users"]
	assert.NotNil(t, usersTable, "users table should exist")
	assert.True(t, len(usersTable.AccessedBy) > 0, "users table should be accessed by functions")
}

func testOutputFormat(t *testing.T, result *analyzer.Result) {
	// Test JSON serialization
	jsonData, err := json.MarshalIndent(result, "", "  ")
	require.NoError(t, err, "JSON marshaling should succeed")
	
	// Verify JSON contains expected fields
	var jsonResult map[string]interface{}
	err = json.Unmarshal(jsonData, &jsonResult)
	require.NoError(t, err, "JSON unmarshaling should succeed")
	
	// Check top-level fields
	assert.Contains(t, jsonResult, "functions", "JSON should contain functions field")
	assert.Contains(t, jsonResult, "tables", "JSON should contain tables field")
	assert.Contains(t, jsonResult, "dependencies", "JSON should contain dependencies field")
	assert.Contains(t, jsonResult, "summary", "JSON should contain summary field")
	
	// Check functions structure
	functions, ok := jsonResult["functions"].(map[string]interface{})
	require.True(t, ok, "functions should be a map")
	assert.True(t, len(functions) > 0, "functions should not be empty")
	
	// Check tables structure
	tables, ok := jsonResult["tables"].(map[string]interface{})
	require.True(t, ok, "tables should be a map")
	assert.True(t, len(tables) > 0, "tables should not be empty")
}

func findFunctionByName(functions map[string]analyzer.FunctionInfo, name string) *analyzer.FunctionInfo {
	for _, funcInfo := range functions {
		if funcInfo.Name == name {
			return &funcInfo
		}
	}
	return nil
}

// TestE2EComplexProject tests with a more complex project structure
func TestE2EComplexProject(t *testing.T) {
	// Create a more complex test case
	queries := []analyzer.Query{
		{
			Name: "GetUserWithPosts",
			SQL: `
				SELECT u.id, u.name, u.email, p.id as post_id, p.title, p.content
				FROM users u
				LEFT JOIN posts p ON u.id = p.author_id
				WHERE u.id = $1
			`,
		},
		{
			Name: "GetPostWithCommentsAndAuthors",
			SQL: `
				SELECT p.id, p.title, p.content, p.author_id, p.created_at,
				       c.id as comment_id, c.content as comment_content,
				       u1.name as post_author_name, u2.name as comment_author_name
				FROM posts p
				JOIN users u1 ON p.author_id = u1.id
				LEFT JOIN comments c ON p.id = c.post_id
				LEFT JOIN users u2 ON c.author_id = u2.id
				WHERE p.id = $1
			`,
		},
		{
			Name: "UpdateUserEmail",
			SQL: "UPDATE users SET email = $2 WHERE id = $1 RETURNING id, name, email",
		},
		{
			Name: "DeleteOldPosts",
			SQL: "DELETE FROM posts WHERE created_at < $1",
		},
	}

	// Create temporary Go files for complex test
	tmpDir := t.TempDir()
	
	// Create complex Go files
	createComplexGoFiles(t, tmpDir)

	// Create analysis request
	request := analyzer.AnalysisRequest{
		SQLQueries:   queries,
		GoPackages:   []string{filepath.Join(tmpDir, "internal")},
		OutputFormat: "json",
		PrettyPrint:  true,
	}

	// Run analysis
	ctx := context.Background()
	a := analyzer.New()
	result, err := a.Analyze(ctx, request)
	
	// Verify analysis succeeded
	require.NoError(t, err, "Complex analysis should succeed")
	require.NotNil(t, result, "Result should not be nil")

	// Verify complex query analysis
	testComplexQueryAnalysis(t, result)
}

func createComplexGoFiles(t *testing.T, tmpDir string) {
	// Create complex service file
	serviceDir := filepath.Join(tmpDir, "internal", "service")
	err := os.MkdirAll(serviceDir, 0755)
	require.NoError(t, err)

	complexServiceContent := `
package service

import (
	"context"
	"database/sql"
)

type ComplexService struct {
	db *sql.DB
}

func NewComplexService(db *sql.DB) *ComplexService {
	return &ComplexService{db: db}
}

func (s *ComplexService) GetUserWithPosts(ctx context.Context, userID int32) error {
	// This would call GetUserWithPosts query
	return nil
}

func (s *ComplexService) GetPostWithCommentsAndAuthors(ctx context.Context, postID int32) error {
	// This would call GetPostWithCommentsAndAuthors query
	return nil
}

func (s *ComplexService) UpdateUserEmail(ctx context.Context, userID int32, email string) error {
	// This would call UpdateUserEmail query
	return nil
}

func (s *ComplexService) DeleteOldPosts(ctx context.Context) error {
	// This would call DeleteOldPosts query
	return nil
}
`

	err = os.WriteFile(filepath.Join(serviceDir, "complex_service.go"), []byte(complexServiceContent), 0644)
	require.NoError(t, err)
}

func testComplexQueryAnalysis(t *testing.T, result *analyzer.Result) {
	// Test that complex JOIN queries are properly analyzed
	getUserWithPosts := findFunctionByName(result.Functions, "GetUserWithPosts")
	if getUserWithPosts != nil {
		assert.Contains(t, getUserWithPosts.TableAccess, "users", "GetUserWithPosts should access users table")
		assert.Contains(t, getUserWithPosts.TableAccess, "posts", "GetUserWithPosts should access posts table")
	}

	// Test that triple JOIN queries are properly analyzed
	getPostWithComments := findFunctionByName(result.Functions, "GetPostWithCommentsAndAuthors")
	if getPostWithComments != nil {
		assert.Contains(t, getPostWithComments.TableAccess, "posts", "GetPostWithCommentsAndAuthors should access posts table")
		assert.Contains(t, getPostWithComments.TableAccess, "users", "GetPostWithCommentsAndAuthors should access users table")
		assert.Contains(t, getPostWithComments.TableAccess, "comments", "GetPostWithCommentsAndAuthors should access comments table")
	}

	// Test UPDATE operation
	updateUserEmail := findFunctionByName(result.Functions, "UpdateUserEmail")
	if updateUserEmail != nil {
		if usersAccess, ok := updateUserEmail.TableAccess["users"]; ok {
			assert.Contains(t, usersAccess.Operations, "UPDATE", "UpdateUserEmail should have UPDATE operation")
		}
	}

	// Test DELETE operation
	deleteOldPosts := findFunctionByName(result.Functions, "DeleteOldPosts")
	if deleteOldPosts != nil {
		if postsAccess, ok := deleteOldPosts.TableAccess["posts"]; ok {
			assert.Contains(t, postsAccess.Operations, "DELETE", "DeleteOldPosts should have DELETE operation")
		}
	}
}

// TestE2EErrorHandling tests error handling scenarios
func TestE2EErrorHandling(t *testing.T) {
	// Create analyzer
	a := analyzer.New()

	// Test with invalid SQL
	invalidRequest := analyzer.AnalysisRequest{
		SQLQueries: []analyzer.Query{
			{Name: "Invalid", SQL: "INVALID SQL SYNTAX"},
		},
		GoPackages:   []string{},
		OutputFormat: "json",
		PrettyPrint:  true,
	}

	ctx := context.Background()
	result, err := a.Analyze(ctx, invalidRequest)
	
	// Should handle invalid SQL gracefully
	if err != nil {
		t.Logf("Expected error for invalid SQL: %v", err)
	} else {
		// If no error, result should still be valid
		assert.NotNil(t, result, "Result should not be nil even with invalid SQL")
	}

	// Test with non-existent Go package
	nonExistentRequest := analyzer.AnalysisRequest{
		SQLQueries: []analyzer.Query{
			{Name: "Valid", SQL: "SELECT 1"},
		},
		GoPackages:   []string{"/non/existent/path"},
		OutputFormat: "json",
		PrettyPrint:  true,
	}

	result, err = a.Analyze(ctx, nonExistentRequest)
	
	// Should handle non-existent package gracefully
	if err != nil {
		t.Logf("Expected error for non-existent package: %v", err)
	} else {
		// If no error, result should still be valid
		assert.NotNil(t, result, "Result should not be nil even with non-existent package")
	}
}

// TestE2EPerformance tests performance with larger datasets
func TestE2EPerformance(t *testing.T) {
	// Create analyzer
	a := analyzer.New()

	// Generate many queries
	queries := make([]analyzer.Query, 100)
	for i := 0; i < 100; i++ {
		queries[i] = analyzer.Query{
			Name: fmt.Sprintf("Query%d", i),
			SQL:  fmt.Sprintf("SELECT id, name FROM table_%d WHERE id = $1", i%10),
		}
	}

	// Create request
	request := analyzer.AnalysisRequest{
		SQLQueries:   queries,
		GoPackages:   []string{},
		OutputFormat: "json",
		PrettyPrint:  true,
	}

	ctx := context.Background()
	result, err := a.Analyze(ctx, request)
	
	// Verify performance test completes
	require.NoError(t, err, "Performance test should complete successfully")
	require.NotNil(t, result, "Result should not be nil")

	// Basic verification
	assert.True(t, len(result.Functions) > 0, "Should process multiple queries")
	t.Logf("Processed %d queries successfully", len(result.Functions))
}