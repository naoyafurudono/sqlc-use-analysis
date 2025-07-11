package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/naoyafurudono/sqlc-use-analysis/internal/config"
	"github.com/naoyafurudono/sqlc-use-analysis/internal/errors"
	"github.com/naoyafurudono/sqlc-use-analysis/internal/io"
	"github.com/naoyafurudono/sqlc-use-analysis/internal/orchestrator"
	"github.com/naoyafurudono/sqlc-use-analysis/pkg/types"
)

const (
	version = "dev"
	name    = "sqlc-analyzer"
)

func main() {
	if err := run(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

func run() error {
	ctx := context.Background()
	
	// エラーコレクターの初期化
	errorCollector := errors.NewErrorCollector(100, true)
	
	// 入力の読み込み
	inputReader := io.NewInputReader()
	request, err := inputReader.ReadRequest()
	if err != nil {
		return fmt.Errorf("failed to read request: %w", err)
	}
	
	// 設定の読み込み
	configLoader := config.NewConfigLoader()
	cfg, err := configLoader.LoadFromRequest(request)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}
	
	// オーケストレーターの初期化
	orch, err := orchestrator.New(cfg, errorCollector)
	if err != nil {
		return fmt.Errorf("failed to create orchestrator: %w", err)
	}
	
	// 解析の実行
	result, err := orch.Execute(ctx, request)
	if err != nil {
		return fmt.Errorf("failed to execute analysis: %w", err)
	}
	
	// 結果の出力
	outputWriter := io.NewOutputWriter(cfg)
	if err := outputWriter.WriteResult(result); err != nil {
		return fmt.Errorf("failed to write result: %w", err)
	}
	
	// sqlcプラグインレスポンスの生成
	responseWriter := io.NewResponseWriter()
	files := []*types.GeneratedFile{
		{
			Name:     ".sqlc_dependency_analysis",
			Contents: []byte("// Analysis completed successfully"),
		},
	}
	
	if err := responseWriter.WriteResponse(files); err != nil {
		return fmt.Errorf("failed to write response: %w", err)
	}
	
	return nil
}

func init() {
	// デバッグ情報の設定
	if os.Getenv("SQLC_ANALYZER_DEBUG") == "true" {
		log.SetFlags(log.LstdFlags | log.Lshortfile)
	}
}