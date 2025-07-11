package sql

import (
	"testing"

	"github.com/naoyafurudono/sqlc-use-analysis/internal/errors"
	"github.com/naoyafurudono/sqlc-use-analysis/pkg/types"
)

func TestAnalyzer_detectOperationType(t *testing.T) {
	analyzer := NewAnalyzer("postgresql", false, errors.NewErrorCollector(10, false))
	
	tests := []struct {
		name     string
		sql      string
		expected types.Operation
		wantErr  bool
	}{
		{
			name:     "Simple SELECT",
			sql:      "SELECT * FROM users",
			expected: types.OpSelect,
			wantErr:  false,
		},
		{
			name:     "INSERT with VALUES",
			sql:      "INSERT INTO users (name, email) VALUES ($1, $2)",
			expected: types.OpInsert,
			wantErr:  false,
		},
		{
			name:     "UPDATE with WHERE",
			sql:      "UPDATE users SET name = $1 WHERE id = $2",
			expected: types.OpUpdate,
			wantErr:  false,
		},
		{
			name:     "DELETE with WHERE",
			sql:      "DELETE FROM users WHERE id = $1",
			expected: types.OpDelete,
			wantErr:  false,
		},
		{
			name:     "CTE with SELECT",
			sql:      "WITH active_users AS (SELECT * FROM users WHERE active = true) SELECT * FROM active_users",
			expected: types.OpSelect,
			wantErr:  false,
		},
		{
			name:     "Complex SELECT with JOIN",
			sql:      `SELECT u.name, p.title 
			          FROM users u 
			          LEFT JOIN posts p ON u.id = p.user_id`,
			expected: types.OpSelect,
			wantErr:  false,
		},
		{
			name:    "Unknown operation",
			sql:     "CREATE TABLE users (id INT)",
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := analyzer.detectOperationType(tt.sql)
			
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("detectOperationType() error = %v", err)
				return
			}
			
			if result != tt.expected {
				t.Errorf("Expected operation %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestAnalyzer_extractTablesFromSelect(t *testing.T) {
	analyzer := NewAnalyzer("postgresql", false, errors.NewErrorCollector(10, false))
	
	tests := []struct {
		name     string
		sql      string
		expected []string
	}{
		{
			name:     "Simple SELECT",
			sql:      "SELECT * FROM users",
			expected: []string{"users"},
		},
		{
			name:     "SELECT with alias",
			sql:      "SELECT u.name FROM users u",
			expected: []string{"users"},
		},
		{
			name:     "SELECT with AS alias",
			sql:      "SELECT u.name FROM users AS u",
			expected: []string{"users"},
		},
		{
			name:     "SELECT with multiple tables",
			sql:      "SELECT * FROM users, posts",
			expected: []string{"users", "posts"},
		},
		{
			name:     "SELECT with INNER JOIN",
			sql:      "SELECT u.name, p.title FROM users u INNER JOIN posts p ON u.id = p.user_id",
			expected: []string{"users", "posts"},
		},
		{
			name:     "SELECT with LEFT JOIN",
			sql:      "SELECT u.name, p.title FROM users u LEFT JOIN posts p ON u.id = p.user_id",
			expected: []string{"users", "posts"},
		},
		{
			name:     "SELECT with multiple JOINs",
			sql:      `SELECT u.name, p.title, c.content 
			          FROM users u 
			          LEFT JOIN posts p ON u.id = p.user_id 
			          RIGHT JOIN comments c ON p.id = c.post_id`,
			expected: []string{"users", "posts", "comments"},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := analyzer.extractTablesFromSelect(tt.sql)
			if err != nil {
				t.Errorf("extractTablesFromSelect() error = %v", err)
				return
			}
			
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d tables, got %d: %v", len(tt.expected), len(result), result)
				return
			}
			
			// 順序は関係ないので、全ての期待値が含まれているかチェック
			for _, expected := range tt.expected {
				found := false
				for _, actual := range result {
					if actual == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected table '%s' not found in result: %v", expected, result)
				}
			}
		})
	}
}

func TestAnalyzer_extractTablesFromInsert(t *testing.T) {
	analyzer := NewAnalyzer("postgresql", false, errors.NewErrorCollector(10, false))
	
	tests := []struct {
		name     string
		sql      string
		expected []string
		wantErr  bool
	}{
		{
			name:     "Simple INSERT",
			sql:      "INSERT INTO users (name, email) VALUES ($1, $2)",
			expected: []string{"users"},
			wantErr:  false,
		},
		{
			name:     "INSERT with schema",
			sql:      "INSERT INTO public.users (name) VALUES ($1)",
			expected: []string{"public.users"},
			wantErr:  false,
		},
		{
			name:    "Invalid INSERT",
			sql:     "INSERT VALUES ($1, $2)",
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := analyzer.extractTablesFromInsert(tt.sql)
			
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("extractTablesFromInsert() error = %v", err)
				return
			}
			
			if len(result) != len(tt.expected) || result[0] != tt.expected[0] {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}

func TestAnalyzer_extractTablesFromUpdate(t *testing.T) {
	analyzer := NewAnalyzer("postgresql", false, errors.NewErrorCollector(10, false))
	
	tests := []struct {
		name     string
		sql      string
		expected []string
		wantErr  bool
	}{
		{
			name:     "Simple UPDATE",
			sql:      "UPDATE users SET name = $1 WHERE id = $2",
			expected: []string{"users"},
			wantErr:  false,
		},
		{
			name:     "UPDATE with FROM",
			sql:      "UPDATE users SET name = p.title FROM posts p WHERE users.id = p.user_id",
			expected: []string{"users", "posts"},
			wantErr:  false,
		},
		{
			name:     "UPDATE with JOIN",
			sql:      "UPDATE users SET name = p.title FROM users INNER JOIN posts p ON users.id = p.user_id",
			expected: []string{"users", "posts"},
			wantErr:  false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := analyzer.extractTablesFromUpdate(tt.sql)
			
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("extractTablesFromUpdate() error = %v", err)
				return
			}
			
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d tables, got %d: %v", len(tt.expected), len(result), result)
				return
			}
			
			for _, expected := range tt.expected {
				found := false
				for _, actual := range result {
					if actual == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected table '%s' not found in result: %v", expected, result)
				}
			}
		})
	}
}

