package io

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/naoyafurudono/sqlc-use-analysis/pkg/types"
)

// OutputWriter writes analysis results to various formats
type OutputWriter struct {
	config *types.Config
}

// NewOutputWriter creates a new output writer
func NewOutputWriter(config *types.Config) *OutputWriter {
	return &OutputWriter{
		config: config,
	}
}

// WriteResult writes the analysis result to the configured output
func (ow *OutputWriter) WriteResult(result *types.DependencyResult) error {
	// メタデータの追加
	if result.Metadata.GeneratedAt.IsZero() {
		result.Metadata.GeneratedAt = time.Now().UTC()
	}
	if result.Metadata.Version == "" {
		result.Metadata.Version = "dev"
	}
	
	// 統計情報の更新
	result.Metadata.TotalFuncs = len(result.FunctionView)
	result.Metadata.TotalTables = len(result.TableView)
	
	// JSON生成
	var jsonBytes []byte
	var err error
	
	if ow.config.Output.Pretty {
		jsonBytes, err = json.MarshalIndent(result, "", "  ")
	} else {
		jsonBytes, err = json.Marshal(result)
	}
	
	if err != nil {
		return fmt.Errorf("failed to marshal output: %w", err)
	}
	
	// ファイルへの書き込み
	outputPath := ow.config.OutputPath
	if !filepath.IsAbs(outputPath) {
		outputPath = filepath.Join(ow.config.RootPath, outputPath)
	}
	
	if err := ow.ensureDir(outputPath); err != nil {
		return fmt.Errorf("failed to create output directory: %w", err)
	}
	
	if err := os.WriteFile(outputPath, jsonBytes, 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}
	
	return nil
}

func (ow *OutputWriter) ensureDir(filePath string) error {
	dir := filepath.Dir(filePath)
	return os.MkdirAll(dir, 0755)
}