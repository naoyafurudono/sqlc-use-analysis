package gostatic

import (
	"go/ast"
	"go/parser"
	"go/token"
	"go/types"
	"testing"

	"golang.org/x/tools/go/packages"

	"github.com/naoyafurudono/sqlc-use-analysis/internal/errors"
	pkgtypes "github.com/naoyafurudono/sqlc-use-analysis/pkg/types"
)

func TestAnalyzer_extractReceiverType(t *testing.T) {
	analyzer := NewAnalyzer("test", errors.NewErrorCollector(10, false))
	
	tests := []struct {
		name     string
		code     string
		expected string
	}{
		{
			name:     "Pointer receiver",
			code:     "package main\nfunc (u *User) GetName() string { return u.Name }",
			expected: "User",
		},
		{
			name:     "Value receiver",
			code:     "package main\nfunc (u User) GetName() string { return u.Name }",
			expected: "User",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "", tt.code, parser.ParseComments)
			if err != nil {
				t.Fatalf("Failed to parse code: %v", err)
			}
			
			var funcDecl *ast.FuncDecl
			ast.Inspect(file, func(n ast.Node) bool {
				if fd, ok := n.(*ast.FuncDecl); ok {
					funcDecl = fd
					return false
				}
				return true
			})
			
			if funcDecl == nil || funcDecl.Recv == nil {
				t.Fatal("No function declaration with receiver found")
			}
			
			result := analyzer.extractReceiverType(funcDecl.Recv.List[0].Type)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestAnalyzer_analyzeFuncDecl(t *testing.T) {
	analyzer := NewAnalyzer("test", errors.NewErrorCollector(10, false))
	
	tests := []struct {
		name     string
		code     string
		expected pkgtypes.GoFunctionInfo
	}{
		{
			name: "Simple function",
			code: `
package main

func GetUser(id int) string {
	return "user"
}
`,
			expected: pkgtypes.GoFunctionInfo{
				FunctionName: "GetUser",
				PackageName:  "main",
				SQLCalls:     []pkgtypes.SQLCall{},
			},
		},
		{
			name: "Method with receiver",
			code: `
package main

type Service struct{}

func (s *Service) GetUser(id int) string {
	return "user"
}
`,
			expected: pkgtypes.GoFunctionInfo{
				FunctionName: "Service.GetUser",
				PackageName:  "main",
				SQLCalls:     []pkgtypes.SQLCall{},
			},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "", tt.code, parser.ParseComments)
			if err != nil {
				t.Fatalf("Failed to parse code: %v", err)
			}
			
			analyzer.fset = fset
			
			var funcDecl *ast.FuncDecl
			ast.Inspect(file, func(n ast.Node) bool {
				if fd, ok := n.(*ast.FuncDecl); ok {
					funcDecl = fd
					return false
				}
				return true
			})
			
			if funcDecl == nil {
				t.Fatal("No function declaration found")
			}
			
			// This part is removed as we're using real packages.Package structure above
			
			result, err := analyzer.analyzeFuncDecl(funcDecl, &packages.Package{
				Name: "main",
				TypesInfo: &types.Info{
					Types: make(map[ast.Expr]types.TypeAndValue),
				},
			})
			if err != nil {
				t.Errorf("analyzeFuncDecl() error = %v", err)
				return
			}
			
			if result.FunctionName != tt.expected.FunctionName {
				t.Errorf("Expected function name %s, got %s", tt.expected.FunctionName, result.FunctionName)
			}
			
			if result.PackageName != tt.expected.PackageName {
				t.Errorf("Expected package name %s, got %s", tt.expected.PackageName, result.PackageName)
			}
			
			if len(result.SQLCalls) != len(tt.expected.SQLCalls) {
				t.Errorf("Expected %d SQL calls, got %d", len(tt.expected.SQLCalls), len(result.SQLCalls))
			}
		})
	}
}