func TestAnalyzer_extractTablesFromDelete(t *testing.T) {
	analyzer := NewAnalyzer("postgresql", false, errors.NewErrorCollector(10, false))
	
	tests := []struct {
		name     string
		sql      string
		expected []string
		wantErr  bool
	}{
		{
			name:     "Simple DELETE",
			sql:      "DELETE FROM users WHERE id = $1",
			expected: []string{"users"},
			wantErr:  false,
		},
		{
			name:     "DELETE with USING",
			sql:      "DELETE FROM users USING posts WHERE users.id = posts.user_id",
			expected: []string{"users", "posts"},
			wantErr:  false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := analyzer.extractTablesFromDelete(tt.sql)
			
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error, but got none")
				}
				return
			}
			
			if err != nil {
				t.Errorf("extractTablesFromDelete() error = %v", err)
				return
			}
			
			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d tables, got %d: %v", len(tt.expected), len(result), result)
				return
			}
			
			for _, expected := range tt.expected {
				found := false
				for _, actual := range result {
					if actual == expected {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Expected table '%s' not found in result: %v", expected, result)
				}
			}
		})
	}
}

func TestAnalyzer_generateMethodName(t *testing.T) {
	analyzer := NewAnalyzer("postgresql", false, errors.NewErrorCollector(10, false))
	
	tests := []struct {
		name      string
		queryName string
		cmd       string
		expected  string
	}{
		{
			name:      "Simple query name",
			queryName: "get_user",
			cmd:       ":one",
			expected:  "GetUser",
		},
		{
			name:      "Many query",
			queryName: "list_user",
			cmd:       ":many",
			expected:  "ListUsers",
		},
		{
			name:      "Query ending with y",
			queryName: "get_company",
			cmd:       ":many",
			expected:  "GetCompanies",
		},
		{
			name:      "Already plural",
			queryName: "get_users",
			cmd:       ":many",
			expected:  "GetUsers",
		},
		{
			name:      "With numbers",
			queryName: "get_user2",
			cmd:       ":one",
			expected:  "GetUser2",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.generateMethodName(tt.queryName, tt.cmd)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestAnalyzer_AnalyzeQuery(t *testing.T) {
	analyzer := NewAnalyzer("postgresql", false, errors.NewErrorCollector(10, false))
	
	query := Query{
		Text:     "SELECT * FROM users WHERE id = $1",
		Name:     "get_user",
		Cmd:      ":one",
		Filename: "queries/users.sql",
	}
	
	result, err := analyzer.AnalyzeQuery(query)
	if err != nil {
		t.Errorf("AnalyzeQuery() error = %v", err)
		return
	}
	
	if result.MethodName != "GetUser" {
		t.Errorf("Expected method name 'GetUser', got '%s'", result.MethodName)
	}
	
	if len(result.Tables) != 1 {
		t.Errorf("Expected 1 table, got %d", len(result.Tables))
		return
	}
	
	table := result.Tables[0]
	if table.TableName != "users" {
		t.Errorf("Expected table 'users', got '%s'", table.TableName)
	}
	
	if len(table.Operations) != 1 || table.Operations[0] != "SELECT" {
		t.Errorf("Expected operations ['SELECT'], got %v", table.Operations)
	}
}