package orchestrator

import (
	"context"
	"testing"
	"time"

	"github.com/naoyafurudono/sqlc-use-analysis/internal/config"
	"github.com/naoyafurudono/sqlc-use-analysis/internal/errors"
	"github.com/naoyafurudono/sqlc-use-analysis/pkg/types"
)

func TestNew(t *testing.T) {
	cfg := &types.Config{
		RootPath:   ".",
		OutputPath: "test.json",
	}
	errorCollector := errors.NewErrorCollector(10, false)
	
	orch, err := New(cfg, errorCollector)
	if err != nil {
		t.Errorf("New() error = %v", err)
	}
	
	if orch == nil {
		t.Error("New() returned nil orchestrator")
	}
	
	if orch.config != cfg {
		t.Error("Expected config to be set")
	}
	
	if orch.errorCollector != errorCollector {
		t.Error("Expected error collector to be set")
	}
}

func TestOrchestrator_Execute(t *testing.T) {
	cfg := &types.Config{
		RootPath:   ".",
		OutputPath: "test.json",
	}
	errorCollector := errors.NewErrorCollector(10, false)
	
	orch, err := New(cfg, errorCollector)
	if err != nil {
		t.Errorf("New() error = %v", err)
	}
	
	request := &config.CodeGeneratorRequest{
		Settings: make(map[string]interface{}),
		Queries:  []interface{}{},
	}
	
	ctx := context.Background()
	result, err := orch.Execute(ctx, request)
	
	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}
	
	if result == nil {
		t.Fatal("Execute() returned nil result")
	}
	
	// 基本的な結果の確認
	if result.Metadata.Version != "dev" {
		t.Errorf("Expected version to be 'dev', got '%s'", result.Metadata.Version)
	}
	
	if result.Metadata.GeneratedAt.IsZero() {
		t.Error("Expected GeneratedAt to be set")
	}
	
	if result.Metadata.AnalysisDuration <= 0 {
		t.Error("Expected AnalysisDuration to be positive")
	}
	
	if result.FunctionView == nil {
		t.Error("Expected FunctionView to be initialized")
	}
	
	if result.TableView == nil {
		t.Error("Expected TableView to be initialized")
	}
	
	// ダミーデータの確認（現在の実装）
	if len(result.FunctionView) == 0 {
		t.Error("Expected FunctionView to have dummy data")
	}
	
	if len(result.TableView) == 0 {
		t.Error("Expected TableView to have dummy data")
	}
}

func TestOrchestrator_Execute_WithContext(t *testing.T) {
	cfg := &types.Config{
		RootPath:   ".",
		OutputPath: "test.json",
	}
	errorCollector := errors.NewErrorCollector(10, false)
	
	orch, err := New(cfg, errorCollector)
	if err != nil {
		t.Errorf("New() error = %v", err)
	}
	
	request := &config.CodeGeneratorRequest{
		Settings: make(map[string]interface{}),
		Queries:  []interface{}{},
	}
	
	// タイムアウトを設定したコンテキスト
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	
	result, err := orch.Execute(ctx, request)
	
	if err != nil {
		t.Errorf("Execute() error = %v", err)
	}
	
	if result == nil {
		t.Fatal("Execute() returned nil result")
	}
}