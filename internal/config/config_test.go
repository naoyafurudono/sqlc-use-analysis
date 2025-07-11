package config

import (
	"testing"

	"github.com/naoyafurudono/sqlc-use-analysis/pkg/types"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()
	
	if config == nil {
		t.Fatal("DefaultConfig() returned nil")
	}
	
	// 基本設定のテスト
	if config.RootPath != "." {
		t.Errorf("Expected RootPath to be '.', got '%s'", config.RootPath)
	}
	
	if config.OutputPath != "db_dependencies.json" {
		t.Errorf("Expected OutputPath to be 'db_dependencies.json', got '%s'", config.OutputPath)
	}
	
	// 除外パターンのテスト
	if len(config.Exclude) == 0 {
		t.Error("Expected exclude patterns to be set")
	}
	
	expectedExcludes := []string{
		"**/*_test.go",
		"**/testdata/**",
		"**/vendor/**",
		"**/.git/**",
	}
	
	for _, expected := range expectedExcludes {
		found := false
		for _, actual := range config.Exclude {
			if actual == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected exclude pattern '%s' not found", expected)
		}
	}
	
	// 解析設定のテスト
	if config.Analysis.IncludeTests {
		t.Error("Expected IncludeTests to be false by default")
	}
	
	if config.Analysis.SQLDialect != "postgresql" {
		t.Errorf("Expected SQLDialect to be 'postgresql', got '%s'", config.Analysis.SQLDialect)
	}
	
	// 出力設定のテスト
	if config.Output.Format != types.FormatJSON {
		t.Errorf("Expected Format to be JSON, got '%s'", config.Output.Format)
	}
	
	if !config.Output.Pretty {
		t.Error("Expected Pretty to be true by default")
	}
	
	// パフォーマンス設定のテスト
	if config.Performance.MaxWorkers <= 0 {
		t.Error("Expected MaxWorkers to be positive")
	}
	
	if config.Performance.MemoryLimit != 1024 {
		t.Errorf("Expected MemoryLimit to be 1024, got %d", config.Performance.MemoryLimit)
	}
}