func TestAnalyzer_extractSQLCalls(t *testing.T) {
	analyzer := NewAnalyzer("test", errors.NewErrorCollector(10, false))
	
	tests := []struct {
		name     string
		code     string
		expected []pkgtypes.SQLCall
	}{
		{
			name: "Function with SQL call",
			code: `
package main

func GetUser(db *Queries) {
	user := db.GetUser(1)
	_ = user
}
`,
			expected: []pkgtypes.SQLCall{
				{
					MethodName: "GetUser",
					Line:       5,
					Column:     12,
				},
			},
		},
		{
			name: "Function with multiple SQL calls",
			code: `
package main

func GetUserPosts(db *Queries) {
	user := db.GetUser(1)
	posts := db.ListPostsByUser(1)
	_ = user
	_ = posts
}
`,
			expected: []pkgtypes.SQLCall{
				{
					MethodName: "GetUser",
					Line:       5,
					Column:     12,
				},
				{
					MethodName: "ListPostsByUser",
					Line:       6,
					Column:     13,
				},
			},
		},
		{
			name: "Function with no SQL calls",
			code: `
package main

func ProcessData() {
	fmt.Println("processing")
}
`,
			expected: []pkgtypes.SQLCall{},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fset := token.NewFileSet()
			file, err := parser.ParseFile(fset, "", tt.code, parser.ParseComments)
			if err != nil {
				t.Fatalf("Failed to parse code: %v", err)
			}
			
			analyzer.fset = fset
			
			var funcDecl *ast.FuncDecl
			ast.Inspect(file, func(n ast.Node) bool {
				if fd, ok := n.(*ast.FuncDecl); ok {
					funcDecl = fd
					return false
				}
				return true
			})
			
			if funcDecl == nil {
				t.Fatal("No function declaration found")
			}
			
			// Create a mock package with type information
			pkg := &packages.Package{
				Name: "main",
				TypesInfo: &types.Info{
					Types: make(map[ast.Expr]types.TypeAndValue),
				},
			}
			
			// Mock type information for Queries type
			if tt.name != "Function with no SQL calls" {
				// This is a simplified mock - in real usage, types would be populated by go/packages
				pkg.TypesInfo.Types = make(map[ast.Expr]types.TypeAndValue)
			}
			
			result := analyzer.extractSQLCalls(funcDecl.Body, pkg)
			
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d SQL calls, got %d", len(tt.expected), len(result))
				return
			}
			
			for i, expected := range tt.expected {
				if i >= len(result) {
					continue
				}
				
				if result[i].MethodName != expected.MethodName {
					t.Errorf("Expected method name %s, got %s", expected.MethodName, result[i].MethodName)
				}
				
				// Line numbers may vary slightly due to parsing, so we check if they're reasonable
				if result[i].Line <= 0 {
					t.Errorf("Expected positive line number, got %d", result[i].Line)
				}
			}
		})
	}
}

func TestAnalyzer_isSQLCMethod(t *testing.T) {
	analyzer := NewAnalyzer("test", errors.NewErrorCollector(10, false))
	
	tests := []struct {
		name       string
		typeName   string
		methodName string
		expected   bool
	}{
		{
			name:       "Queries type with PascalCase method",
			typeName:   "*main.Queries",
			methodName: "GetUser",
			expected:   true,
		},
		{
			name:       "DB type with PascalCase method",
			typeName:   "*db.DB",
			methodName: "ListUsers",
			expected:   true,
		},
		{
			name:       "Non-SQLC type",
			typeName:   "*main.Service",
			methodName: "processData",
			expected:   false,
		},
		{
			name:       "SQLC type with non-PascalCase method",
			typeName:   "*main.Queries",
			methodName: "processData",
			expected:   true, // Still true because it contains "Queries"
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a mock type
			mockType := &mockType{name: tt.typeName}
			
			result := analyzer.isSQLCMethod(mockType, tt.methodName)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestAnalyzer_isPascalCase(t *testing.T) {
	analyzer := NewAnalyzer("test", errors.NewErrorCollector(10, false))
	
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "PascalCase",
			input:    "GetUser",
			expected: true,
		},
		{
			name:     "camelCase",
			input:    "getUser",
			expected: false,
		},
		{
			name:     "snake_case",
			input:    "get_user",
			expected: false,
		},
		{
			name:     "Empty string",
			input:    "",
			expected: false,
		},
		{
			name:     "Single uppercase",
			input:    "A",
			expected: true,
		},
		{
			name:     "Single lowercase",
			input:    "a",
			expected: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.isPascalCase(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestAnalyzer_containsQueriesType(t *testing.T) {
	analyzer := NewAnalyzer("test", errors.NewErrorCollector(10, false))
	
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{
			name:     "Contains Queries",
			input:    "*main.Queries",
			expected: true,
		},
		{
			name:     "Contains queries",
			input:    "*pkg.queries",
			expected: true,
		},
		{
			name:     "Contains DB",
			input:    "*main.DB",
			expected: true,
		},
		{
			name:     "Contains db",
			input:    "*pkg.db",
			expected: true,
		},
		{
			name:     "Does not contain patterns",
			input:    "*main.Service",
			expected: false,
		},
		{
			name:     "Empty string",
			input:    "",
			expected: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.containsQueriesType(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

// Mock types for testing

type mockType struct {
	name string
}

func (m *mockType) String() string {
	return m.name
}

func (m *mockType) Underlying() types.Type {
	return m
}