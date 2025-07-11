package dependency

import (
	"testing"

	"github.com/naoyafurudono/sqlc-use-analysis/internal/errors"
	"github.com/naoyafurudono/sqlc-use-analysis/pkg/types"
)

func TestEngine_ValidateInput(t *testing.T) {
	engine := NewEngine(errors.NewErrorCollector(10, false))
	
	tests := []struct {
		name         string
		queries      []types.QueryInfo
		packagePaths []string
		wantErr      bool
	}{
		{
			name: "Valid input",
			queries: []types.QueryInfo{
				{Name: "GetUser", SQL: "SELECT * FROM users WHERE id = $1"},
			},
			packagePaths: []string{"./test"},
			wantErr:      false,
		},
		{
			name:         "Empty queries",
			queries:      []types.QueryInfo{},
			packagePaths: []string{"./test"},
			wantErr:      true,
		},
		{
			name: "Empty package paths",
			queries: []types.QueryInfo{
				{Name: "GetUser", SQL: "SELECT * FROM users WHERE id = $1"},
			},
			packagePaths: []string{},
			wantErr:      true,
		},
		{
			name: "Query with empty name",
			queries: []types.QueryInfo{
				{Name: "", SQL: "SELECT * FROM users WHERE id = $1"},
			},
			packagePaths: []string{"./test"},
			wantErr:      true,
		},
		{
			name: "Query with empty SQL",
			queries: []types.QueryInfo{
				{Name: "GetUser", SQL: ""},
			},
			packagePaths: []string{"./test"},
			wantErr:      true,
		},
		{
			name: "Invalid package path",
			queries: []types.QueryInfo{
				{Name: "GetUser", SQL: "SELECT * FROM users WHERE id = $1"},
			},
			packagePaths: []string{"../../../dangerous/path"},
			wantErr:      true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := engine.ValidateInput(tt.queries, tt.packagePaths)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateInput() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestEngine_analyzeSQLQueries(t *testing.T) {
	engine := NewEngine(errors.NewErrorCollector(10, false))
	
	queries := []types.QueryInfo{
		{
			Name: "GetUser",
			SQL:  "SELECT id, name FROM users WHERE id = $1",
		},
		{
			Name: "ListUsers",
			SQL:  "SELECT id, name FROM users ORDER BY id",
		},
		{
			Name: "CreateUser",
			SQL:  "INSERT INTO users (name) VALUES ($1)",
		},
	}
	
	result, err := engine.analyzeSQLQueries(queries)
	if err != nil {
		t.Errorf("analyzeSQLQueries() error = %v", err)
		return
	}
	
	t.Logf("SQL analysis result: %+v", result)
	t.Logf("Number of methods found: %d", len(result))
	
	if len(result) != 3 {
		t.Errorf("Expected 3 methods, got %d", len(result))
	}
	
	// Check Getuser method (note: PascalCase conversion makes it lowercase)
	if method, exists := result["Getuser"]; exists {
		if method.MethodName != "Getuser" {
			t.Errorf("Expected method name 'Getuser', got '%s'", method.MethodName)
		}
		if len(method.Tables) != 1 {
			t.Errorf("Expected 1 table for Getuser, got %d", len(method.Tables))
		}
		if len(method.Tables) > 0 && method.Tables[0].TableName != "users" {
			t.Errorf("Expected table 'users', got '%s'", method.Tables[0].TableName)
		}
	} else {
		t.Error("Getuser method not found")
	}
	
	// Check Createuser method
	if method, exists := result["Createuser"]; exists {
		if method.MethodName != "Createuser" {
			t.Errorf("Expected method name 'Createuser', got '%s'", method.MethodName)
		}
		if len(method.Tables) != 1 {
			t.Errorf("Expected 1 table for Createuser, got %d", len(method.Tables))
		}
		if len(method.Tables) > 0 && method.Tables[0].TableName != "users" {
			t.Errorf("Expected table 'users', got '%s'", method.Tables[0].TableName)
		}
	} else {
		t.Error("Createuser method not found")
	}
}

func TestEngine_GetStats(t *testing.T) {
	engine := NewEngine(errors.NewErrorCollector(10, false))
	
	// Add some errors to test stats
	err1 := errors.NewError(errors.CategoryAnalysis, errors.SeverityError, "test error 1")
	err2 := errors.NewError(errors.CategoryMapping, errors.SeverityWarning, "test warning 1")
	
	engine.errorCollector.Add(err1)
	engine.errorCollector.Add(err2)
	
	stats := engine.GetStats()
	
	if stats.ErrorCount != 2 {
		t.Errorf("Expected error count 2, got %d", stats.ErrorCount)
	}
	
	if !stats.HasErrors {
		t.Error("Expected HasErrors to be true")
	}
	
	if !stats.HasWarnings {
		t.Error("Expected HasWarnings to be true")
	}
	
	if len(stats.ErrorsByCategory) != 2 {
		t.Errorf("Expected 2 error categories, got %d", len(stats.ErrorsByCategory))
	}
}

func TestEngine_SetMaxErrors(t *testing.T) {
	engine := NewEngine(errors.NewErrorCollector(10, false))
	
	engine.SetMaxErrors(5)
	
	stats := engine.GetStats()
	if stats.ErrorCount != 0 {
		t.Errorf("Expected error count 0 after reset, got %d", stats.ErrorCount)
	}
}

func TestEngine_Reset(t *testing.T) {
	engine := NewEngine(errors.NewErrorCollector(10, false))
	
	// Add an error
	err := errors.NewError(errors.CategoryAnalysis, errors.SeverityError, "test error")
	engine.errorCollector.Add(err)
	
	// Check error was added
	if engine.GetStats().ErrorCount != 1 {
		t.Errorf("Expected error count 1 before reset, got %d", engine.GetStats().ErrorCount)
	}
	
	// Reset
	engine.Reset()
	
	// Check error was cleared
	if engine.GetStats().ErrorCount != 0 {
		t.Errorf("Expected error count 0 after reset, got %d", engine.GetStats().ErrorCount)
	}
}

func TestEngine_isValidPackagePath(t *testing.T) {
	tests := []struct {
		name string
		path string
		want bool
	}{
		{
			name: "Current directory",
			path: ".",
			want: true,
		},
		{
			name: "Relative path",
			path: "./internal/...",
			want: true,
		},
		{
			name: "Absolute path",
			path: "/usr/local/src/project",
			want: true,
		},
		{
			name: "Go module path",
			path: "github.com/user/project",
			want: true,
		},
		{
			name: "Invalid path with ..",
			path: "../../dangerous",
			want: false,
		},
		{
			name: "Empty path",
			path: "",
			want: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isValidPackagePath(tt.path)
			if result != tt.want {
				t.Errorf("isValidPackagePath(%q) = %v, want %v", tt.path, result, tt.want)
			}
		})
	}
}

func TestEngine_EnableDebugMode(t *testing.T) {
	engine := NewEngine(errors.NewErrorCollector(10, false))
	
	// Initially debug mode should be false
	if engine.errorCollector.IsDebugMode() {
		t.Error("Expected debug mode to be false initially")
	}
	
	// Enable debug mode
	engine.EnableDebugMode()
	
	// Check debug mode is enabled
	if !engine.errorCollector.IsDebugMode() {
		t.Error("Expected debug mode to be true after enabling")
	}
}

// Test helper functions

func createTestQueries() []types.QueryInfo {
	return []types.QueryInfo{
		{
			Name: "GetUser",
			SQL:  "SELECT id, name FROM users WHERE id = $1",
		},
		{
			Name: "ListUsers",
			SQL:  "SELECT id, name FROM users ORDER BY id",
		},
		{
			Name: "CreateUser",
			SQL:  "INSERT INTO users (name) VALUES ($1)",
		},
		{
			Name: "UpdateUser",
			SQL:  "UPDATE users SET name = $2 WHERE id = $1",
		},
		{
			Name: "DeleteUser",
			SQL:  "DELETE FROM users WHERE id = $1",
		},
	}
}

func createTestPackagePaths() []string {
	return []string{
		"./test",
		"./internal/test",
	}
}

// Integration test with mock data
func TestEngine_IntegrationTest(t *testing.T) {
	engine := NewEngine(errors.NewErrorCollector(100, false))
	
	queries := createTestQueries()
	packagePaths := createTestPackagePaths()
	
	// Validate input
	err := engine.ValidateInput(queries, packagePaths)
	if err != nil {
		t.Errorf("ValidateInput() error = %v", err)
		return
	}
	
	// Analyze SQL queries
	sqlMethods, err := engine.analyzeSQLQueries(queries)
	if err != nil {
		t.Errorf("analyzeSQLQueries() error = %v", err)
		return
	}
	
	if len(sqlMethods) != len(queries) {
		t.Errorf("Expected %d SQL methods, got %d", len(queries), len(sqlMethods))
	}
	
	// Check that all expected methods are present (note: PascalCase conversion)
	expectedMethods := []string{"Getuser", "Listusers", "Createuser", "Updateuser", "Deleteuser"}
	for _, expected := range expectedMethods {
		if _, exists := sqlMethods[expected]; !exists {
			t.Errorf("Expected method '%s' not found", expected)
		}
	}
}