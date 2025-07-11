package sql

import (
	"testing"

	"github.com/naoyafurudono/sqlc-use-analysis/internal/errors"
)

func TestExtractFromClause(t *testing.T) {
	analyzer := NewAnalyzer("postgresql", false, errors.NewErrorCollector(10, false))
	
	tests := []struct {
		name     string
		sql      string
		expected []string
	}{
		{
			name:     "Simple FROM",
			sql:      "SELECT * FROM users",
			expected: []string{"users"},
		},
		{
			name:     "FROM with alias",
			sql:      "SELECT u.name FROM users u",
			expected: []string{"users"},
		},
		{
			name:     "FROM with AS alias",
			sql:      "SELECT u.name FROM users AS u",
			expected: []string{"users"},
		},
		{
			name:     "FROM with WHERE",
			sql:      "SELECT * FROM users WHERE id = 1",
			expected: []string{"users"},
		},
		{
			name:     "FROM with JOIN",
			sql:      "SELECT * FROM users u JOIN posts p ON u.id = p.user_id",
			expected: []string{"users"},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := analyzer.extractFromClause(tt.sql)
			if err != nil {
				t.Errorf("extractFromClause() error = %v", err)
				return
			}
			
			t.Logf("Input: %s", tt.sql)
			t.Logf("Result: %v", result)
			t.Logf("Expected: %v", tt.expected)
			
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

func TestParseTableList(t *testing.T) {
	analyzer := NewAnalyzer("postgresql", false, errors.NewErrorCollector(10, false))
	
	tests := []struct {
		name      string
		tableList string
		expected  []string
	}{
		{
			name:      "Single table",
			tableList: "users",
			expected:  []string{"users"},
		},
		{
			name:      "Table with alias",
			tableList: "users u",
			expected:  []string{"users"},
		},
		{
			name:      "Table with AS alias",
			tableList: "users AS u",
			expected:  []string{"users"},
		},
		{
			name:      "Multiple tables",
			tableList: "users, posts",
			expected:  []string{"users", "posts"},
		},
		{
			name:      "Multiple tables with aliases",
			tableList: "users u, posts AS p",
			expected:  []string{"users", "posts"},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := analyzer.parseTableList(tt.tableList)
			
			t.Logf("Input: '%s'", tt.tableList)
			t.Logf("Result: %v", result)
			t.Logf("Expected: %v", tt.expected)
			
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